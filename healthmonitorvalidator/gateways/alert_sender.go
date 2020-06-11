package gateways

import (
	"errors"
	"fmt"
	"healthmonitor/healthmonitorvalidator/domain"
	"net/http"
	"net/url"
	"strings"
)

type AlertSender struct {
	accountSID  string
	authToken   string
	phoneNumber string
	url         string
}

const (
	alertGreetingFormat = "WARNING! You have active alerts for device %v from HEALTHMONITOR!\n"
	alertRowHeader      = "ALERT_TYPE --- STATUS --- START TIME\n"
	alertRowFormat      = "%v --- %v --- %v\n"
)

func NewAlertSender(accountSID string, authToken string, phoneNumber string) *AlertSender {
	return &AlertSender{
		accountSID:  accountSID,
		authToken:   authToken,
		phoneNumber: phoneNumber,
		url:         "https://api.twilio.com/2010-04-01/Accounts/" + accountSID + "/Messages.json",
	}
}

func (as *AlertSender) SendAlerts(receiverPhoneNumber string, did string, alerts []*domain.Alert) error {
	msgData := url.Values{}
	msgData.Set("To", receiverPhoneNumber)
	msgData.Set("From", as.phoneNumber)
	msgData.Set("Body", as.generateMessageFromAlerts(did, alerts))
	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("POST", as.url, &msgDataReader)
	req.SetBasicAuth(as.accountSID, as.authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := client.Do(req)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New(fmt.Sprintf("failed to send text message to %v, got status %v", receiverPhoneNumber, resp.StatusCode))
	}

	return nil
}

func (as *AlertSender) generateMessageFromAlerts(did string, alerts []*domain.Alert) string {
	alertMessage := fmt.Sprintf(alertGreetingFormat, did)
	alertMessage = alertMessage + alertRowHeader
	for _, alert := range alerts {
		alertMessage = alertMessage + fmt.Sprintf(alertRowFormat, alert.AlertType, alert.Status, alert.CreatedTimestamp)
	}

	return alertMessage
}
