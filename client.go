package geotrigger_golang

import (
	"errors"
	"fmt"
)

type Client struct {
	session Session
}

// Create and register a new device associated with the provided client_id
func NewDeviceClient(clientId string) (*Client, error) {
	device := &Device {
		ClientId: clientId,
	}

	return getTokens(device)
}

// Pass in an existing device to be used with the client.
// The device will be registered if access and refresh tokens are
// not both present.
func ExistingDeviceClient(device *Device) (*Client, error) {
	if device == nil {
		return nil, errors.New(fmt.Sprintf("Device cannot be nil."))
	}

	if len(device.AccessToken) == 0 ||  len(device.RefreshToken) == 0 {
		// no tokens!
		return getTokens(device)
	}

	return &Client{session: device}, nil
}

// Create and register a new application associated with the provided client_id
// and client_secret.
func NewApplicationClient(clientId string, clientSecret string) (*Client, error) {
	application := &Application {
		ClientId: clientId,
		ClientSecret: clientSecret,
	}

	return getTokens(application)
}

// Pass in an existing application to be used with the client.
// The application will request a token if no access token is present.
func ExistingApplicationClient(application *Application) (*Client, error) {
	if application == nil {
		return nil, errors.New(fmt.Sprintf("Application cannot be nil."))
	}

	if len(application.AccessToken) == 0 {
		// no token!
		return getTokens(application)
	}

	return &Client{session: application}, nil
}

func (client *Client) request(route string, data map[string]interface{}) (map[string]interface{}, error) {
	return client.session.GeotriggerAPIRequest(route, data)
}

func getTokens(session Session) (*Client, error) {
	err := session.RequestAccess()
	if err == nil {
		return &Client{session: session}, nil
	} else {
		return nil, err
	}
}
