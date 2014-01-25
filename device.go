package geotrigger_golang

import (
	"net/url"
)

type device struct {
	tokenManager
	clientId  string
	deviceId  string
}

/* Device JSON structs */
type deviceRegisterResponse struct {
	DeviceJSON      deviceJSON      `json:"device"`
	DeviceTokenJSON deviceTokenJSON `json:"deviceToken"`
}

type deviceTokenJSON struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"`
}

type deviceJSON struct {
	DeviceId string `json:"deviceId"`
}

type deviceRefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64    `json:"expires_in"`
}

func (device *device) Request(route string, params map[string]interface{}, responseJSON interface{}) chan error {
	errorChan := make(chan error)
	go func() {
		errorChan <- geotriggerPost(device, route, params, responseJSON)
	}()
	return errorChan
}

func (device *device) GetSessionInfo() map[string]string {
	return map[string]string{
		"access_token":  device.getAccessToken(),
		"refresh_token": device.getRefreshToken(),
		"device_id":     device.deviceId,
		"client_id":     device.clientId,
	}
}

func newDevice(clientId string) (Session, chan error) {
	device := &device{
		clientId: clientId,
	}

	errorChan := make(chan error)
	go device.register(errorChan)

	return device, errorChan
}

func (device *device) register(errorChan chan error) {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.clientId)
	values.Set("f", "json")

	// make request
	var deviceRegisterResponse deviceRegisterResponse
	if err := agoPost(ago_register_route, []byte(values.Encode()), &deviceRegisterResponse); err != nil {
		go func() {
			errorChan <- err
		}()
		return
	}

	device.deviceId = deviceRegisterResponse.DeviceJSON.DeviceId
	device.tokenManager = newTokenManager(deviceRegisterResponse.DeviceTokenJSON.AccessToken,
		deviceRegisterResponse.DeviceTokenJSON.RefreshToken, deviceRegisterResponse.DeviceTokenJSON.ExpiresIn)

	go func() {
		errorChan <- nil
	}()
}

func (device *device) refresh(refreshToken string) error {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.clientId)
	values.Set("f", "json")
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", refreshToken)

	// make request
	var refreshResponse deviceRefreshResponse
	if err := agoPost(ago_token_route, []byte(values.Encode()), &refreshResponse); err != nil {
		return err
	}

	// store the new access token
	device.setAccessToken(refreshResponse.AccessToken)
	device.setExpiresAt(refreshResponse.ExpiresIn)

	return nil
}
