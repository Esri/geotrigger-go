package geotrigger_golang

import (
	"errors"
	"fmt"
	"reflect"
)

// The client struct type. Has one, un-exported field for a session that handles
// auth for you. Make API requests with the "request" method. This is the type
// you should use directly for interacting with the geotrigger API.
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
		// missing crucial token(s)!
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

// The method to use for making requests!
// `responseJSON` can be a struct modeling the expected JSON, or an arbitrary JSON map (map[string]interface{})
// that can be used with the helper method `GetValueFromJSONObject`.
// Make sure to check the error return! Errors should be explicit and helpful.
func (client *Client) Request(route string, params map[string]interface{}, responseJSON interface{}) (error) {
	return client.session.GeotriggerAPIRequest(route, params, responseJSON)
}

// A helpful method (with explicit error messages) for unpacking values out of arbitrary JSON objects
func GetValueFromJSONObject(jsonObject map[string]interface{}, key string, value interface{}) (err error) {
	if jsonObject == nil {
		return errors.New("Attempt to get value from a nil JSON object.")
	}

	if len(key) == 0 {
		return errors.New("Attempt to pull value for empty key from JSON object.")
	}

	jsonVal, gotVal := jsonObject[key]
	if !gotVal {
		return errors.New(fmt.Sprintf("No value found for key: %s", key))
	}

	// make sure the interface provided is a pointer, so that we can modify the value
	expectedType := reflect.TypeOf(value)
	if expectedType.Kind() != reflect.Ptr {
		return errors.New("Provided value is of invalid type (must be pointer).")
	}

	// we know it's a pointer, so get the type of value being pointed to
	expectedType = expectedType.Elem()

	// compare that type to the type pulled from the JSON
	actualType := reflect.TypeOf(jsonVal)
	if actualType != expectedType {
		return errors.New(fmt.Sprintf("Provided reference to value of type %s did not match actual type: (%s).",
			expectedType, actualType))
	}

	// recover from any panics that might occur below, although we should be safe
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Panic during assignment from JSON to provided interface value.")
			}
		}
	}()

	// Time to set the new value being pointed to by the passed in interface.
	// We know it's a pointer, so its value will be a reference to the value
	// we are actually interested in changing.
	rv := reflect.ValueOf(value)
	// Elem() gets the value being pointed to,
	trueV := rv.Elem()
	// and we can set it directly to what we found in the JSON, since we
	// have already checked that they are the same type.
	trueV.Set(reflect.ValueOf(jsonVal))
	return
}

// Un-exported helper to just DRY up the client constructors above.
func getTokens(session Session) (*Client, error) {
	err := session.RequestAccess()
	if err == nil {
		return &Client{session: session}, nil
	} else {
		return nil, err
	}
}
