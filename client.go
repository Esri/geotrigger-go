// Package geotrigger_golang provides API access to the Geotrigger Service,
// a cloud based system of geofencing and push notifications. The library
// makes it easier to interact with the Geotrigger API as either a Device or
// an Application. This assumes you have a developer account on
// developers.arcgis.com, from which you can create an Application and obtain
// the necessary credentials to use with this golang library.
//
// For more information about the Geotrigger Service, please look here:
// https://developers.arcgis.com/en/geotrigger-service/
//
// Documentation for this library can be found on github:
// https://github.com/Esri/geotrigger_golang
package geotrigger_golang

// The client manages credentials for an ArcGIS Application or Device based on what you pass in to the
// provided constructors.
//
// Make API requests with the "Request" method. Get info about the current session with the
// "Info" method.
//
// This is the type you should use directly for interacting with the Geotrigger API.
type Client struct {
	session
}

// Create and register a new application associated with the provided client_id
// and client_secret.
func NewApplication(clientId string, clientSecret string) (*Client, error) {
	session, err := newApplication(clientId, clientSecret)

	return &Client{session}, err
}

// Create and register a new device associated with the provided client_id.
func NewDevice(clientId string) (*Client, error) {
	session, err := newDevice(clientId)

	return &Client{session}, err
}

// Inflate a client using existing device tokens and credentials obtained elsewhere.
//
// Provided primarily as a way of debugging an active mobile install.
func ExistingDevice(clientId string, deviceId string, accessToken string, expiresIn int64, refreshToken string) *Client {
	device := &device{
		clientId: clientId,
		deviceId: deviceId,
	}

	device.tokenManager = newTokenManager(accessToken, refreshToken, expiresIn)

	return &Client{device}
}

// The method to use for making requests!
//
// `response` can be a pointer to a struct modeling the expected JSON, or to an arbitrary JSON map (map[string]interface{})
// that can then be used with the helper methods `GetValueFromJSONObject` and `GetValueFromJSONArray`.
//
// `route` should start with a slash.
func (client *Client) Request(route string, params map[string]interface{}, response interface{}) error {
	return client.request(route, params, response)
}

// Get info about the current session.
//
// If this is an application session, the following keys will be present: `access_token`, `client_id`, `client_secret`.
//
// If this is a device session, the following keys will be present: `access_token`, `refresh_token`, `device_id`, `client_id`.
func (client *Client) Info() map[string]string {
	return client.info()
}
