package geotrigger_golang

import (
	"net/url"
)

type Device struct {
	ClientId string
	DeviceId string
	AccessToken string
	RefreshToken string
	ExpiresIn int
}

type deviceRegisterResponse struct {
	deviceJSON *deviceJSON `json:"device"`
	deviceTokenJSON *deviceTokenJSON `json:"deviceToken"`
}

type tokenRefreshResponse struct {
	accessToken string `json:"access_token"`
}

type deviceTokenJSON struct {
	accessToken string `json:"access_token"`
	refreshToken string `json:"refresh_token"`
	expiresIn int `json:"expires_in"`
}

type deviceJSON struct {
	deviceId string
}

func (device *Device) RequestAccess() (error) {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.ClientId)
	values.Set("f", "json")

	// make request
	var deviceRegisterResponse deviceRegisterResponse
	if err := agoPost("sharing/oauth2/registerDevice", []byte(values.Encode()), &deviceRegisterResponse); err != nil {
		return err
	}

	device.DeviceId = deviceRegisterResponse.deviceJSON.deviceId
	device.AccessToken = deviceRegisterResponse.deviceTokenJSON.accessToken
	device.RefreshToken = deviceRegisterResponse.deviceTokenJSON.refreshToken
	device.ExpiresIn = deviceRegisterResponse.deviceTokenJSON.expiresIn

	return nil
}

func (device *Device) Refresh() (error) {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.ClientId)
	values.Set("f", "json")
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", device.RefreshToken)

	// make request
	var tokenRefreshResponse tokenRefreshResponse
	if err := agoPost("sharing/oauth2/token", []byte(values.Encode()), &tokenRefreshResponse); err != nil {
		return err
	}

	// store the new access token
	device.AccessToken = tokenRefreshResponse.accessToken

	return nil
}

func (device *Device) GeotriggerAPIRequest(route string, data map[string]interface{}, jsonContainer interface{}) (error) {
	return nil
}
