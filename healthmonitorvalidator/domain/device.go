package domain

import "time"

type DeviceInfoES struct {
	DID                     string   `json:"did"`
	LastSeenTimestamp       string   `json:"last_seen_timestamp"`
	LastValidationTimestamp string   `json:"last_validation_timestamp"`
	PatientName             string   `json:"patient_name"`
	SubscribedPhones        []string `json:"subscribed_phones"`
}

type DeviceInfo struct {
	DID                     string    `json:"did"`
	LastSeenTimestamp       time.Time `json:"last_seen_timestamp"`
	LastValidationTimestamp time.Time `json:"last_validation_timestamp"`
	PatientName             string    `json:"patient_name"`
	SubscribedPhones        []string  `json:"subscribed_phones"`
}
type DeviceDataES struct {
	DID         string  `json:"did"`
	Temperature float64 `json:"temperature"`
	Heartrate   float64 `json:"heart_rate"`
	ECG         float64 `json:"heart_ecg"`
	SPO2        float64 `json:"spo2"`
	Timestamp   string  `json:"timestamp"`
}

type DeviceData struct {
	DID         string    `json:"did"`
	Temperature float64   `json:"temperature"`
	Heartrate   float64   `json:"heart_rate"`
	ECG         float64   `json:"heart_ecg"`
	SPO2        float64   `json:"spo2"`
	Timestamp   time.Time `json:"timestamp"`
}

type DeviceDataset struct {
	DID  string        `json:"did"`
	Data []*DeviceData `json:"data"`
}
