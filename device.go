package geotrigger_golang

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

type device struct {
	clientId            string
	deviceId            string
	accessToken         string
	refreshToken        string
	expiresIn           int
	refreshStatusChecks chan *refreshStatusCheck
}

const (
	accessNeeded    = iota
	refreshNeeded   = iota
	refreshComplete = iota
)

type refreshStatusCheck struct {
	purpose int
	resp    chan *refreshStatusResponse
}

type refreshStatusResponse struct {
	token         string
	isAccessToken bool
}

type DeviceRegisterResponse struct {
	DeviceJSON      DeviceJSON      `json:"device"`
	DeviceTokenJSON DeviceTokenJSON `json:"deviceToken"`
}

type TokenRefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type DeviceTokenJSON struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type DeviceJSON struct {
	DeviceId string `json:"deviceId"`
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
	device.accessToken = deviceRegisterResponse.DeviceTokenJSON.AccessToken
	device.refreshToken = deviceRegisterResponse.DeviceTokenJSON.RefreshToken
	device.expiresIn = deviceRegisterResponse.DeviceTokenJSON.ExpiresIn

	go func() {
		errorChan <- nil
	}()
}

func (device *device) refresh() error {
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
	device.expiresIn = tokenRefreshResponse.ExpiresIn

	return nil
}

func (device *device) geotriggerAPIRequest(route string, params map[string]interface{},
	responseJSON interface{}, errorChan chan error) {
	payload, err := json.Marshal(params)
	if err != nil {
		go func() {
			errorChan <- errors.New(fmt.Sprintf("Error while marshaling params into JSON for route: %s. %s",
				route, err))
		}()
		return
	}

	var refreshFunc refreshHandler = func() (string, error) {
		statusCheck := &refreshStatusCheck{
			purpose: refreshNeeded,
			resp:    make(chan *refreshStatusResponse),
		}
		device.refreshStatusChecks <- statusCheck

		statusResp := <-statusCheck.resp

		if statusResp.isAccessToken {
			return statusResp.token, nil
		}

		if err = device.refresh(); err == nil {
			accessToken := device.accessToken
			refreshSuccess := &refreshStatusCheck{
				purpose: refreshComplete,
				resp:    nil,
			}
			device.refreshStatusChecks <- refreshSuccess

			return accessToken, nil
		} else {
			return "", err
		}
	}

	statusCheck := &refreshStatusCheck{
		purpose: accessNeeded,
		resp:    make(chan *refreshStatusResponse),
	}
	device.refreshStatusChecks <- statusCheck

	statusResp := <-statusCheck.resp

	err = geotriggerPost(route, payload, responseJSON, statusResp.token, refreshFunc)

	go func() {
		errorChan <- err
	}()
}

func (device *device) getSessionInfo() map[string]string {
	return map[string]string {
		"access_token":  device.accessToken,
		"refresh_token": device.refreshToken,
		"device_id":     device.deviceId,
		"client_id":     device.clientId,
	}
}

func (device *device) tokenManager() {
	waitingChecks := make([]*refreshStatusCheck, 10)
	refreshInProgress := false
	for {
		statusCheck := <-device.refreshStatusChecks
		if refreshInProgress {
			waitingChecks = append(waitingChecks, statusCheck)
		} else if statusCheck.purpose == refreshComplete {
			if !refreshInProgress {
				fmt.Println("Warning: refresh completed when we assumed none were occurring.")
			}
			refreshInProgress = false

			// copy status checks to new slice for iterating
			currentWaitingChecks := make([]*refreshStatusCheck, len(waitingChecks))
			copy(currentWaitingChecks, waitingChecks)

			// clear main status checks slice (as we might get more added shortly)
			waitingChecks = append([]*refreshStatusCheck(nil), waitingChecks[:0]...)

			for _, waitingCheck := range currentWaitingChecks {
				go func() {
					waitingCheck.resp <- &refreshStatusResponse{
						token: device.accessToken, isAccessToken: true}
				}()
			}
		} else if statusCheck.purpose == refreshNeeded {
			refreshInProgress = true
			statusCheck.resp <- &refreshStatusResponse{
				token: device.refreshToken, isAccessToken: false}
		} else if statusCheck.purpose == accessNeeded {
			statusCheck.resp <- &refreshStatusResponse{
				token: device.accessToken, isAccessToken: true}
		}
	}
}
