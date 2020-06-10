package main

import (
	"context"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	config := &HealthMonitorValidatorServiceConfig{
		KafkaBrokers: []string{"127.0.0.1:9092"},
	}
	service := NewHealthMonitorValidatorService(config)

	err := service.Start(ctx)
	if err != nil {
		log.WithError(err).Fatalln("failed to start HealthMonitorValidatorService")
	} else {
		log.Infoln("successfully started HeathMonitorValidatorService")
	}
}