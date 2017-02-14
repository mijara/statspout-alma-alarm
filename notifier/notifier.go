package notifier

import (
	"github.com/mijara/statspoutalarm/detections"
)

// Notifier represent an entity that is able to notify using some engine.
type Notifier interface {
	Notify(detections []*detections.Detection)
	Close()
}
