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
	"strings"
	"time"
)

const (
	authorizationHeader = "Authorization"
	authorizationType = "Bearer"

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
	var statusCode int
	var bodyBytes []byte

	defer func() {
		resp.WriteHeader(statusCode)
		_, err := resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	ctx := req.Context()

	dids := req.URL.Query()[didQueryParam]

	if len(dids) != 1 {
		statusCode = 400
		return
	}

	did := dids[0]
	if did == "" {
		statusCode = 400
		return
	}

	auth := req.Header.Get(authorizationHeader)
	token := strings.TrimPrefix(auth, authorizationType + " ")

	userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorln("error trying to authenticate token")
		statusCode = 500
		return
	}

	if !userAuth {
		statusCode = 403
		return
	}

	info, err := h.devicesRepo.GetDeviceInfo(ctx, did)
	if err != nil {
		log.WithError(err).Errorln("error getting device info")
		statusCode = 500
		return
	}

	bodyBytes, err = json.Marshal(info)
	if err != nil {
		log.WithError(err).Errorln("failed to marshall device info response")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) GetDeviceData(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var bodyBytes []byte

	defer func() {
		resp.WriteHeader(statusCode)
		_, err := resp.Write(bodyBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to write device data response")
		}
	}()

	ctx := req.Context()

	dids := req.URL.Query()[didQueryParam]
	sinces := req.URL.Query()[sinceQueryParam]

	if len(dids) != 1 || len(sinces) != 1 {
		statusCode = 400
		return
	}

	did := req.URL.Query()[didQueryParam][0]
	since := req.URL.Query()[sinceQueryParam][0]

	if did == "" || since == "" {
		statusCode = 400
		return
	}

	sinceTimestamp, err := strconv.ParseInt(since, 10, 64)
	if err != nil {
		log.WithError(err).Errorln("error trying to parse since param")
		statusCode = 400
		return
	}
	sinceTime := time.Unix(sinceTimestamp, 0)

	auth := req.Header.Get(authorizationHeader)
	token := strings.TrimPrefix(auth, authorizationType + " ")

	userAuth, err := h.usersRepo.AuthToken(ctx, token)
	if err != nil {
		log.WithError(err).Errorln("error trying to authenticate token")
		statusCode = 500
		return
	}

	if !userAuth {
		statusCode = 403
		return
	}

	data, err := h.devicesRepo.GetDeviceData(ctx, did, sinceTime)
	if err != nil {
		log.WithError(err).Errorln("error getting device data response")
		statusCode = 500
		return
	}

	bodyBytes, err = json.Marshal(data)
	if err != nil {
		log.WithError(err).Errorln("failed to marshall device data response")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) RegisterDeviceInfo(resp http.ResponseWriter, req *http.Request) {
	var statusCode int

	defer func() {
		resp.WriteHeader(statusCode)
	}()

	ctx := req.Context()

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register device info request body")
		statusCode = 500
		return
	}

	var deviceInfoRequest domain.DeviceInfo
	err = json.Unmarshal(bodyBytes, &deviceInfoRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register device info request body")
		statusCode = 500
		return
	}

	err = h.devicesRepo.RegisterDeviceInfo(ctx, deviceInfoRequest)
	if err != nil {
		log.WithError(err).Errorln("error registering device info")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) RegisterDeviceData(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	defer func() {
		resp.WriteHeader(statusCode)
	}()
	ctx := req.Context()

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Errorln("error reading register device info request body")
		statusCode = 500
		return
	}

	var deviceDatasetRequest domain.DeviceDataset
	err = json.Unmarshal(bodyBytes, &deviceDatasetRequest)
	if err != nil {
		log.WithError(err).Errorln("error unmarshalling register device info request body")
		statusCode = 500
		return
	}

	err = h.devicesRepo.RegisterDeviceData(ctx, deviceDatasetRequest)
	if err != nil {
		log.WithError(err).Errorln("error registering device info")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) RegisterUser(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var bodyBytes []byte

	defer func() {
		resp.WriteHeader(statusCode)
		_, err := resp.Write(bodyBytes)
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

	saltedPassword  := registerUserRequest.Password + h.passwordSalt
	hashedPassword := sha256.Sum256([]byte(saltedPassword))
	cryptedPassword := hex.EncodeToString(hashedPassword[:])

	err = h.usersRepo.RegisterUser(ctx, registerUserRequest.Username, cryptedPassword)
	if err != nil {
		log.WithError(err).Errorln("error trying to register user")
		statusCode = 500
		return
	}

	registerUserResponse := &domain.RegisterUserResponse{
		Username: registerUserRequest.Username,
	}

	bodyBytes, err = json.Marshal(registerUserResponse)
	if err != nil {
		log.WithError(err).Errorln("error marshalling register response body")
		statusCode = 500
		return
	}

	statusCode = 200
}

func (h *APIHandler) LoginUser(resp http.ResponseWriter, req *http.Request) {
	var statusCode int
	var bodyBytes []byte

	defer func() {
		resp.WriteHeader(statusCode)
		_, err := resp.Write(bodyBytes)
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

	saltedPassword  := loginUserRequest.Password + h.passwordSalt
	hashedPassword := sha256.Sum256([]byte(saltedPassword))
	cryptedPassword := hex.EncodeToString(hashedPassword[:])

	userAuth, token, err := h.usersRepo.LoginUser(ctx, loginUserRequest.Username, cryptedPassword)
	if err != nil {
		log.WithError(err).Errorln("error trying to authenticate user")
		statusCode = 500
		return
	}

	if !userAuth {
		statusCode = 403
		return
	}

	loginUserResponse := &domain.LoginUserResponse{
		Token: token,
	}

	bodyBytes, err = json.Marshal(loginUserResponse)
	if err != nil {
		log.WithError(err).Errorln("error marshalling login response body")
		statusCode = 500
		return
	}

	statusCode = 200
}