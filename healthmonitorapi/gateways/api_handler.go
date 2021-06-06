package gateways

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorapi/domain"
	"healthmonitor/healthmonitorapi/usecases/validation"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	authorizationHeader = "Authorization"
	authorizationType   = "Bearer"

	didQueryParam   = "did"
	sinceQueryParam = "since"
	toQueryParam    = "to"
)

type APIHandler struct {
	usersRepo          *UsersRepo
	devicesRepo        *DevicesRepo
	MessagingRepo      *MessagingRepo
	validator          *validation.MinimalistValidator
	passwordSalt       string
	validationInterval time.Duration
	fileUploader *ClientUploader
}

func NewAPIHandler(usersRepo *UsersRepo, devicesRepo *DevicesRepo, messagingRepo *MessagingRepo, validator *validation.MinimalistValidator, fileUploader *ClientUploader, passwordSalt string, validationInterval time.Duration) *APIHandler {
	return &APIHandler{
		usersRepo:          usersRepo,
		devicesRepo:        devicesRepo,
		MessagingRepo:      messagingRepo,
		validator:          validator,
		passwordSalt:       passwordSalt,
		validationInterval: validationInterval,
		fileUploader: fileUploader,
	}
}

func (h *APIHandler) GetHealth(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)
}

func (h *APIHandler) GetDeviceInfo(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var info *domain.DeviceInfo
	var apiErrors []error

	ctx := req.Context()

	defer func() {
		// Form and write response.
		infoResp := &domain.DeviceInfoResponse{
			Info:      info,
			APIErrors: apiErrors,
		}

		bodyBytes, err := json.Marshal(infoResp)
		if err != nil {
			log.WithError(err).Errorln("failed to marshall device info response")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	// Auth received token.
	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		apiErrors = append(apiErrors, fmt.Errorf("invalid token=%v", token))
		statusCode = 403
		return
	}

	// Validate the received DID.
	did, err := validateDIDParam(req)
	if err != nil {
		apiErrors = append(apiErrors, err)
		statusCode = 400
		return
	}

	// Get devices that the user can access.
	userDevices, err := h.usersRepo.GetDevicesForUser(ctx, username)
	if err != nil {
		log.WithError(err).Errorf("error getting devices for user=%v", username)
		statusCode = 500
		return
	}
	if !stringInList(did, userDevices) {
		apiErrors = append(apiErrors, fmt.Errorf("permission denied"))
		statusCode = 403
		return
	}

	// Get info for that specific device.
	info, err = h.devicesRepo.GetDeviceInfo(ctx, did)
	if err != nil {
		log.WithError(err).Errorln("error getting device info")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) GetDeviceData(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var dataset *domain.DeviceDataset
	var apiErrors []error

	ctx := req.Context()

	defer func() {
		// Form and write response.
		datasetResp := &domain.DeviceDataResponse{
			Dataset:   dataset,
			APIErrors: apiErrors,
		}

		bodyBytes, err := json.Marshal(datasetResp)
		if err != nil {
			log.WithError(err).Errorln("failed to marshall device data response")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	// Auth received token.
	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		apiErrors = append(apiErrors, fmt.Errorf("invalid token=%v", token))
		statusCode = 403
		return
	}

	// Validate received params.
	did, err := validateDIDParam(req)
	if err != nil {
		statusCode = 400
		apiErrors = append(apiErrors, err)
		return
	}
	sinceTime, err := validateSinceParam(req)
	if err != nil {
		statusCode = 400
		apiErrors = append(apiErrors, err)
		return
	}

	toTime, err := validateToParam(req)
	if err != nil {
		statusCode = 400
		apiErrors = append(apiErrors, err)
		return
	}

	// Get devices that the user can access.
	userDevices, err := h.usersRepo.GetDevicesForUser(ctx, username)
	if err != nil {
		log.WithError(err).Errorf("error getting devices for user=%v", username)
		statusCode = 500
		return
	}
	if !stringInList(did, userDevices) {
		apiErrors = append(apiErrors, fmt.Errorf("permission denied"))
		statusCode = 403
		return
	}

	// Get data for the specific device.
	dataset, err = h.devicesRepo.GetDeviceData(ctx, did, sinceTime, toTime)
	if err != nil {
		log.WithError(err).Errorln("error getting device data response")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) RegisterDeviceInfo(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var deviceInfo *domain.DeviceInfo
	var apiErrors []error

	defer func() {
		registerInfoResp := &domain.RegisterDeviceInfoResponse{
			DeviceInfo: deviceInfo,
			APIErrors:  apiErrors,
		}

		bodyBytes, err := json.Marshal(registerInfoResp)
		if err != nil {
			log.WithError(err).Errorln("failed to marshall device data response")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	ctx := req.Context()

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register device info request body")
		statusCode = 500
		return
	}

	var registerDeviceRequest domain.RegisterDeviceInfoRequest
	err = json.Unmarshal(bodyBytes, &registerDeviceRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register device info request body")
		statusCode = 500
		return
	}

	did := registerDeviceRequest.DID
	patient_name := registerDeviceRequest.PatientName
	deviceInfo, err = h.devicesRepo.RegisterDeviceInfo(ctx, did, patient_name)
	if err != nil {
		log.WithError(err).Errorln("error registering device info")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) RegisterDeviceData(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var alertCodes []int

	defer func() {
		registerDataResp := &domain.RegisterDeviceDataResponse{
			AlertCodes: alertCodes,
		}

		bodyBytes, err := json.Marshal(registerDataResp)
		if err != nil {
			log.WithError(err).Errorln("failed to marshall device data response")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	ctx := req.Context()

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register device data request body")
		statusCode = 500
		return
	}
	var deviceDatasetRequest domain.DeviceDatasetAPI
	err = json.Unmarshal(bodyBytes, &deviceDatasetRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register device data request body")
		statusCode = 500
		return
	}

	var deviceDataset domain.DeviceDataset
	deviceDataset.DID = deviceDatasetRequest.DID
	for _, data := range deviceDatasetRequest.Data {
		timestamp, err := time.Parse(time.RFC3339, data.Timestamp)
		if err != nil {
			fmt.Println(err)
			continue
		}
		parsedData := &domain.DeviceData{
			DID:         data.DID,
			Timestamp:   timestamp,
			Temperature: data.Temperature,
			Heartrate:   data.Heartrate,
			ECG:         data.ECG,
			SPO2:        data.SPO2,
		}
		deviceDataset.Data = append(deviceDataset.Data, parsedData)
	}

	err = h.devicesRepo.RegisterDeviceData(ctx, deviceDataset)
	if err != nil {
		log.WithError(err).Errorln("error registering device info")
		statusCode = 500
		return
	}

	deviceinfo, err := h.devicesRepo.GetDeviceInfo(ctx, deviceDataset.DID)
	if err != nil {
		log.WithError(err).Errorf("error sending a validation request for did=%v", deviceDatasetRequest.DID)
		statusCode = 500
		return
	}

	if time.Since(deviceinfo.LastValidationTimestamp) > h.validationInterval {
		err = h.MessagingRepo.SendValidationRequest(ctx, deviceDatasetRequest.DID)
		if err != nil {
			log.WithError(err).Errorf("error sending a validation request for did=%v", deviceDatasetRequest.DID)
			statusCode = 500
			return
		}
	}

	alertCodes = h.validator.CheckDataset(&deviceDataset)

	statusCode = 200
}

func (h *APIHandler) RegisterUser(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var username string
	var apiErrors []error

	defer func() {
		registerUserResponse := &domain.RegisterUserResponse{
			Username:  username,
			APIErrors: apiErrors,
		}

		bodyBytes, err := json.Marshal(registerUserResponse)
		if err != nil {
			log.WithError(err).Errorln("error marshalling register response body")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	ctx := req.Context()

	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register request body")
		statusCode = 500
		return
	}
	var registerUserRequest domain.RegisterUserRequest
	err = json.Unmarshal(bytes, &registerUserRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register request body")
		statusCode = 500
		return
	}
	username = registerUserRequest.Username

	err = h.usersRepo.RegisterUser(ctx, registerUserRequest.Username, h.encryptPassword(registerUserRequest.Password), registerUserRequest.PhoneNumber)
	if err != nil {
		log.WithError(err).Errorln("error trying to register user")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) LoginUser(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var token string

	defer func() {
		loginUserResponse := &domain.LoginUserResponse{
			Token: token,
		}

		bodyBytes, err := json.Marshal(loginUserResponse)
		if err != nil {
			log.WithError(err).Errorln("error marshalling login response body")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	ctx := req.Context()

	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading login request body")
		statusCode = 500
		return
	}

	var loginUserRequest domain.LoginUserRequest
	err = json.Unmarshal(bytes, &loginUserRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling login request body")
		statusCode = 500
		return
	}

	userAuth, token, err := h.usersRepo.LoginUser(ctx, loginUserRequest.Username, h.encryptPassword(loginUserRequest.Password))
	if err != nil {
		log.WithError(err).Errorln("error trying to authenticate user")
		statusCode = 500
		return
	}
	if !userAuth {
		statusCode = 403
		return
	}

	statusCode = 200
}

func (h *APIHandler) AddDevices(resp http.ResponseWriter, req *http.Request) {
	var statusCode int

	defer func() {
		resp.WriteHeader(statusCode)
	}()

	ctx := req.Context()

	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		statusCode = 403
		return
	}

	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading add devices request body")
		statusCode = 500
		return
	}
	var addDevicesRequest domain.AddDeleteDevicesRequest
	err = json.Unmarshal(bytes, &addDevicesRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling add devices request body")
		statusCode = 500
		return
	}

	if addDevicesRequest.UserDevice == "" {
		statusCode = 400
		return
	}

	err = h.usersRepo.AddDevicesForUser(ctx, username, []string{addDevicesRequest.UserDevice})
	if err != nil {
		log.WithError(err).Errorf("failed to add devices for user=%v devices=%v", username, addDevicesRequest.UserDevice)
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) GetDevices(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var apiErrors []error
	var userDevices []string

	defer func() {
		getDevicesResponse := &domain.GetDevicesResponse{
			UserDevices: userDevices,
			APIErrors:   apiErrors,
		}

		bodyBytes, err := json.Marshal(getDevicesResponse)
		if err != nil {
			log.WithError(err).Errorln("error marshalling login response body")
			statusCode = 500
			return
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	ctx := req.Context()

	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		statusCode = 403
		return
	}

	userDevices, err = h.usersRepo.GetDevicesForUser(ctx, username)
	if err != nil {
		log.WithError(err).Errorf("error getting devices for user=%v", username)
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) DeleteDevices(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var apiErrors []error
	var userDevices []string

	defer func() {
		getDevicesResponse := &domain.GetDevicesResponse{
			UserDevices: userDevices,
			APIErrors:   apiErrors,
		}

		bodyBytes, err := json.Marshal(getDevicesResponse)
		if err != nil {
			log.WithError(err).Errorln("error marshalling login response body")
			statusCode = 500
			return
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	ctx := req.Context()

	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		statusCode = 403
		return
	}

	did, err := validateDIDParam(req)
	if err != nil {
		apiErrors = append(apiErrors, err)
		statusCode = 400
		return
	}

	err = h.usersRepo.DeleteDevicesForUser(ctx, username, []string{did})
	if err != nil {
		log.WithError(err).Errorf("failed to add devices for user=%v devices=%v", username, did)
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) StartReportGeneration(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var apiErrors []error
	var reportName string
	defer func() {
		reportGenerationResponse := &domain.DeviceReportGenerationResponse{
			ReportName: reportName,
			APIErrors:  apiErrors,
		}

		bodyBytes, err := json.Marshal(reportGenerationResponse)
		if err != nil {
			log.WithError(err).Errorln("error marshalling login response body")
			statusCode = 500
			return
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	ctx := req.Context()

	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		statusCode = 403
		return
	}

	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading login request body")
		statusCode = 500
		return
	}

	var reportGenRequest domain.DeviceReportGenerationRequest
	err = json.Unmarshal(bytes, &reportGenRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling login request body")
		statusCode = 500
		return
	}

	reportName = fmt.Sprintf("%v_%v_%v", username, reportGenRequest.DID, uuid.New().String())
	err = h.MessagingRepo.SendReportGenerationRequest(ctx, reportName)
	if err != nil {
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) GetAlerts(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var alerts []*domain.Alert
	var apiErrors []error

	ctx := req.Context()

	defer func() {
		// Form and write response.
		datasetResp := &domain.DeviceAlertsResponse{
			Alerts:    alerts,
			APIErrors: apiErrors,
		}

		bodyBytes, err := json.Marshal(datasetResp)
		if err != nil {
			log.WithError(err).Errorln("failed to marshall device alerts response")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device alerts response")
		}
	}()

	// Auth received token.
	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		apiErrors = append(apiErrors, fmt.Errorf("invalid token=%v", token))
		statusCode = 403
		return
	}

	// Validate received params.
	did, err := validateDIDParam(req)
	if err != nil {
		statusCode = 400
		apiErrors = append(apiErrors, err)
		return
	}

	// Get devices that the user can access.
	userDevices, err := h.usersRepo.GetDevicesForUser(ctx, username)
	if err != nil {
		log.WithError(err).Errorf("error getting devices for user=%v", username)
		statusCode = 500
		return
	}
	if !stringInList(did, userDevices) {
		apiErrors = append(apiErrors, fmt.Errorf("permission denied"))
		statusCode = 403
		return
	}

	// Get alerts for the specific device.
	alerts, err = h.devicesRepo.GetAlerts(ctx, did)
	if err != nil {
		log.WithError(err).Errorln("error getting device alerts response")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) AddSubscription(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var apiErrors []error

	ctx := req.Context()

	defer func() {
		// Form and write response.
		datasetResp := &domain.AddDeleteSubscriptionResponse{
			APIErrors: apiErrors,
		}

		bodyBytes, err := json.Marshal(datasetResp)
		if err != nil {
			log.WithError(err).Errorln("failed to marshall device alerts response")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device alerts response")
		}
	}()

	// Auth received token.
	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		apiErrors = append(apiErrors, fmt.Errorf("invalid token=%v", token))
		statusCode = 403
		return
	}

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register device info request body")
		statusCode = 500
		return
	}
	var addSubscriptionRequest domain.AddDeleteSubscriptionRequest
	err = json.Unmarshal(bodyBytes, &addSubscriptionRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register device info request body")
		statusCode = 500
		return
	}

	// Get devices that the user can access.
	userDevices, err := h.usersRepo.GetDevicesForUser(ctx, username)
	if err != nil {
		log.WithError(err).Errorf("error getting devices for user=%v", username)
		statusCode = 500
		return
	}
	if !stringInList(addSubscriptionRequest.DID, userDevices) {
		apiErrors = append(apiErrors, fmt.Errorf("permission denied"))
		statusCode = 403
		return
	}

	phone, err := h.usersRepo.GetUserPhone(ctx, username)
	if err != nil {
		log.WithError(err).Errorf("error getting devices for user=%v", username)
		statusCode = 500
		return
	}

	// Get alerts for the specific device.
	err = h.devicesRepo.AddSubscription(ctx, addSubscriptionRequest.DID, phone)
	if err != nil {
		log.WithError(err).Errorln("error getting device alerts response")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) DeleteSubscription(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var apiErrors []error

	ctx := req.Context()

	defer func() {
		// Form and write response.
		datasetResp := &domain.AddDeleteSubscriptionResponse{
			APIErrors: apiErrors,
		}

		bodyBytes, err := json.Marshal(datasetResp)
		if err != nil {
			log.WithError(err).Errorln("failed to marshall device alerts response")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device alerts response")
		}
	}()

	// Auth received token.
	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		apiErrors = append(apiErrors, fmt.Errorf("invalid token=%v", token))
		statusCode = 403
		return
	}

	did, err := validateDIDParam(req)
	if err != nil {
		apiErrors = append(apiErrors, err)
		statusCode = 400
		return
	}

	// Get devices that the user can access.
	userDevices, err := h.usersRepo.GetDevicesForUser(ctx, username)
	if err != nil {
		log.WithError(err).Errorf("error getting devices for user=%v", username)
		statusCode = 500
		return
	}
	if !stringInList(did, userDevices) {
		apiErrors = append(apiErrors, fmt.Errorf("permission denied"))
		statusCode = 403
		return
	}

	phone, err := h.usersRepo.GetUserPhone(ctx, username)
	if err != nil {
		log.WithError(err).Errorf("error getting devices for user=%v", username)
		statusCode = 500
		return
	}

	// Get alerts for the specific device.
	err = h.devicesRepo.DeleteSubscription(ctx, did, phone)
	if err != nil {
		log.WithError(err).Errorln("error getting device alerts response")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) encryptPassword(password string) string {
	saltedPassword := password + h.passwordSalt
	hashedPassword := sha256.Sum256([]byte(saltedPassword))
	cryptedPassword := hex.EncodeToString(hashedPassword[:])
	return cryptedPassword
}

func (h *APIHandler) UpdateDeviceInfo(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var apiErrors []error

	ctx := req.Context()

	defer func() {
		// Form and write response.
		datasetResp := &domain.AddDeleteSubscriptionResponse{
			APIErrors: apiErrors,
		}

		bodyBytes, err := json.Marshal(datasetResp)
		if err != nil {
			log.WithError(err).Errorln("failed to marshall device alerts response")
			statusCode = 500
		}

		resp.WriteHeader(statusCode)
		_, err = resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device alerts response")
		}
	}()

	// Auth received token.
	token := getAuthToken(req)
	username, userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorf("error trying to authenticate token=%v", token)
		statusCode = 500
		return
	}
	if !userAuth {
		apiErrors = append(apiErrors, fmt.Errorf("invalid token=%v", token))
		statusCode = 403
		return
	}

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register device info request body")
		statusCode = 500
		return
	}
	var addSubscriptionRequest domain.UpdateDeviceInfoRequest
	err = json.Unmarshal(bodyBytes, &addSubscriptionRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register device info request body")
		statusCode = 500
		return
	}

	// Get devices that the user can access.
	userDevices, err := h.usersRepo.GetDevicesForUser(ctx, username)
	if err != nil {
		log.WithError(err).Errorf("error getting devices for user=%v", username)
		statusCode = 500
		return
	}
	if !stringInList(addSubscriptionRequest.DID, userDevices) {
		apiErrors = append(apiErrors, fmt.Errorf("permission denied"))
		statusCode = 403
		return
	}

	// Get alerts for the specific device.
	err = h.devicesRepo.UpdateDeviceInfo(ctx, addSubscriptionRequest.DID, addSubscriptionRequest.PatientName)
	if err != nil {
		log.WithError(err).Errorln("error getting device alerts response")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) GenerateExport(resp http.ResponseWriter, req *http.Request) {

	ctx := req.Context()
	uploader := h.fileUploader

	go func() {
		data, err := h.devicesRepo.ScrollDeviceData(ctx)
		if err != nil {
			log.WithError(err).Errorln("got error scrolling data")
			return
		}

		y, m, d := time.Now().Date()
		h, min, s := time.Now().Clock()
		fileName := fmt.Sprintf("%v%v%v_%v%v%v", y,m,d,h,min,s)

		_ = req.ParseMultipartForm(32 << 20)
		file, handler, err := req.FormFile(fileName)
		if err != nil {
			log.WithError(err).Errorln("got error with creating form file")
			return
		}
		defer file.Close()

		f, err := os.OpenFile(handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.WithError(err).Errorln("got error opening file")
			return
		}

		dataBytes,err := json.Marshal(data)
		if err != nil {
			log.WithError(err).Errorln("failed to marshall data")
			return
		}

		_, err = f.Write(dataBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write in file")
			return
		}

		err = uploader.UploadFile(f, fileName)
		if err != nil {
			log.WithError(err).Errorln("failed to upload file")
			return
		}
	}()

	resp.WriteHeader(200)
}

func stringInList(itemToFind string, items []string) bool {
	if items == nil {
		return false
	}

	for _, item := range items {
		if itemToFind == item {
			return true
		}
	}

	return false
}

func validateDIDParam(req *http.Request) (string, error) {
	dids := req.URL.Query()[didQueryParam]

	if len(dids) != 1 {
		return "", fmt.Errorf("got multiple or no devices; did=%v", dids)
	}

	did := dids[0]
	if did == "" {
		return "", fmt.Errorf("device id can not be an empty string")
	}

	return did, nil
}

func validateSinceParam(req *http.Request) (time.Time, error) {
	sinces := req.URL.Query()[sinceQueryParam]

	if len(sinces) != 1 {
		return time.Time{}, fmt.Errorf("got multiple or no since params; since=%v", sinces)
	}

	since := sinces[0]
	if since == "" {
		return time.Time{}, fmt.Errorf("since parameter can not be an empty string")
	}

	sinceTimestamp, err := strconv.ParseInt(since, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("error trying to parse since param; should be a unix timestamp; since=%v", since)
	}

	return time.Unix(sinceTimestamp, 0), nil
}

func validateToParam(req *http.Request) (time.Time, error) {
	tos := req.URL.Query()[toQueryParam]

	if len(tos) == 0 {
		return time.Now(), nil
	}

	if len(tos) > 1 {
		return time.Time{}, fmt.Errorf("got multiple or no to params; to=%v", tos)
	}

	to := tos[0]
	if to == "" {
		return time.Now(), nil
	}

	if to == "now" {
		return time.Now(), nil
	}

	toTimestamp, err := strconv.ParseInt(to, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("error trying to parse since param; should be a unix timestamp; to=%v", to)
	}

	return time.Unix(toTimestamp, 0), nil
}

func getAuthToken(req *http.Request) string {
	auth := req.Header.Get(authorizationHeader)
	token := strings.TrimPrefix(auth, authorizationType+" ")
	return token
}
