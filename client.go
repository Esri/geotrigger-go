package geotrigger_golang

import (
	"errors"
	"fmt"
	"reflect"
)

// The client struct type. Has one, un-exported field for a session that handles
// auth for you. Make API requests with the "Request" method. This is the type
// you should use directly for interacting with the geotrigger API.
type Client struct {
	session session
}

// Create and register a new device associated with the provided client_id
// The channel that is returned will be written to once. If the read value is a nil,
// then the returned client pointer has been successfully inflated and is ready for use.
// Otherwise, the error will contain information about what went wrong.
func NewDeviceClient(clientId string) (*Client, chan error) {
	refreshStatusChecks := make(chan *refreshStatusCheck)
	device := &device{
		clientId:            clientId,
		refreshStatusChecks: refreshStatusChecks,
	}

	return getTokens(device)
}

// Create and register a new application associated with the provided client_id
// and client_secret.
// The channel that is returned will be written to once. If the read value is a nil,
// then the returned client pointer has been successfully inflated and is ready for use.
// Otherwise, the error will contain information about what went wrong.
func NewApplicationClient(clientId string, clientSecret string) (*Client, chan error) {
	application := &application{
		clientId:     clientId,
		clientSecret: clientSecret,
	}

	return getTokens(application)
}

// The method to use for making requests!
// `responseJSON` can be a struct modeling the expected JSON, or an arbitrary JSON map (map[string]interface{})
// that can be used with the helper method `GetValueFromJSONObject`.
// The channel that is returned will be written to once. If the read value is a nil,
// then the provided responseJSON has been successfully inflated and is ready for use.
// Otherwise, the error will contain information about what went wrong.
func (client *Client) Request(route string, params map[string]interface{}, responseJSON interface{}) chan error {
	errorChan := make(chan error)
	go client.session.geotriggerAPIRequest(route, params, responseJSON, errorChan)
	return errorChan
}

// Get info about the current session.
// If this is an application session, the following keys will be present:
// `access_token`
// `client_id`
// `client_secret`
// If this is a device session, the following keys will be present:
// `access_token`
// `refresh_token`
// `device_id`
// `client_id`
func (client *Client) GetSessionInfo() map[string]string {
	return client.session.getSessionInfo()
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
		return errors.New(fmt.Sprintf("Provided reference to value of type %s did not match actual type: %s.",
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
	pv := reflect.ValueOf(value)
	// Elem() gets the value being pointed to,
	v := pv.Elem()
	// and we can set it directly to what we found in the JSON, since we
	// have already checked that they are the same type.
	v.Set(reflect.ValueOf(jsonVal))
	return
}

// Un-exported helper to just DRY up the client constructors above.
func getTokens(session session) (*Client, chan error) {
	errorChan := make(chan error)
	client := &Client{session: session}

	go session.requestAccess(errorChan)
	go session.tokenManager()
	return client, errorChan
}
