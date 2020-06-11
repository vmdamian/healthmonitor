package domain

import "time"

const (
	ALERT_TYPE_HEARTRATE_HIGH = "HEARTRATE_HIGH"
	ALERT_TYPE_HEARTRATE_LOW = "HEARTRATE_LOW"

	ALERT_TYPE_TEMP_HIGH = "TEMPERATURE_HIGH"
	ALERT_TYPE_TEMP_LOW = "TEMPERATURE_LOW"

	ALERT_TYPE_ECG_HIGH = "ECG_HIGH"
	ALERT_TYPE_ECG_LOW = "ECG_LOW"

	ALERT_TYPE_SPO2_HIGH = "SP02_HIGH"
	ALERT_TYPE_SP02_LOW = "SP02_LOW"

	ALERT_STATUS_ACTIVE = "ACTIVE"
	ALERT_STATUS_RESOLVED = "RESOLVED"

	ALERT_UPDATE_TYPE_RESOLVED = "RESOLVED"
	ALERT_UPDATE_TYPE_CREATED = "CREATED"
	ALERT_UPDATE_TYPE_CONTINUED = "CONTINUED"
)

type Alert struct {
	DID string `json:"did"`
	AlertType string `json:"alert_type"`
	Status string `json:"status"`
	CreatedTimestamp time.Time `json:"created_timestamp"`
	LastActiveTimestamp time.Time `json:"last_active_timestamp"`
	ResolvedTimestamp time.Time `json:"resolved_timestamp"`
}

type AlertUpdate struct {
	UpdateType string
	DocID string
	Alert *Alert
}

type AlertES struct {
	DID string `json:"did"`
	AlertType string `json:"alert_type"`
	Status string `json:"status"`
	CreatedTimestamp string `json:"created_timestamp"`
	LastActiveTimestamp string `json:"last_active_timestamp"`
	ResolvedTimestamp string `json:"resolved_timestamp"`
}