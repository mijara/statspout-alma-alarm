package repo

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/mijara/statspout/common"
	"github.com/mijara/statspout/log"
	"github.com/mijara/statspout/repo"
	"github.com/mijara/statspout/stats"
	"github.com/mijara/statspoutalarm/detections"
	"github.com/mijara/statspoutalarm/notifier"
)

type Cooldown struct {
	CPU int
	MEM int
}

// AlarmDetector is a repository that raises alarms if certain maximums are exceeded. It will
// bypass all stats to InfluxDB.
type AlarmDetector struct {
	influx *common.InfluxDB

	cooldown       map[string]*Cooldown
	cooldownCycles int

	notifiers []notifier.Notifier
}

// AlarmDetectorOpts represents options for the alarms and InfluxDB.
type AlarmDetectorOpts struct {
	*common.InfluxOpts

	RabbitMQ struct {
		Enabled bool
		URI     string
		Queue   string
	}

	Stdout bool

	CooldownCycles int
}

// NewAlarmDetector will create a new instance of the AlarmDetector, and will initialize InfluxDB
// aswell. Use the options to set which notifiers will be used.
func NewAlarmDetector(opts *AlarmDetectorOpts) (*AlarmDetector, error) {
	influx, err := common.NewInfluxDB(opts.InfluxOpts)
	if err != nil {
		return nil, err
	}

	ad := &AlarmDetector{
		notifiers:      make([]notifier.Notifier, 0),
		influx:         influx,
		cooldown:       make(map[string]*Cooldown),
		cooldownCycles: opts.CooldownCycles,
	}

	if opts.RabbitMQ.Enabled {
		rabbitMQ := notifier.NewRabbitMQ(opts.RabbitMQ.URI, opts.RabbitMQ.Queue)
		if rabbitMQ != nil {
			log.Info.Println("AlarmDetector: RabbitMQ notifier enabled.")
			ad.notifiers = append(ad.notifiers, rabbitMQ)
		}
	}

	if opts.Stdout {
		ad.notifiers = append(ad.notifiers, notifier.NewStdout())
		log.Info.Println("AlarmDetector: Stdout notifier enabled.")
	}

	return ad, nil
}

// Clear will bypass to InfluxDB's Clear method.
func (ad *AlarmDetector) Clear(name string) {
	ad.influx.Clear(name)
}

// Close will bypass to InfluxDB's Close method and close all notifiers.
func (ad *AlarmDetector) Close() {
	ad.influx.Close()

	for _, n := range ad.notifiers {
		n.Close()
	}
}

// Create will return a new instance of the detector.
func (ad *AlarmDetector) Create(v interface{}) (repo.Interface, error) {
	return NewAlarmDetector(v.(*AlarmDetectorOpts))
}

// Name returns the command line name for the alarm detector (alarm).
func (ad *AlarmDetector) Name() string {
	return "alarm"
}

// Push will send stats to InfluxDB and inspect them to find if CPU or MEM maximums are exceeded.
func (ad *AlarmDetector) Push(stats *stats.Stats) error {
	// first make sure the stats arrive to the database.
	ad.influx.Push(stats)

	// then check metadata, if some of them are not specified, the value of max will be zero, and
	// must be ignored.
	cpuMax, _ := strconv.ParseFloat(stats.Labels["cl.alma.max-cpu"], 32)
	memMax, _ := strconv.ParseFloat(stats.Labels["cl.alma.max-mem"], 32)

	// booleans that represent if one of the resources is exceeded.
	cpuExceeded := cpuMax > 0 && stats.CpuPercent > cpuMax
	memExceeded := memMax > 0 && stats.MemoryPercent > memMax

	// current batch of messages.
	detectionList := make([]*detections.Detection, 0)

	// this object tells us if the warning was already raised for each resource.
	cooldown := ad.GetCooldown(stats.Name)

	if cpuExceeded {
		// only raise warning if CPU is exceeded and the cooldown is 0.
		if cooldown.CPU <= 0 {
			message := fmt.Sprintf("Max %.2f%% CPU exceeded: ", cpuMax) + stats.String()
			detectionList = append(detectionList, &detections.Detection{
				Resource:      detections.CPU,
				Timestamp:     stats.Timestamp,
				ContainerName: stats.Name,
				Message:       message,
			})
		}
		cooldown.CPU = ad.cooldownCycles
	} else {
		if cooldown.CPU == 1 {
			log.Debug.Printf("%s container CPU is normal again: %.2f%%", stats.Name, stats.CpuPercent)
		}

		cooldown.CPU--
	}

	if memExceeded {
		// only raise warning if MEM is exceeded and the cooldown is 0.
		if cooldown.MEM <= 0 {
			message := fmt.Sprintf("Max %.2f%% MEM exceeded: ", memMax) + stats.String()
			detectionList = append(detectionList, &detections.Detection{
				Resource:      detections.MEM,
				Timestamp:     stats.Timestamp,
				ContainerName: stats.Name,
				Message:       message,
			})
		}
		cooldown.MEM = ad.cooldownCycles
	} else {
		if cooldown.MEM == 1 {
			log.Debug.Printf("%s container MEM is normal again: %.2f%%", stats.Name, stats.MemoryPercent)
		}

		cooldown.MEM--
	}

	// if there's any message, send the batch of notifications.
	if len(detectionList) > 0 {
		go ad.notifyAll(detectionList)
	}

	return nil
}

// GetTrigger will get or create a trigger of some container.
func (ad *AlarmDetector) GetCooldown(name string) *Cooldown {
	cooldown, ok := ad.cooldown[name]
	if !ok {
		ad.cooldown[name] = &Cooldown{CPU: 0, MEM: 0}
		cooldown = ad.cooldown[name]
	}

	return cooldown
}

func (ad *AlarmDetector) notifyAll(detections []*detections.Detection) {
	for _, n := range ad.notifiers {
		if n != nil {
			n.Notify(detections)
		}
	}
}

// CreateAlarmDetectorOpts will return the options object for the alarm detector.
func CreateAlarmDetectorOpts() *AlarmDetectorOpts {
	o := &AlarmDetectorOpts{
		InfluxOpts: common.CreateInfluxDBOpts(),
	}

	flag.IntVar(&o.CooldownCycles,
		"alarm.cycles",
		10,
		"Cycles of cooldown after a the detection stopped.")

	flag.BoolVar(&o.Stdout,
		"alarm.stdout",
		true,
		"Enable or disable the Stdout notifier.")

	flag.BoolVar(&o.RabbitMQ.Enabled,
		"alarm.rabbitmq",
		false,
		"Enable or disable the RabbitMQ notifier.")

	flag.StringVar(&o.RabbitMQ.URI,
		"alarm.rabbitmq.uri",
		"amqp://localhost:5672/",
		"Broker URI. See https://www.rabbitmq.com/uri-spec.html")

	flag.StringVar(&o.RabbitMQ.Queue,
		"alarm.rabbitmq.queue",
		"alarms",
		"Queue for alarms raised.")

	return o
}
