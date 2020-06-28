package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorapi/domain"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	stopCommand = "stop"
	statsCommand = "stats"

	healthmonitorAPIURL = "http://healthmonitor-d2400c9ab166d3ea.elb.us-east-2.amazonaws.com"
	loginPath = "/healthmonitorapi/auth/login"
	userDevicesPath = "/healthmonitorapi/entities/users/devices"
	registerDevicePath = "/healthmonitorapi/entities/devices/info"
	registerDataPath = "/healthmonitorapi/entities/devices/data"

	contentType = "application/json"
	authorizationHeader = "Authorization"
	authorizationType = "Bearer"
)

func main() {
	if len(os.Args) != 6 {
		log.Errorln("> usage ./healthmonitormockdevice device_count data_interval_seconds username password dataOK")
		return
	}

	count, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Errorf("> failed to parse device_count = %v", os.Args[1])
		return
	}

	interval, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Errorf("> failed to parse data_interval_seconds = %v", os.Args[2])
		return
	}

	username := os.Args[3]
	password := os.Args[4]
	dataOK := os.Args[5]

	wg := sync.WaitGroup{}

	log.Infoln("> creating devices")
	devices := make([]*Device, 0, count)
	dids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		device := NewDevice(fmt.Sprintf("%v_%v%v", username, didPrefix, i), time.Duration(interval) * time.Second, dataOK)
		devices = append(devices, device)
		dids = append(dids, device.GetDID())
	}

	log.Infof("> registering devices for user=%v dids=%v", username, dids)
	err = registerDevicesForUser(username, password, dids)
	if err != nil {
		log.Errorf("> failed to register dids for user err=%v", err)
		return
	}

	log.Infoln("> starting devices")
	for _, device := range devices {
		device.Start(&wg)
	}

	waitForCommand(devices)

	log.Infoln("> stopping devices")
	for _, device := range devices {
		device.Stop()
	}

	wg.Wait()
}

func waitForCommand(devices []*Device) {
	consoleReader := bufio.NewReader(os.Stdin)

	for {
		log.Infoln("> enter the string " + stopCommand + " to stop the devices")
		log.Info("> ")

		command, err := consoleReader.ReadString('\n')
		if err != nil {
			log.WithError(err).Errorln("> error reading command from stdin")
			return
		}

		if strings.ToLower(strings.Trim(command, "\n")) == stopCommand {
			return
		}

		if strings.ToLower(strings.Trim(command, "\n")) == statsCommand {

			var totalSum time.Duration
			totalCount := 0
			for _, device := range devices {
				sumTime, count := device.GetStats()
				totalSum = totalSum + sumTime
				totalCount = totalCount + count
			}

			fmt.Printf("Stats : average write duration is time=%v, count=%v\n\n", totalCount, totalSum)

			continue
		}

		log.Errorln("> unrecognised command")
	}
}

func registerDevicesForUser(username string, password string, dids []string) error {
	client := &http.Client{}

	loginReq := domain.LoginUserRequest{
		Username: username,
		Password: password,
	}

	bodyBytes, err := json.Marshal(loginReq)
	if err != nil {
		return errors.New("failed to marshal login request")
	}
	reader := bytes.NewReader(bodyBytes)

	req, err := http.NewRequest("POST", healthmonitorAPIURL + loginPath, reader)
	if err != nil {
		return errors.New("failed to create login request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return errors.New("failed to do login request")
	}
	if resp.StatusCode != 200 {
		return errors.New("login request was not 200")
	}
	defer resp.Body.Close()

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("failed to read login response")
	}

	var loginResponseBody domain.LoginUserResponse
	err = json.Unmarshal(bodyBytes, &loginResponseBody)
	if err != nil {
		return errors.New("failed to unmarshall login response")
	}

	token := loginResponseBody.Token
	if token == "" {
		return errors.New("got empty token")
	}

	log.Infof("login ok token=%v", token)

	for _, did := range dids {
		addDevicesReq := domain.AddDeleteDevicesRequest{
			UserDevice: did,
		}

		bodyBytes, err = json.Marshal(addDevicesReq)
		if err != nil {
			return errors.New("failed to marshal add devices request")
		}
		reader = bytes.NewReader(bodyBytes)

		req, err = http.NewRequest("POST", healthmonitorAPIURL + userDevicesPath, reader)
		if err != nil {
			return errors.New("failed to create add devices request")
		}
		req.Header.Set(authorizationHeader, authorizationType + " " + token)

		resp, err = client.Do(req)
		if err != nil {
			return errors.New("failed to do add devices request")
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Errorf("got status=%v", resp.StatusCode)
			return errors.New("add devices request was not 200")
		}
	}

	return nil
}
