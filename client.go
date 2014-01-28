// Package geotrigger_golang provides API access to the Geotrigger Service,
// a cloud based system of geofencing and push notifications. The library
// makes it easier to interact with the Geotrigger API as either a Device or
// an Application. This assumes you have a developer account on
// developers.arcgis.com, from which you can create an Application and obtain
// the necessary credentials to use with this golang library.
//
// For more information about the Geotrigger Service, please see:
// https://developers.arcgis.com/en/geotrigger-service/
package geotrigger_golang

// Client manages credentials for an ArcGIS Application or Device based on what
// you pass in to the provided constructors.
type Client struct {
	session
}

// NewApplication creates and registers a new application associated with the
// provided client_id and client_secret.
func NewApplication(clientID string, clientSecret string) (*Client, error) {
	session, err := newApplication(clientID, clientSecret)

	return &Client{session}, err
}

// NewDevice creates and registers a new device associated with the provided client_id.
func NewDevice(clientID string) (*Client, error) {
	session, err := newDevice(clientID)

	return &Client{session}, err
}

// ExistingDevice creates a client using existing device tokens and credentials.
//
// Provided primarily as a way of debugging an active mobile install.
func ExistingDevice(clientID string, deviceID string, accessToken string, expiresIn int64, refreshToken string) *Client {
	device := &device{
		clientID: clientID,
		deviceID: deviceID,
	}

	device.tokenManager = newTokenManager(accessToken, refreshToken, expiresIn)

	return &Client{device}
}

// Request performs API requests given an endpoint route and parameters.
//
// `response` can be a pointer to a struct modeling the expected JSON, or to an
// arbitrary JSON map (`map[string]interface{}`) that can then be used with the
// helper methods `GetValueFromJSONObject` and `GetValueFromJSONArray`.
//
// `route` must start with a slash.
func (client *Client) Request(route string, params map[string]interface{}, response interface{}) error {
	return client.request(route, params, response)
}

// Info returns information about the current session.
//
// If this is an application session, the following keys will be present: `access_token`, `client_id`, `client_secret`.
//
// If this is a device session, the following keys will be present: `access_token`, `refresh_token`, `device_id`, `client_id`.
func (client *Client) Info() map[string]string {
	return client.info()
}
