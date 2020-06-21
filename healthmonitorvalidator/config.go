package main

import "time"

type HealthMonitorValidatorServiceConfig struct {
	KafkaBrokers      []string
	ElasticsearchHost string

	KafkaWorkerCount int

	AlertSenderAccountSID  string
	AlertSenderToken       string
	AlertSenderPhoneNumber string
	ValidationPeriod       time.Duration

	SendCreatedAlert   bool
	SendContinuedAlert bool
	SendResolvedAlert  bool
}
