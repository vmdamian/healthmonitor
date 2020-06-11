package main

type HealthMonitorAPIServiceConfig struct {
	Port              string
	PasswordSalt      string
	CassandraHost     string
	ElasticsearchHost string
	KafkaBrokers      []string
}
