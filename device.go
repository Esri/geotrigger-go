package geotrigger_golang

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

type device struct {
	TokenManager
	clientId      string
	deviceId      string
	expiresIn     int
}

/* Device JSON structs */
type DeviceRegisterResponse struct {
	DeviceJSON      DeviceJSON      `json:"device"`
	DeviceTokenJSON DeviceTokenJSON `json:"deviceToken"`
}

type DeviceTokenJSON struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type DeviceJSON struct {
	DeviceId string `json:"deviceId"`
}

type TokenRefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (device *device) Request(route string, params map[string]interface{}, responseJSON interface{}) chan error {
	errorChan := make(chan error)
	go device.request(route, params, responseJSON, errorChan)
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
		clientId:      clientId,
	}

	errorChan := make(chan error)
	go device.requestAccess(errorChan)

	return device, errorChan
}

func (device *device) requestAccess(errorChan chan error) {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.clientId)
	values.Set("f", "json")

	// make request
	var deviceRegisterResponse DeviceRegisterResponse
	if err := agoPost(ago_register_route, []byte(values.Encode()), &deviceRegisterResponse); err != nil {
		go func() {
			errorChan <- err
		}()
		return
	}

	device.deviceId = deviceRegisterResponse.DeviceJSON.DeviceId
	device.expiresIn = deviceRegisterResponse.DeviceTokenJSON.ExpiresIn
	device.TokenManager = newTokenManager(deviceRegisterResponse.DeviceTokenJSON.AccessToken,
		deviceRegisterResponse.DeviceTokenJSON.RefreshToken)

	go device.manageTokens()

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
	var tokenRefreshResponse TokenRefreshResponse
	if err := agoPost(ago_token_route, []byte(values.Encode()), &tokenRefreshResponse); err != nil {
		return err
	}

	// store the new access token
	device.setAccessToken(tokenRefreshResponse.AccessToken)
	device.expiresIn = tokenRefreshResponse.ExpiresIn

	return nil
}

func (device *device) request(route string, params map[string]interface{},
	responseJSON interface{}, errorChan chan error) {
	payload, err := json.Marshal(params)
	if err != nil {
		go func() {
			errorChan <- errors.New(fmt.Sprintf("Error while marshaling params into JSON for route: %s. %s",
				route, err))
		}()
		return
	}

	// This func gets a blocking call if we get a 498 from the geotrigger server
	var refreshFunc refreshHandler = func() (string, error) {
		tr := newTokenRequest(refreshNeeded, true)
		go device.tokenRequest(tr)

		tokenResp := <-tr.tokenResponses

		if tokenResp.isAccessToken {
			return tokenResp.token, nil
		}


		error := device.refresh(tokenResp.token)
		var refreshResult *tokenRequest
		var accessToken string
		if error == nil {
			accessToken = device.getAccessToken()
			refreshResult = newTokenRequest(refreshComplete, false)
		} else {
			refreshResult = newTokenRequest(refreshFailed, false)
		}

		go device.tokenRequest(refreshResult)

		return accessToken, error
	}

	tr := newTokenRequest(accessNeeded, true)
	go device.tokenRequest(tr)

	tokenResp := <-tr.tokenResponses
	err = geotriggerPost(route, payload, responseJSON, tokenResp.token, refreshFunc)

	go func() {
		errorChan <- err
	}()
}
