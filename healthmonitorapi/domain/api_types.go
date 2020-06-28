package domain

type RegisterUserRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	PhoneNumber string `json:"phone_number"`
}

type RegisterUserResponse struct {
	Username  string  `json:"username"`
	APIErrors []error `json:"errors"`
}

type LoginUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginUserResponse struct {
	Token     string  `json:"token"`
	APIErrors []error `json:"errors"`
}

type AddDeleteDevicesRequest struct {
	UserDevice string `json:"user_device"`
}

type AddDeleteDevicesResponse struct {
	APIErrors []error `json:"errors"`
}

type GetDevicesResponse struct {
	UserDevices []string `json:"user_devices"`
	APIErrors   []error  `json:"errors"`
}

type AddDeleteSubscriptionRequest struct {
	DID string `json:"did"`
}

type AddDeleteSubscriptionResponse struct {
	APIErrors []error `json:"errors"`
}

type UpdateDeviceInfoRequest struct {
	DID         string `json:"did"`
	PatientName string `json:"patient_name"`
}

type RegisterDeviceInfoRequest struct {
	DID string `json:"did"`
}

type RegisterDeviceInfoResponse struct {
	DeviceInfo *DeviceInfo `json:"info,omitempty"`
	APIErrors  []error     `json:"errors"`
}

type DeviceDataResponse struct {
	Dataset   *DeviceDataset `json:"device_dataset"`
	APIErrors []error        `json:"errors"`
}

type DeviceAlertsResponse struct {
	Alerts    []*Alert `json:"alerts"`
	APIErrors []error  `json:"errors"`
}

type RegisterDeviceDataResponse struct {
	AlertCodes []int `json:"alert_codes"`
}

type DeviceReportGenerationRequest struct {
	DID string `json:"did"`
}

type DeviceReportGenerationResponse struct {
	ReportName string  `json:"report_name"`
	APIErrors  []error `json:"errors"`
}
