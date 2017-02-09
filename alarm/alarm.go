package alarm

import (
	"fmt"
	"strconv"

	"github.com/mijara/statspout/stats"
	"github.com/mijara/statspout/repo"
	"github.com/mijara/statspout/common"
	"github.com/mijara/statspout/log"
	"github.com/mijara/statspoutalarm/notifier"
)

type Trigger struct {
	CPU bool
	MEM bool
}

type AlarmDetector struct {
	influx *common.InfluxDB
	triggered map[string]*Trigger
}

type AlarmDetectorOpts struct {
	*common.InfluxOpts
}

func NewAlarmDetector(opts *AlarmDetectorOpts) (*AlarmDetector, error) {
	ad := &AlarmDetector{}
	influx, err := common.NewInfluxDB(opts.InfluxOpts)
	if err != nil {
		return nil, err
	}
	ad.influx = influx
	ad.triggered = make(map[string]*Trigger)
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
	triggered := ad.GetTrigger(stats.Name)

	// only raise warning if the CPU is exceeded and it is not triggered at the moment.
	if cpuExceeded && !triggered.CPU {
		messages = append(messages,
			fmt.Sprintf("Max %f%% CPU exceeded! :: ", cpuMax) + stats.String())
		triggered.CPU = true
	} else {
		triggered.CPU = false
	}

	// only raise warning if the MEM is exceeded and it is not triggered at the moment.
	if memExceeded && !triggered.MEM {
		messages = append(messages,
			fmt.Sprintf("Max %f%% MEM exceeded! :: ", memMax) + stats.String())
		triggered.MEM = true
	} else {
		triggered.MEM = false
	}

	// if there's any message, send the batch of notifications.
	if len(messages) > 0 {
		for _, message := range messages {
			log.Warning.Println(message)
		}

		go notifier.Email(messages)
	}

	return nil
}

func (ad *AlarmDetector) GetTrigger(name string) *Trigger {
	trigger, ok := ad.triggered[name]
	if !ok {
		ad.triggered[name] = &Trigger{CPU: false, MEM: false}
		trigger = ad.triggered[name]
	}

	return trigger
}

func CreateAlarmDetectorOpts() *AlarmDetectorOpts {
	o := &AlarmDetectorOpts{
		InfluxOpts: common.CreateInfluxDBOpts(),
	}
	return o
}
