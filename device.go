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

/* consts and structs for channel coordination */
const (
	accessNeeded    = iota
	refreshNeeded   = iota
	refreshComplete = iota
	refreshFailed   = iota
)

type refreshStatusCheck struct {
	purpose int
	resp    chan *refreshStatusResponse
}

type refreshStatusResponse struct {
	token         string
	isAccessToken bool
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
		"access_token":  device.accessToken,
		"refresh_token": device.refreshToken,
		"device_id":     device.deviceId,
		"client_id":     device.clientId,
	}
}

func newDevice(clientId string) (Session, chan error) {
	device := &device{
		clientId:            clientId,
		refreshStatusChecks: make(chan *refreshStatusCheck),
	}

	return device, sessionInit(device)
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

func (device *device) tokenManager() {
	var waitingChecks []*refreshStatusCheck
	refreshInProgress := false
	for {
		statusCheck := <-device.refreshStatusChecks

		switch {
		case statusCheck.purpose == refreshFailed:
			nextAttempt := waitingChecks[0]
			waitingChecks = waitingChecks[1:]

			if nextAttempt.purpose == refreshNeeded {
				refreshInProgress = true
				go device.refreshApproved(nextAttempt)
			} else if nextAttempt.purpose == accessNeeded {
				refreshInProgress = false
				go device.accessApproved(nextAttempt)
			}
		case statusCheck.purpose == refreshComplete:
			if !refreshInProgress {
				fmt.Println("Warning: refresh completed when we assumed none were occurring.")
			}
			refreshInProgress = false

			// copy status checks to new slice for iterating
			currentWaitingChecks := make([]*refreshStatusCheck, len(waitingChecks))
			copy(currentWaitingChecks, waitingChecks)

			// clear main status checks slice (as we might get more added shortly)
			waitingChecks = waitingChecks[:0]

			for _, waitingCheck := range currentWaitingChecks {
				// get a ref to the channel from outside the routine,
				// otherwise, the loop will have moved forward by the time you
				// access the channel. `range` reuses memory addresses for each
				// iteration, so you would end up using the channel for a different
				// object in the array.
				currentResp := waitingCheck.resp
				go func() {
					currentResp <- &refreshStatusResponse{
						token:         device.accessToken,
						isAccessToken: true,
					}
				}()
			}
		case refreshInProgress:
			waitingChecks = append(waitingChecks, statusCheck)
		case statusCheck.purpose == refreshNeeded:
			refreshInProgress = true
			go device.refreshApproved(statusCheck)
		case statusCheck.purpose == accessNeeded:
			go device.accessApproved(statusCheck)
		}
	}
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
			go func() {
				device.refreshStatusChecks <- refreshSuccess
			}()

			return accessToken, nil
		} else {
			refreshFailure := &refreshStatusCheck{
				purpose: refreshFailed,
				resp:    nil,
			}
			go func() {
				device.refreshStatusChecks <- refreshFailure
			}()

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

func (device *device) accessApproved(statusCheck *refreshStatusCheck) {
	statusCheck.resp <- &refreshStatusResponse{
		token:         device.accessToken,
		isAccessToken: true,
	}
}

func (device *device) refreshApproved(statusCheck *refreshStatusCheck) {
	statusCheck.resp <- &refreshStatusResponse{
		token:         device.refreshToken,
		isAccessToken: false,
	}
}
