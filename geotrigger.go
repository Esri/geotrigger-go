package geotrigger_golang

import (
	"fmt"
	"encoding/json"
	"net/http"
	"io/ioutil"
	"bytes"
	"net/url"
	"errors"
)

const AGO_CLIENT_ID = ""
const AGO_CLIENT_SECRET = ""

type Location struct {
	// yyyy-MM-dd'T'HH:mm:ssZ
	Timestamp string `json:"timestamp"`
	Latitude float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy int `json:"accuracy"`
	Speed int `json:"speed"`
	Altitude int `json:"altitude"`
	Bearing float64 `json:"bearing"`
	Battery int `json:"battery"`
	BatteryState string `json:"batteryState"`
	TrackingProfile string `json:"trackingProfile"`
}

type LocationUpdate struct {
	Previous *Location `json:"previous,omitempty"`
	Locations *[]Location `json:"locations"`
}

type Device struct {
	DeviceId string
	AccessToken string
	RefreshToken string
}

func (device *Device) Register() (err error) {
	fmt.Println("Registering new device.")
	resp, err := device.AgoPost("sharing/oauth2/registerDevice", url.Values{})
	if err != nil { return err }
	data, err := parseResponse(resp, "device-register")
	if err != nil { return err }
	err = checkAgoError(data)
	if err != nil { return err }
	deviceObject := data["device"].(map[string]interface{})
	device.DeviceId = deviceObject["deviceId"].(string)
	deviceToken := data["deviceToken"].(map[string]interface{})
	device.AccessToken = deviceToken["access_token"].(string)
	device.RefreshToken = deviceToken["refresh_token"].(string)

	return
}

func (device *Device) Refresh() (err error) {
	v := url.Values{}
	v.Set("grant_type", "refresh_token")
	v.Set("refresh_token", device.RefreshToken)
	resp, err := device.AgoPost("sharing/oauth2/token", v)
	if err != nil { return err }
	data, err := parseResponse(resp, "token-refresh")
	if err != nil { return err }
	err = checkAgoError(data)
	if err != nil { return err }
	device.AccessToken = data["access_token"].(string)

	return
}

func (device *Device) LocationUpdate(update *LocationUpdate) (err error) {
	payload, err := json.Marshal(update)
	if err != nil { return errors.New(fmt.Sprintf("Error stringifying location update. %s", err)) }

	return device.GeotriggerPost("location/update", payload)
}

func (device *Device) UpdateTags(tags []string) (err error) {
	fmt.Println("updating tags on device", device.DeviceId)
	params := make(map[string]interface{})
	params["addTags"] = tags
	data, err := json.Marshal(params)
	if err != nil { return errors.New(fmt.Sprintf("Error marshalling device update. %s", err)) }

	return device.GeotriggerPost("device/update", data)
}

func (device *Device) AgoPost(route string, values url.Values) (resp *http.Response, err error) {
	values.Set("client_id", AGO_CLIENT_ID)
	values.Set("f", "json")
	body := []byte(values.Encode())
	var baseUrl = "http://www.arcgis.com/"
	req, err := http.NewRequest("POST", baseUrl + route, bytes.NewReader(body))
	if err != nil { return nil, errors.New(fmt.Sprintf("Error creating AgoPost for route %s. %s", route, err)) }
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return device.Do(req)
}

func (device *Device) GeotriggerPost(route string, body []byte) (err error) {
	var baseUrl = "http://geotrigger.arcgis.com/"
	req, err := http.NewRequest("POST", baseUrl + route, bytes.NewReader(body))
	if err != nil { return errors.New(fmt.Sprintf("Error creating GeotriggerPost for route %s. %s", route, err)) }
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", device.AccessToken))
	req.Header.Set("Content-Type", "application/json")
	resp, err := device.Do(req)
	if err != nil { return errors.New(fmt.Sprintf("Could not post to %s. %s", route, err)) }
	data, err := parseResponse(resp, route)
	if err != nil { return err }

	error, gotError := data["error"].(map[string]interface {})
	if gotError {
		errorCode, gotCode := error["code"].(int)

		if gotCode && errorCode == 498 {
			err = device.Refresh()
			if err == nil {
				return device.GeotriggerPost(route, body)
			} else {
				return err
			}
		} else {
			return errors.New(fmt.Sprintf("Unexpected error returned from server: %s", data))
		}
	} else {
		fmt.Printf("Successful post to %s for device: %s\n", route, device.DeviceId)
	}

	return
}

func (device *Device) Do(request *http.Request) (resp *http.Response, err error) {
	client := &http.Client{}
	return client.Do(request)
}

func parseResponse(resp *http.Response, reqName string) (data map[string]interface{}, err error) {
	if resp.StatusCode == 200 {
		defer resp.Body.Close()
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil { return nil, errors.New(fmt.Sprintf("Error while unpacking response from %s. %s", reqName, err)) }
		var returnMap map[string]interface{}
		err = json.Unmarshal(contents, &returnMap)
		if err != nil { return nil, errors.New(fmt.Sprintf("Error parsing response from %s into JSON. %s", reqName, err)) }

		return returnMap, nil
	}

	return nil, errors.New(fmt.Sprintf("Received status code %d from %s.", resp.StatusCode, reqName))
}

func checkAgoError(data map[string]interface {}) (err error) {
	error, gotError := data["error"].(map[string]interface {})

	if gotError {
		errorCode, gotCode := error["code"].(int)
		errorMessage, gotMessage := error["error_description"].(string)
		finalMessage := "Error received from AGO."

		if gotCode {
			finalMessage = fmt.Sprintf(finalMessage, "Code:", errorCode)
		}
		if gotMessage {
			finalMessage = fmt.Sprintf(finalMessage, ",", errorMessage)
		}
		return errors.New(finalMessage)
	}

	return
}
