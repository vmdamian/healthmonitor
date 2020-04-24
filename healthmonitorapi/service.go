package main

import (
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"healthmonitor/healthmonitorapi/gateways"
	"net/http"
)

const (
	getDeviceInfoPath = "/healthmonitorapi/entities/devices/info"
	getDeviceDataPath = "/healthmonitorapi/entities/devices/data"

	registerPath = "/healthmonitorapi/auth/register"
	loginPath = "/healthmonitorapi/auth/login"
)

type HealthMonitorAPIService struct {
	UsersRepo *gateways.UsersRepo
	DevicesRepo *gateways.DevicesRepo
	APIHandler *gateways.APIHandler
	config *HealthMonitorAPIServiceConfig
	router *mux.Router
}

func NewHealthMonitorAPIService(config *HealthMonitorAPIServiceConfig) *HealthMonitorAPIService {
	usersRepo := gateways.NewUsersRepo(config.CassandraHost)
	devicesRepo := gateways.NewDevicesRepo()
	apiHandler := gateways.NewAPIHandler(usersRepo, devicesRepo, config.PasswordSalt)

	service := &HealthMonitorAPIService{
		UsersRepo: usersRepo,
		DevicesRepo: devicesRepo,
		APIHandler: apiHandler,
		config: config,
	}

	service.registerRoutes()

	return service
}
func (s *HealthMonitorAPIService) Start() error {
	err := s.UsersRepo.Start()
	if err != nil {
		return err
	}

	allowedHeaders := handlers.AllowedHeaders([]string{"Accept", "Accept-Language", "Content-Type", "Content-Language", "Origin"})

	err = http.ListenAndServe(":" + s.config.Port, handlers.CORS(allowedHeaders)(s.router))
	if err != nil {
		return err
	}

	return nil
}

func (s *HealthMonitorAPIService) registerRoutes() {
	router := mux.NewRouter()

	router.HandleFunc(getDeviceInfoPath, s.APIHandler.GetDeviceInfo).Methods("GET")
	router.HandleFunc(getDeviceDataPath, s.APIHandler.GetDeviceData).Methods("GET")
	router.HandleFunc(registerPath, s.APIHandler.RegisterUser).Methods("POST")
	router.HandleFunc(loginPath, s.APIHandler.LoginUser).Methods("POST")

	s.router = router
}