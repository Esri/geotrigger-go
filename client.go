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

// The client struct type. Holds onto a Session on your behalf and performs necessary setup.
// Make API requests with the "Request" method. Get info about the current session with the
// "GetSessionInfo" method. See the "Session" interface for further descriptions of those methods.
// This is the type you should use directly for interacting with the geotrigger API.
type Client struct {
	Session
}

// Create and register a new device associated with the provided client_id
// The channel that is returned will be written to once. If the read value is a nil,
// then the returned client pointer has been successfully inflated and is ready for use.
// Otherwise, the error will contain information about what went wrong.
func NewDeviceClient(clientId string) (*Client, chan error) {
	session, errorChan := newDevice(clientId)

	return &Client{Session: session}, errorChan
}

// Create and register a new application associated with the provided client_id
// and client_secret.
// The channel that is returned will be written to once. If the read value is a nil,
// then the returned client pointer has been successfully inflated and is ready for use.
// Otherwise, the error will contain information about what went wrong.
func NewApplicationClient(clientId string, clientSecret string) (*Client, chan error) {
	session, errorChan := newApplication(clientId, clientSecret)

	return &Client{Session: session}, errorChan
}
