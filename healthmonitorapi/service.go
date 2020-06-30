package main

import (
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"healthmonitor/healthmonitorapi/gateways"
	"healthmonitor/healthmonitorapi/usecases/jobs"
	"healthmonitor/healthmonitorapi/usecases/validation"
	"net/http"
)

const (
	healthPath     = "/healthmonitorapi/health"
	deviceInfoPath = "/healthmonitorapi/entities/devices/info"
	deviceDataPath = "/healthmonitorapi/entities/devices/data"

	alertsPath = "/healthmonitorapi/entities/devices/alerts"

	userDevicesPath       = "/healthmonitorapi/entities/users/devices"
	userSubscriptionsPath = "/healthmonitorapi/entities/users/subscriptions"

	registerPath = "/healthmonitorapi/auth/register"
	loginPath    = "/healthmonitorapi/auth/login"

	deviceReportsPath = "/healthmonitorapi/entities/devices/reports"
)

type HealthMonitorAPIService struct {
	UsersRepo     *gateways.UsersRepo
	DevicesRepo   *gateways.DevicesRepo
	MessagingRepo *gateways.MessagingRepo
	CronJobRunner *jobs.CronJobRunner
	APIHandler    *gateways.APIHandler
	config        *HealthMonitorAPIServiceConfig
	router        *mux.Router
}

func NewHealthMonitorAPIService(config *HealthMonitorAPIServiceConfig) *HealthMonitorAPIService {
	temperatureValidationBound := validation.NewValidationBound(config.TemperatureMinBound, config.TemperatureMaxBound)
	heartrateValidationBound := validation.NewValidationBound(config.HeartrateMinBound, config.HeartrateMaxBound)
	ecgValidationBound := validation.NewValidationBound(config.ECGMinBound, config.ECGMaxBound)
	spo2ValidationBound := validation.NewValidationBound(config.Spo2MinBound, config.Spo2MaxBound)
	minimalistValidator := validation.NewMinimalistValidator(temperatureValidationBound, heartrateValidationBound, ecgValidationBound, spo2ValidationBound)

	usersRepo := gateways.NewUsersRepo(config.CassandraHost)
	devicesRepo := gateways.NewDevicesRepo(config.ElasticsearchHost)
	messagingRepo := gateways.NewMessagingRepo(config.KafkaBrokers)
	cronJobRunner := jobs.NewCronJobRunner(messagingRepo, config.CronJobInterval, config.MaxDatapointAge)

	apiHandler := gateways.NewAPIHandler(usersRepo, devicesRepo, messagingRepo, minimalistValidator, config.PasswordSalt, config.ValidationInterval)

	service := &HealthMonitorAPIService{
		UsersRepo:     usersRepo,
		DevicesRepo:   devicesRepo,
		MessagingRepo: messagingRepo,
		APIHandler:    apiHandler,
		CronJobRunner: cronJobRunner,
		config:        config,
	}

	service.registerRoutes()

	return service
}
func (s *HealthMonitorAPIService) Start() error {
	err := s.UsersRepo.Start()
	if err != nil {
		return err
	}

	err = s.DevicesRepo.Start()
	if err != nil {
		return err
	}

	s.MessagingRepo.Start()

	allowedHeaders := handlers.AllowedHeaders([]string{"Authorization", "Accept", "Accept-Language", "Content-Type", "Content-Language", "Origin", "Access-Control-Allow-Origin"})
	allowedMethods := handlers.AllowedMethods([]string{"POST", "GET", "DELETE", "PUT", "OPTIONS"})
	err = http.ListenAndServe(":"+s.config.Port, handlers.CORS(allowedMethods, allowedHeaders)(s.router))
	if err != nil {
		return err
	}

	return nil
}

func (s *HealthMonitorAPIService) registerRoutes() {
	router := mux.NewRouter()

	router.HandleFunc(healthPath, s.APIHandler.GetHealth).Methods("GET")

	router.HandleFunc(deviceInfoPath, s.APIHandler.GetDeviceInfo).Methods("GET")
	router.HandleFunc(deviceInfoPath, s.APIHandler.RegisterDeviceInfo).Methods("POST")
	router.HandleFunc(deviceInfoPath, s.APIHandler.UpdateDeviceInfo).Methods("PUT")

	router.HandleFunc(deviceDataPath, s.APIHandler.GetDeviceData).Methods("GET")
	router.HandleFunc(deviceDataPath, s.APIHandler.RegisterDeviceData).Methods("POST")

	router.HandleFunc(alertsPath, s.APIHandler.GetAlerts).Methods("GET")

	router.HandleFunc(userDevicesPath, s.APIHandler.GetDevices).Methods("GET")
	router.HandleFunc(userDevicesPath, s.APIHandler.AddDevices).Methods("POST")
	router.HandleFunc(userDevicesPath, s.APIHandler.DeleteDevices).Methods("DELETE")

	router.HandleFunc(userSubscriptionsPath, s.APIHandler.AddSubscription).Methods("POST")
	router.HandleFunc(userSubscriptionsPath, s.APIHandler.DeleteSubscription).Methods("DELETE")

	router.HandleFunc(registerPath, s.APIHandler.RegisterUser).Methods("POST")
	router.HandleFunc(loginPath, s.APIHandler.LoginUser).Methods("POST")

	router.HandleFunc(deviceReportsPath, s.APIHandler.StartReportGeneration).Methods("POST")

	s.router = router
}
