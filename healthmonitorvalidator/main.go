package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"time"
)

func main() {
	ctx := context.Background()

	config := &HealthMonitorValidatorServiceConfig{
		KafkaBrokers: []string{"127.0.0.1:9092"},
		ElasticsearchHost: "http://127.0.0.1:9200",
		AlertSenderAccountSID: "AC3bc32ab97a438ceee53a2fe0bd873d7a",
		AlertSenderToken: "bdefff5e6f7894ae2b30f33fe654139a",
		AlertSenderPhoneNumber: "+12055519501",
		ValidationPeriod: time.Hour,
		SendCreatedAlert: true,
		SendContinuedAlert: false,
		SendResolvedAlert: false,
	}
	service := NewHealthMonitorValidatorService(config)

	err := service.Start(ctx)
	if err != nil {
		log.WithError(err).Fatalln("failed to start HealthMonitorValidatorService")
	} else {
		log.Infoln("successfully started HeathMonitorValidatorService")
	}
}