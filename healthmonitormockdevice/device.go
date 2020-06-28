package main

import (
	"bytes"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorapi/domain"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const (
	didPrefix = "testdevice"
)

type Device struct {
	did string
	interval time.Duration
	ticker *time.Ticker
	stopChan chan struct{}
	totalTime time.Duration
	dataOK string
	count int
}

func NewDevice(did string, interval time.Duration, dataOK string) *Device {
	return &Device{
		did: did,
		interval: interval,
		stopChan: make(chan struct{}),
		totalTime:  0,
		dataOK: dataOK,
		count: 0,
	}
}

func (d *Device) GetDID() string {
	return d.did
}

func (d *Device) Start(wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {

		log.Infof("> device %v started with interval %v", d.did, d.interval)

		if d.createDevice() {
			d.generateDeviceData()
		}

		log.Infof("> device %v stopped", d.did)

		wg.Done()
	}()
}

func (d *Device) Stop() {
	close(d.stopChan)
}

func (d *Device) createDevice() bool {
	deviceInfo := &domain.RegisterDeviceInfoRequest{
		DID: d.did,
		PatientName: "John Doe",
	}

	bodyBytes, err := json.Marshal(deviceInfo)
	if err != nil {
		log.WithError(err).Errorf("> device %v error marshalling device info", d.did)
		return false
	}
	reader := bytes.NewReader(bodyBytes)

	resp, err := http.Post(healthmonitorAPIURL + registerDevicePath, contentType, reader)
	if err != nil {
		log.WithError(err).Errorf("> device %v error sending request to healthmonitorapi service to register device", d.did)
		return false
	}

	if resp.StatusCode != 200 {
		log.Errorf("> device %v register device response was not OK = %v", d.did, resp.StatusCode)
		return false
	}

	return true
}

func (d *Device) generateDeviceData() {

	d.ticker = time.NewTicker(d.interval)

	for {
		select {
		case <- d.ticker.C:
			var temp, hr, ecg, spo2 float64
			if d.dataOK == "good" {
				temp = generateRandomFloat64(35, 37)
				hr = generateRandomFloat64(60, 100)
				ecg = generateRandomFloat64(200, 1000)
				spo2 = generateRandomFloat64(90, 100)
			} else {
				temp = generateRandomFloat64(36, 38.5)
				hr = generateRandomFloat64(90, 120)
				ecg = generateRandomFloat64(200, 1000)
				spo2 = generateRandomFloat64(80, 95)
			}

			deviceDataset := domain.DeviceDataset{
				DID: d.did,
				Data: []*domain.DeviceData{
					{
						Temperature: temp,
						Heartrate: hr,
						ECG:  ecg,
						SPO2: spo2,
						Timestamp: time.Now(),
					},
				},
			}

			bodyBytes, err := json.Marshal(deviceDataset)
			if err != nil {
				log.WithError(err).Errorf("> device %v error marshalling device data", d.did)
				return
			}
			reader := bytes.NewReader(bodyBytes)

			startTime := time.Now()
			resp, err := http.Post(healthmonitorAPIURL + registerDataPath, contentType, reader)
			d.totalTime = d.totalTime + time.Since(startTime)
			d.count = d.count + 1
			if err != nil {
				log.WithError(err).Errorf("> device %v error sending request to healthmonitorapi service to register device data", d.did)
				return
			}

			if resp.StatusCode != 200 {
				log.Errorf("> device %v register device data response was not OK = %v", d.did, resp.StatusCode)
				return
			}



			resp.Body.Close()
		case <- d.stopChan:
			d.ticker.Stop()
			return
		}
	}
}

func (d *Device) GetStats() (time.Duration, int) {
	return d.totalTime, d.count
}

func generateRandomFloat64(min float64, max float64) float64 {
	rand.Seed(time.Now().UnixNano())

	return min + rand.Float64() * (max - min)
}