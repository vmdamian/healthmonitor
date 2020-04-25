package main

import (
	"bufio"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	stopCommand = "stop"
)

func main() {
	if len(os.Args) != 3 {
		log.Errorln("> usage ./healthmonitormockdevice device_count data_interval_seconds")
		return
	}

	count, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Errorf("> failed to parse device_count = %v", os.Args[1])
	}

	interval, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Errorf("> failed to parse data_interval_seconds = %v", os.Args[2])
	}

	wg := sync.WaitGroup{}

	log.Infoln("starting devices")
	devices := make([]*Device, 0, count)
	for i := 0; i < count; i++ {
		device := NewDevice(fmt.Sprintf("%v%v", didPrefix, i), time.Duration(interval) * time.Second)
		devices = append(devices, device)
		device.Start(&wg)
	}

	waitForStopCommand()

	log.Infoln("> stopping devices")
	for _, device := range devices {
		device.Stop()
	}

	wg.Wait()
}

func waitForStopCommand() {
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

		log.Errorln("> unrecognised command")
	}
}

