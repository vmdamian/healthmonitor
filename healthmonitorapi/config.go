package main

import "time"

type HealthMonitorAPIServiceConfig struct {
	Port               string
	PasswordSalt       string
	ValidationInterval time.Duration
	HealthMonitorDependenciesConfig
	HealthMonitorBoundsConfig
	HealthMonitorCleanupConfig
}

type HealthMonitorDependenciesConfig struct {
	CassandraHost     string
	ElasticsearchHost string
	KafkaBrokers      []string
}

type HealthMonitorBoundsConfig struct {
	TemperatureMinBound float64
	TemperatureMaxBound float64
	HeartrateMinBound   float64
	HeartrateMaxBound   float64
	ECGMinBound         float64
	ECGMaxBound         float64
	Spo2MinBound        float64
	Spo2MaxBound        float64
}

type HealthMonitorCleanupConfig struct {
	CronJobInterval time.Duration
	MaxDatapointAge time.Duration
}
