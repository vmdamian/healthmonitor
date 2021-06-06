package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"time"
)

func main() {
	ctx := context.Background()

	es := flag.String("elasticsearch", "http://127.0.0.1:9200", "elasticsearch host")
	kafka := flag.String("kafka", "127.0.0.1:9092", "kafka host")
	kafkaWorkerCount := flag.Int("kafkaWorkers", 1, "kafka thread pool size")
	flag.Parse()

	config := &HealthMonitorValidatorServiceConfig{
		KafkaBrokers:           []string{*kafka},
		ElasticsearchHost:      *es,
		KafkaWorkerCount:       *kafkaWorkerCount,
		AlertSenderAccountSID:  "ACba5d5938272686a1ccf9a7a10211d209",
		AlertSenderToken:       "5ae99d73318207c2113d6dea71abd7a4",
		AlertSenderPhoneNumber: "+12024997424",
		ValidationPeriod:       time.Hour,
		SendCreatedAlert:       true,
		SendContinuedAlert:     false,
		SendResolvedAlert:      false,
	}
	service := NewHealthMonitorValidatorService(config)

	err := service.Start(ctx)
	if err != nil {
		log.WithError(err).Fatalln("failed to start HealthMonitorValidatorService")
	} else {
		log.Infoln("successfully started HeathMonitorValidatorService")
	}
}
