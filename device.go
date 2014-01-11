package geotrigger_golang

import (
	"net/url"
	"encoding/json"
	"errors"
	"fmt"
)

type Device struct {
	ClientId string
	DeviceId string
	AccessToken string
	RefreshToken string
	ExpiresIn int
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

func (device *Device) RequestAccess() (error) {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.ClientId)
	values.Set("f", "json")

	// make request
	var deviceRegisterResponse DeviceRegisterResponse
	if err := agoPost(AGO_TOKEN_ROUTE, []byte(values.Encode()), &deviceRegisterResponse); err != nil {
		return err
	}

	device.DeviceId = deviceRegisterResponse.DeviceJSON.DeviceId
	device.AccessToken = deviceRegisterResponse.DeviceTokenJSON.AccessToken
	device.RefreshToken = deviceRegisterResponse.DeviceTokenJSON.RefreshToken
	device.ExpiresIn = deviceRegisterResponse.DeviceTokenJSON.ExpiresIn

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
	var tokenRefreshResponse TokenRefreshResponse
	if err := agoPost(AGO_REGISTER_ROUTE, []byte(values.Encode()), &tokenRefreshResponse); err != nil {
		return err
	}

	// store the new access token
	device.AccessToken = tokenRefreshResponse.AccessToken

	return nil
}

func (device *Device) GeotriggerAPIRequest(route string, params map[string]interface{},
	responseJSON interface{}) (error) {
	payload, err := json.Marshal(params)
	if err != nil {
		return errors.New(fmt.Sprintf("Error while marshaling params into JSON. %s", err))
	}

	// declare func first, so it can be called from within own definition
	var errorHandlerFunc errorHandler
	errorHandlerFunc = func(errResponse *ErrorResponse) error {
		if errResponse.Error.Code == 498 {
			if err = device.Refresh(); err == nil {
				return geotriggerPost(route, payload, responseJSON, device.AccessToken, errorHandlerFunc)
			} else {
				return err
			}
		} else {
			return errors.New(fmt.Sprintf("Error from Geotrigger Service, code: %d. Message: %s",
				errResponse.Error.Code, errResponse.Error.Message))
		}
	}

	return geotriggerPost(route, payload, responseJSON, device.AccessToken, errorHandlerFunc)
}
