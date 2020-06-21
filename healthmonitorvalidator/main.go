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
		AlertSenderAccountSID:  "AC3bc32ab97a438ceee53a2fe0bd873d7a",
		AlertSenderToken:       "bdefff5e6f7894ae2b30f33fe654139a",
		AlertSenderPhoneNumber: "+12055519501",
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
