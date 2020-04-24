package main

import log "github.com/sirupsen/logrus"

func main() {
	config := &HealthMonitorAPIServiceConfig{
		Port: "9000",
		PasswordSalt: "720036c8101f751b82cdba6e74fbd217419a2d478dd49f6d7ba6697ed3810ece",
		CassandraHost: "127.0.0.1",
	}
	service := NewHealthMonitorAPIService(config)
	err := service.Start()
	if err != nil {
		log.WithError(err).Fatalln("failed to start HealthMonitorAPIService")
	} else {
		log.Infoln("successfully started HeathMonitorAPIService")
	}
}