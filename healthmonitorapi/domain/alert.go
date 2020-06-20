package domain

import "time"

type Alert struct {
	DID                 string    `json:"did"`
	AlertType           string    `json:"alert_type"`
	Status              string    `json:"status"`
	CreatedTimestamp    time.Time `json:"created_timestamp"`
	LastActiveTimestamp time.Time `json:"last_active_timestamp"`
	ResolvedTimestamp   time.Time `json:"resolved_timestamp"`
}

type AlertES struct {
	DID                 string `json:"did"`
	AlertType           string `json:"alert_type"`
	Status              string `json:"status"`
	CreatedTimestamp    string `json:"created_timestamp"`
	LastActiveTimestamp string `json:"last_active_timestamp"`
	ResolvedTimestamp   string `json:"resolved_timestamp"`
}
