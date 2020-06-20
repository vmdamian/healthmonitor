package main

import (
	log "github.com/sirupsen/logrus"
	"time"
)

func main() {
	boundsConfig := HealthMonitorBoundsConfig{
		TemperatureMinBound: 34,
		TemperatureMaxBound: 38,
		HeartrateMinBound:   60,
		HeartrateMaxBound:   100,
		ECGMinBound:         0,
		ECGMaxBound:         1000,
		Spo2MinBound:        80,
		Spo2MaxBound:        100,
	}

	cleanupConfig := HealthMonitorCleanupConfig{
		CronJobInterval: 24 * time.Hour,
		MaxDatapointAge: 7 * 24 * time.Hour,
	}

	dependenciesConfig := HealthMonitorDependenciesConfig{
		CassandraHost:     "127.0.0.1",
		ElasticsearchHost: "http://127.0.0.1:9200",
		KafkaBrokers:      []string{"127.0.0.1:9092"},
	}

	config := &HealthMonitorAPIServiceConfig{
		Port:                            "9000",
		PasswordSalt:                    "720036c8101f751b82cdba6e74fbd217419a2d478dd49f6d7ba6697ed3810ece",
		HealthMonitorBoundsConfig:       boundsConfig,
		HealthMonitorCleanupConfig:      cleanupConfig,
		HealthMonitorDependenciesConfig: dependenciesConfig,
	}

	service := NewHealthMonitorAPIService(config)
	err := service.Start()
	if err != nil {
		log.WithError(err).Fatalln("failed to start HealthMonitorAPIService")
	}
}
