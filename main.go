package main

import (
	"github.com/mijara/statspout/opts"
	"github.com/mijara/statspout/common"
	"github.com/mijara/statspout"
	"github.com/mijara/statspoutalarm/alarm"
)

func main() {
	cfg := opts.NewConfig()

	cfg.AddRepository(&common.Stdout{}, nil)
	cfg.AddRepository(&alarm.AlarmDetector{}, alarm.CreateAlarmDetectorOpts())

	statspout.Start(cfg)
}