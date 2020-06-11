package main

import (
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"healthmonitor/healthmonitorapi/gateways"
	"net/http"
)

const (
	DeviceInfoPath = "/healthmonitorapi/entities/devices/info"
	DeviceDataPath = "/healthmonitorapi/entities/devices/data"

	registerPath = "/healthmonitorapi/auth/register"
	loginPath    = "/healthmonitorapi/auth/login"

	userDevicesPath = "/healthmonitorapi/entities/users/devices"
)

type HealthMonitorAPIService struct {
	UsersRepo     *gateways.UsersRepo
	DevicesRepo   *gateways.DevicesRepo
	MessagingRepo *gateways.MessagingRepo
	APIHandler    *gateways.APIHandler
	config        *HealthMonitorAPIServiceConfig
	router        *mux.Router
}

func NewHealthMonitorAPIService(config *HealthMonitorAPIServiceConfig) *HealthMonitorAPIService {
	usersRepo := gateways.NewUsersRepo(config.CassandraHost)
	devicesRepo := gateways.NewDevicesRepo(config.ElasticsearchHost)
	messagingRepo := gateways.NewMessagingRepo(config.KafkaBrokers)
	apiHandler := gateways.NewAPIHandler(usersRepo, devicesRepo, messagingRepo, config.PasswordSalt)

	service := &HealthMonitorAPIService{
		UsersRepo:     usersRepo,
		DevicesRepo:   devicesRepo,
		MessagingRepo: messagingRepo,
		APIHandler:    apiHandler,
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

	allowedHeaders := handlers.AllowedHeaders([]string{"Authorization", "Accept", "Accept-Language", "Content-Type", "Content-Language", "Origin"})

	err = http.ListenAndServe(":"+s.config.Port, handlers.CORS(allowedHeaders)(s.router))
	if err != nil {
		return err
	}

	return nil
}

func (s *HealthMonitorAPIService) registerRoutes() {
	router := mux.NewRouter()

	router.HandleFunc(DeviceInfoPath, s.APIHandler.GetDeviceInfo).Methods("GET")
	router.HandleFunc(DeviceDataPath, s.APIHandler.GetDeviceData).Methods("GET")
	router.HandleFunc(DeviceInfoPath, s.APIHandler.RegisterDeviceInfo).Methods("POST")
	router.HandleFunc(DeviceDataPath, s.APIHandler.RegisterDeviceData).Methods("POST")

	router.HandleFunc(registerPath, s.APIHandler.RegisterUser).Methods("POST")
	router.HandleFunc(loginPath, s.APIHandler.LoginUser).Methods("POST")

	router.HandleFunc(userDevicesPath, s.APIHandler.AddDevices).Methods("POST")
	router.HandleFunc(userDevicesPath, s.APIHandler.GetDevices).Methods("GET")

	s.router = router
}
