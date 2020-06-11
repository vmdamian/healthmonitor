package main

import "time"

type HealthMonitorValidatorServiceConfig struct {
	KafkaBrokers      []string
	ElasticsearchHost string

	AlertSenderAccountSID  string
	AlertSenderToken       string
	AlertSenderPhoneNumber string
	ValidationPeriod       time.Duration

	SendCreatedAlert   bool
	SendContinuedAlert bool
	SendResolvedAlert  bool
}
