package domain

import "time"

type DataPoint struct {
	timestamp time.Time
	value     float64
}

type Validator interface {
	CheckData(*DeviceDataset) []*Alert
}
