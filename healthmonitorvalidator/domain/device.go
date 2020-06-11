package domain

type DeviceInfoES struct {
	DID               string `json:"did"`
	LastSeenTimestamp string `json:"last_seen_timestamp"`
}

type DeviceInfo struct {
	DID               string `json:"did"`
	LastSeenTimestamp int64  `json:"last_seen_timestamp"`
}

type DeviceDataES struct {
	DID         string  `json:"did"`
	Temperature float32 `json:"temperature"`
	Heartrate   int64   `json:"heart_rate"`
	Timestamp   string  `json:"timestamp"`
}
type DeviceData struct {
	Temperature float32 `json:"temperature"`
	Heartrate   int64   `json:"heart_rate"`
	Timestamp   int64   `json:"timestamp"`
}

type DeviceDataset struct {
	DID  string        `json:"did"`
	Data []*DeviceData `json:"data"`
}
