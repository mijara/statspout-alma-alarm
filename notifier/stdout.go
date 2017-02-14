package notifier

import (
	"github.com/mijara/statspout/log"
	"github.com/mijara/statspoutalarm/detections"
)

// Stdout is the basic notifier that outputs alarm to Standard output.
type Stdout struct {
}

// NewStdout creates a new instance of the Stdout notifier.
func NewStdout() *Stdout {
	return &Stdout{}
}

// Notify will print each message.
func (s *Stdout) Notify(detections []*detections.Detection) {
	for _, d := range detections {
		log.Warning.Println(d.Message)
	}
}

// Close will do nothing.
func (s *Stdout) Close() {
}
