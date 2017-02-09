package alarm

import (
	"fmt"
	"strconv"

	"github.com/mijara/statspout/stats"
	"github.com/mijara/statspout/repo"
	"github.com/mijara/statspout/common"
	"github.com/mijara/statspout/log"
	"github.com/mijara/statspoutalarm/notifier"
	"flag"
)

type Cooldown struct {
	CPU int
	MEM int
}

type AlarmDetector struct {
	influx *common.InfluxDB
	triggered map[string]*Cooldown
	cooldownCycles int
}

type AlarmDetectorOpts struct {
	*common.InfluxOpts
	cooldownCycles int
}

func NewAlarmDetector(opts *AlarmDetectorOpts) (*AlarmDetector, error) {
	ad := &AlarmDetector{
		cooldownCycles: opts.cooldownCycles,
	}
	influx, err := common.NewInfluxDB(opts.InfluxOpts)
	if err != nil {
		return nil, err
	}
	ad.influx = influx
	ad.triggered = make(map[string]*Cooldown)
	return ad, nil
}

func (ad *AlarmDetector) Clear(name string) {

}

func (ad *AlarmDetector) Close() {

}

func (ad *AlarmDetector) Create(v interface{}) (repo.Interface, error) {
	return NewAlarmDetector(v.(*AlarmDetectorOpts))
}

func (ad *AlarmDetector) Name() string {
	return "alarm"
}

func (ad *AlarmDetector) Push(stats *stats.Stats) error {
	// first make sure the stats arrive to the database.
	ad.influx.Push(stats)

	// then check metadata, if some of them are not specified, the value of max will be zero, and must be ignored.
	cpuMax, _ := strconv.ParseFloat(stats.Labels["cl.alma.max-cpu"], 32)
	memMax, _ := strconv.ParseFloat(stats.Labels["cl.alma.max-mem"], 32)

	// booleans that represent if one of the resources is exceeded.
	cpuExceeded := cpuMax > 0 && stats.CpuPercent > cpuMax
	memExceeded := memMax > 0 && stats.MemoryPercent > memMax

	// current batch of messages.
	messages := make([]string, 0)

	// this object tells us if the warning was already raised for each resource.
	triggered := ad.GetCooldown(stats.Name)

	// only raise warning if the CPU is exceeded and it is not triggered at the moment.
	if cpuExceeded && triggered.CPU <= 0 {
		messages = append(messages,
			fmt.Sprintf("Max %f%% CPU exceeded! :: ", cpuMax) + stats.String())
		triggered.CPU = ad.cooldownCycles
	} else if triggered.CPU > 0 {
		triggered.CPU--
	}

	// only raise warning if the MEM is exceeded and it is not triggered at the moment.
	if memExceeded && triggered.MEM <= 0 {
		messages = append(messages,
			fmt.Sprintf("Max %f%% MEM exceeded! :: ", memMax) + stats.String())
		triggered.MEM = ad.cooldownCycles
	} else if triggered.MEM > 0 {
		triggered.MEM--
	}

	fmt.Println(triggered)

	// if there's any message, send the batch of notifications.
	if len(messages) > 0 {
		for _, message := range messages {
			log.Warning.Println(message)
		}

		go notifier.Email(messages)
	}

	return nil
}

func (ad *AlarmDetector) GetCooldown(name string) *Cooldown {
	trigger, ok := ad.triggered[name]
	if !ok {
		ad.triggered[name] = &Cooldown{CPU: 0, MEM: 0}
		trigger = ad.triggered[name]
	}

	return trigger
}

func CreateAlarmDetectorOpts() *AlarmDetectorOpts {
	o := &AlarmDetectorOpts{
		InfluxOpts: common.CreateInfluxDBOpts(),
	}

	flag.IntVar(&o.cooldownCycles,
		"alarm.cycles",
		5,
		"Container push cycles to skip after a warning.",
	)

	return o
}
