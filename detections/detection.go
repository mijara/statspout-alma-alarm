package detections

import "time"

type Resource string

const (
	MEM = Resource("mem")
	CPU = Resource("cpu")
)

type Detection struct {
	Resource      Resource
	Timestamp     time.Time
	ContainerName string
	Message       string
}
