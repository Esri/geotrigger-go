package geotrigger_golang

import "net/url"

type Device struct {
	ClientId string
	DeviceId string
	AccessToken string
	RefreshToken string
	ExpiresIn int
}

func (device *Device) RequestAccess() (err error) {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.ClientId)
	values.Set("f", "json")

	// make request
	resp, err := agoPost("sharing/oauth2/registerDevice", []byte(values.Encode()))
	if err != nil { return err }
	// parse response
	data, err := parseJSONResponse(resp, "device registration")
	if err != nil { return err }
	// check for param errors returned by AGO
	err = checkAgoError(data)
	if err != nil { return err }

	// unpack device tokens & ids
	deviceObject := data["device"].(map[string]interface{})
	device.DeviceId = deviceObject["deviceId"].(string)
	deviceToken := data["deviceToken"].(map[string]interface{})
	device.AccessToken = deviceToken["access_token"].(string)
	device.ExpiresIn = deviceToken["expires_in"].(int)
	device.RefreshToken = deviceToken["refresh_token"].(string)

	return
}

func (device *Device) Refresh() (err error) {
	// prep values
	values := url.Values{}
	values.Set("client_id", device.ClientId)
	values.Set("f", "json")
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", device.RefreshToken)

	// make request
	resp, err := agoPost("sharing/oauth2/token", []byte(values.Encode()))
	if err != nil { return err }
	// parse response
	data, err := parseJSONResponse(resp, "token-refresh")
	if err != nil { return err }
	// check for param errors returned by AGO
	err = checkAgoError(data)
	if err != nil { return err }

	// store the new access token
	device.AccessToken = data["access_token"].(string)

	return
}

func (device *Device) GeotriggerAPIRequest(route string, data map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}
