package main

import (
	"github.com/mijara/statspout"
	"github.com/mijara/statspout/common"
	"github.com/mijara/statspout/opts"
	"github.com/mijara/statspoutalarm/repo"
)

func main() {
	cfg := opts.NewConfig()

	cfg.AddRepository(&common.Stdout{}, nil)
	cfg.AddRepository(&repo.AlarmDetector{}, repo.CreateAlarmDetectorOpts())

	statspout.Start(cfg)
}
