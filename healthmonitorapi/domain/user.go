package domain

type RegisterUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterUserResponse struct {
	Username string `json:"username"`
}

type LoginUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginUserResponse struct {
	Token string `json:"token"`
}

type AddDevicesRequest struct {
	UserDevices []string `json:"user_devices"`
}

type GetDevicesResponse struct {
	UserDevices []string `json:"user_devices"`
}
