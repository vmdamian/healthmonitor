package gateways

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorapi/domain"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	usernameQueryParam = "username"
	passwordQueryParam = "password"
	didQueryParam = "did"
	sinceQueryParam = "since"
)

type APIHandler struct {
	usersRepo *UsersRepo
	devicesRepo *DevicesRepo
	passwordSalt string
}

func NewAPIHandler(usersRepo *UsersRepo, devicesRepo *DevicesRepo, passwordSalt string) *APIHandler {
	return &APIHandler{
		usersRepo: usersRepo,
		devicesRepo: devicesRepo,
		passwordSalt: passwordSalt,
	}
}

func (h *APIHandler) GetDeviceInfo(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)
	ctx := req.Context()

	// TODO: Make this use token instead of username and password.
	username := req.URL.Query()[usernameQueryParam][0]
	password := req.URL.Query()[passwordQueryParam][0]
	did := req.URL.Query()[didQueryParam][0]

	if username == "" || password == "" || did == "" {
		resp.WriteHeader(400)
		return
	}

	saltedPassword  := password + h.passwordSalt
	hashedPassword := sha256.Sum256([]byte(saltedPassword))
	cryptedPassword := hex.EncodeToString(hashedPassword[:])

	userAuth, err := h.usersRepo.AuthUser(ctx, username, cryptedPassword)
	if err != nil {
		log.WithError(err).Errorln("error trying to authenticate user")
		resp.WriteHeader(500)
		return
	}

	if !userAuth {
		resp.WriteHeader(403)
		return
	}

	info, err := h.devicesRepo.GetDeviceInfo(ctx, did)
	if err != nil {
		log.WithError(err).Errorln("error getting device info")
		resp.WriteHeader(500)
		return
	}

	bytes, err := json.Marshal(info)
	if err != nil {
		log.WithError(err).Errorln("failed to marshall device info response")
		resp.WriteHeader(500)
		return
	}

	_, err = resp.Write(bytes)
	if err != nil {
		log.WithError(err).Errorln("failed to write device info response")
		resp.WriteHeader(500)
		return
	}
}

func (h *APIHandler) GetDeviceData(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)
	ctx := req.Context()

	// TODO: Make this use token instead of username and password.
	username := req.URL.Query()[usernameQueryParam][0]
	password := req.URL.Query()[passwordQueryParam][0]
	did := req.URL.Query()[didQueryParam][0]
	since := req.URL.Query()[sinceQueryParam][0]

	if username == "" || password == "" || did == "" || since == "" {
		resp.WriteHeader(400)
		return
	}

	sinceTimestamp, err := strconv.ParseInt(since, 10, 64)
	if err != nil {
		log.WithError(err).Errorln("error trying to parse since param")
		resp.WriteHeader(400)
		return
	}
	sinceTime := time.Unix(sinceTimestamp, 0)

	saltedPassword  := password + h.passwordSalt
	hashedPassword := sha256.Sum256([]byte(saltedPassword))
	cryptedPassword := hex.EncodeToString(hashedPassword[:])

	userAuth, err := h.usersRepo.AuthUser(ctx, username, cryptedPassword)
	if err != nil {
		log.WithError(err).Errorln("error trying to authenticate user")
		resp.WriteHeader(500)
		return
	}

	if !userAuth {
		resp.WriteHeader(403)
		return
	}

	data, err := h.devicesRepo.GetDeviceData(ctx, did, sinceTime)
	if err != nil {
		log.WithError(err).Errorln("error getting device data response")
		resp.WriteHeader(500)
		return
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Errorln("failed to marshall device data response")
		resp.WriteHeader(500)
		return
	}

	_, err = resp.Write(bytes)
	if err != nil {
		log.WithError(err).Errorln("failed to write device data response")
		resp.WriteHeader(500)
		return
	}
}

func (h *APIHandler) RegisterDeviceInfo(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)
	ctx := req.Context()

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register device info request body")
		resp.WriteHeader(500)
		return
	}

	var deviceInfoRequest domain.DeviceInfo
	err = json.Unmarshal(bodyBytes, &deviceInfoRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register device info request body")
		resp.WriteHeader(500)
		return
	}

	err = h.devicesRepo.RegisterDeviceInfo(ctx, deviceInfoRequest)
	if err != nil {
		log.WithError(err).Errorln("error registering device info")
		resp.WriteHeader(500)
		return
	}
}

func (h *APIHandler) RegisterDeviceData(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)
	ctx := req.Context()

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register device info request body")
		resp.WriteHeader(500)
		return
	}

	var deviceDatasetRequest domain.DeviceDataset
	err = json.Unmarshal(bodyBytes, &deviceDatasetRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register device info request body")
		resp.WriteHeader(500)
		return
	}

	err = h.devicesRepo.RegisterDeviceData(ctx, deviceDatasetRequest)
	if err != nil {
		log.WithError(err).Errorln("error registering device info")
		resp.WriteHeader(500)
		return
	}
}

func (h *APIHandler) RegisterUser(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)
	ctx := req.Context()

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register request body")
		resp.WriteHeader(500)
		return
	}

	var registerUserRequest domain.RegisterUserRequest
	err = json.Unmarshal(bodyBytes, &registerUserRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register request body")
		resp.WriteHeader(500)
		return
	}

	saltedPassword  := registerUserRequest.Password + h.passwordSalt
	hashedPassword := sha256.Sum256([]byte(saltedPassword))
	cryptedPassword := hex.EncodeToString(hashedPassword[:])

	err = h.usersRepo.RegisterUser(ctx, registerUserRequest.Username, cryptedPassword)
	if err != nil {
		log.WithError(err).Errorln("error trying to register user")
		resp.WriteHeader(500)
		return
	}

	registerUserResponse := &domain.RegisterUserResponse{
		Username: registerUserRequest.Username,
	}

	bytes, err := json.Marshal(registerUserResponse)
	if err != nil {
		log.WithError(err).Errorln("error marshalling register response body")
		resp.WriteHeader(500)
		return
	}

	_, err = resp.Write(bytes)
	if err != nil {
		log.WithError(err).Errorln("error writing register response body")
		resp.WriteHeader(500)
		return
	}
}

func (h *APIHandler) LoginUser(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)
	ctx := req.Context()

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading login request body")
		resp.WriteHeader(500)
		return
	}

	var loginUserRequest domain.LoginUserRequest
	err = json.Unmarshal(bodyBytes, &loginUserRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling login request body")
		resp.WriteHeader(500)
		return
	}

	saltedPassword  := loginUserRequest.Password + h.passwordSalt
	hashedPassword := sha256.Sum256([]byte(saltedPassword))
	cryptedPassword := hex.EncodeToString(hashedPassword[:])

	userAuth, token, err := h.usersRepo.LoginUser(ctx, loginUserRequest.Username, cryptedPassword)
	if err != nil {
		log.WithError(err).Errorln("error trying to authenticate user")
		resp.WriteHeader(500)
		return
	}

	if !userAuth {
		resp.WriteHeader(403)
		return
	}

	loginUserResponse := &domain.LoginUserResponse{
		Token: token,
	}

	bytes, err := json.Marshal(loginUserResponse)
	if err != nil {
		log.WithError(err).Errorln("error marshalling login response body")
		resp.WriteHeader(500)
		return
	}

	_, err = resp.Write(bytes)
	if err != nil {
		log.WithError(err).Errorln("error writing login response body")
		resp.WriteHeader(500)
		return
	}
}