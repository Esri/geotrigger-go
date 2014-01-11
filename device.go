package geotrigger_golang

import (
	"net/url"
	"encoding/json"
	"errors"
	"fmt"
)

type Device struct {
	clientId string
	deviceId string
	accessToken string
	refreshToken string
	expiresIn int
	refreshInProgress bool
	shouldRefresh chan bool
}

type DeviceRegisterResponse struct {
	DeviceJSON DeviceJSON `json:"device"`
	DeviceTokenJSON DeviceTokenJSON `json:"deviceToken"`
}

type TokenRefreshResponse struct {
	AccessToken string `json:"access_token"`
}

type DeviceTokenJSON struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn int `json:"expires_in"`
}

type DeviceJSON struct {
	DeviceId string `json:"deviceId"`
}

func (device *Device) requestAccess(errorChan chan error) {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.clientId)
	values.Set("f", "json")

	// make request
	var deviceRegisterResponse DeviceRegisterResponse
	if err := agoPost(ago_register_route, []byte(values.Encode()), &deviceRegisterResponse); err != nil {
		errorChan <- err
		return
	}

	device.deviceId = deviceRegisterResponse.DeviceJSON.DeviceId
	device.accessToken = deviceRegisterResponse.DeviceTokenJSON.AccessToken
	device.refreshToken = deviceRegisterResponse.DeviceTokenJSON.RefreshToken
	device.expiresIn = deviceRegisterResponse.DeviceTokenJSON.ExpiresIn

	errorChan <- nil
}

func (device *Device) refresh() (error) {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.clientId)
	values.Set("f", "json")
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", device.refreshToken)

	// make request
	var tokenRefreshResponse TokenRefreshResponse
	if err := agoPost(ago_token_route, []byte(values.Encode()), &tokenRefreshResponse); err != nil {
		return err
	}

	// store the new access token
	device.accessToken = tokenRefreshResponse.AccessToken

	return nil
}

func (device *Device) geotriggerAPIRequest(route string, params map[string]interface{},
	responseJSON interface{}, errorChan chan error) {
	payload, err := json.Marshal(params)
	if err != nil {
		errorChan <- errors.New(fmt.Sprintf("Error while marshaling params into JSON. %s", err))
		return
	}

	var refreshFunc refreshHandler = func() (string, error) {
		if err = device.refresh(); err == nil {
			return device.accessToken, nil
		} else {
			return "", err
		}
	}

	err = geotriggerPost(route, payload, responseJSON, device.accessToken, refreshFunc)
	errorChan <- err
}

func (device *Device) getAccessToken() (string) {
	return device.accessToken
}

func (device *Device) getRefreshToken() (string) {
	return device.refreshToken
}
