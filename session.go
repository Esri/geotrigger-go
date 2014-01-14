package geotrigger_golang

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
)

// The following are vars so that they can be changed by tests
var (
	geotrigger_base_url = "https://geotrigger.arcgis.com"
	ago_base_url        = "https://www.arcgis.com"
)

const ago_token_route = "/sharing/oauth2/token"
const ago_register_route = "/sharing/oauth2/registerDevice"

// The Session type obfuscates whether we are a device or an application,
// both of which implement the interface slightly differently.
type Session interface {
	// The method to use for making requests!
	// `responseJSON` can be a struct modeling the expected JSON, or an arbitrary JSON map (map[string]interface{})
	// that can be used with the helper method `GetValueFromJSONObject`.
	// The channel that is returned will be written to once. If the read value is a nil,
	// then the provided responseJSON has been successfully inflated and is ready for use.
	// Otherwise, the error will contain information about what went wrong.
	Request(string, map[string]interface{}, interface{}) chan error
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
	GetSessionInfo() map[string]string
	requestAccess(chan error)
}

type ErrorResponse struct {
	Error ErrorJSON `json:"error"`
}

type ErrorJSON struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// func type for passing in to `post`. called when we get a 498 invalid token
type refreshHandler func() (string, error)

func agoPost(route string, body []byte, responseJSON interface{}) error {
	req, err := http.NewRequest("POST", ago_base_url+route, bytes.NewReader(body))
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating AgoPost for route %s. %s", route, err))
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return post(req, body, responseJSON, func() (string, error) {
		return "", errors.New("Expired token response from AGO. This is basically a 500.")
	})
}

func geotriggerPost(route string, body []byte, responseJSON interface{}, accessToken string,
	refreshFunc refreshHandler) error {
	req, err := http.NewRequest("POST", geotrigger_base_url+route, bytes.NewReader(body))
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating GeotriggerPost for route %s. %s", route, err))
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	return post(req, body, responseJSON, refreshFunc)
}

func post(req *http.Request, body []byte, responseJSON interface{}, refreshFunc refreshHandler) error {
	path := req.URL.Path

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New(fmt.Sprintf("Error while posting to: %s.  Error: %s", path, err))
	}

	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Received status code %d from %s.", resp.StatusCode, path))
	}

	contents, err := readResponseBody(resp)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not read response body from %s. %s", path, err))
	}

	if errResponse := errorCheck(contents); errResponse != nil {
		if errResponse.Error.Code == 498 {
			if token, err := refreshFunc(); err == nil {
				// time to refresh!
				// the body of the req cannot be reused, because it has already been read
				// and the standard lib can't rewind the pointer on the same content.
				// so, we have passed the underlying []byte down here so we can
				// make a new reader from it. This is a bit unsafe (we are skipping
				// the NewRequest constructor), but since the data is the same, all should be well.
				var bodyReader io.Reader
				bodyReader = bytes.NewReader(body)
				rc, ok := bodyReader.(io.ReadCloser)
				if !ok && bodyReader != nil {
					rc = ioutil.NopCloser(bodyReader)
				}
				req.Body = rc
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
				return post(req, body, responseJSON, refreshFunc)
			} else {
				return err
			}
		} else {
			return errors.New(fmt.Sprintf("Error from %s, code: %d. Message: %s",
				path, errResponse.Error.Code, errResponse.Error.Message))
		}
	}

	return parseJSONResponse(contents, responseJSON)
}

func readResponseBody(resp *http.Response) (contents []byte, err error) {
	defer resp.Body.Close()
	contents, err = ioutil.ReadAll(resp.Body)
	return
}

func errorCheck(resp []byte) *ErrorResponse {
	var errorContainer ErrorResponse
	if err := json.Unmarshal(resp, &errorContainer); err != nil {
		// Don't return an error here, as it is possible for the response
		// to not be parsed into an ErrorResponse, causing an error to be thrown, but still
		// be valid, ie: the root element of the response is an array.
		// We are just looking to see if we can spot a known server error anyway.
		return nil
	}

	if errorContainer.Error.Code > 0 && len(errorContainer.Error.Message) > 0 {
		return &errorContainer
	}

	return nil
}

func parseJSONResponse(resp []byte, responseJSON interface{}) error {
	t := reflect.TypeOf(responseJSON)
	if t.Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("Provided responseJSON interface should be a pointer (to struct or map)."))
	}

	if err := json.Unmarshal(resp, responseJSON); err != nil {
		return errors.New(fmt.Sprintf("Error parsing response: %s  Error: %s", string(resp), err))
	}

	return nil
}
