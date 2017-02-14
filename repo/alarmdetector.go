package repo

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/mijara/statspout/common"
	"github.com/mijara/statspout/log"
	"github.com/mijara/statspout/repo"
	"github.com/mijara/statspout/stats"
	"github.com/mijara/statspoutalarm/notifier"
)

// Trigger represents a state of a container, whether it's alarm has been triggered or not.
type Trigger struct {
	// CPU tells us if the cpu alarm has been triggered.
	CPU bool

	// MEM tells us if the mem alarm has been triggered.
	MEM bool
}

// AlarmDetector is a repository that raises alarms if certain maximums are exceeded. It will
// bypass all stats to InfluxDB.
type AlarmDetector struct {
	influx    *common.InfluxDB
	triggered map[string]*Trigger
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
}

// NewAlarmDetector will create a new instance of the AlarmDetector, and will initialize InfluxDB
// aswell. Use the options to set which notifiers will be used.
func NewAlarmDetector(opts *AlarmDetectorOpts) (*AlarmDetector, error) {
	influx, err := common.NewInfluxDB(opts.InfluxOpts)
	if err != nil {
		return nil, err
	}

	ad := &AlarmDetector{
		notifiers: make([]notifier.Notifier, 0),
		influx:    influx,
		triggered: make(map[string]*Trigger),
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

	for _, notifier := range ad.notifiers {
		notifier.Close()
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
	messages := make([]string, 0)

	// this object tells us if the warning was already raised for each resource.
	triggered := ad.GetTrigger(stats.Name)

	// only raise warning if the CPU is exceeded and it is not triggered at the moment.
	if cpuExceeded && !triggered.CPU {
		messages = append(messages, fmt.Sprintf("Max %f%% CPU exceeded: ", cpuMax)+stats.String())
		triggered.CPU = true
	} else {
		triggered.CPU = false
	}

	// only raise warning if the MEM is exceeded and it is not triggered at the moment.
	if memExceeded && !triggered.MEM {
		messages = append(messages, fmt.Sprintf("Max %f%% MEM exceeded: ", memMax)+stats.String())
		triggered.MEM = true
	} else {
		triggered.MEM = false
	}

	// if there's any message, send the batch of notifications.
	if len(messages) > 0 {
		go ad.notifyAll(messages)
	}

	return nil
}

// GetTrigger will get or create a trigger of some container.
func (ad *AlarmDetector) GetTrigger(name string) *Trigger {
	trigger, ok := ad.triggered[name]
	if !ok {
		ad.triggered[name] = &Trigger{CPU: false, MEM: false}
		trigger = ad.triggered[name]
	}

	return trigger
}

func (ad *AlarmDetector) notifyAll(messages []string) {
	for _, notifier := range ad.notifiers {
		if notifier != nil {
			notifier.Notify(messages)
		}
	}
}

// CreateAlarmDetectorOpts will return the options object for the alarm detector.
func CreateAlarmDetectorOpts() *AlarmDetectorOpts {
	o := &AlarmDetectorOpts{
		InfluxOpts: common.CreateInfluxDBOpts(),
	}

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
