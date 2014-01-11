package geotrigger_golang

import (
	"net/http"
	"bytes"
	"fmt"
	"errors"
	"io/ioutil"
	"encoding/json"
	"reflect"
)

// The following are vars so that they can be changed by tests
var (
	GEOTRIGGER_BASE_URL = "https://geotrigger.arcgis.com"
	AGO_BASE_URL = "https://www.arcgis.com"
)

const AGO_TOKEN_ROUTE = "/sharing/oauth2/token"
const AGO_REGISTER_ROUTE = "/sharing/oauth2/registerDevice"

type Session interface {
	RequestAccess() (error)
	GeotriggerAPIRequest(route string, params map[string]interface{}, responseJSON interface{}) (error)
}

type ErrorResponse struct {
	Error ErrorJSON `json:"error"`
}

type ErrorJSON struct {
	Code int `json:"code"`
	Message string `json:"message"`
}

type errorHandler func(*ErrorResponse)(error)

func agoPost(route string, body []byte, responseJSON interface{}) (error) {
	req, err := http.NewRequest("POST", AGO_BASE_URL + route, bytes.NewReader(body))
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating AgoPost for route %s. %s", route, err))
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return post(req, responseJSON, func(errResponse *ErrorResponse) error {
		return errors.New(fmt.Sprintf("Error from AGO, code: %d. Message: %s", errResponse.Error.Code,
			errResponse.Error.Message))
	})
}

func geotriggerPost(route string, body []byte, responseJSON interface{}, accessToken string,
	errHandler errorHandler) (error) {
	req, err := http.NewRequest("POST", GEOTRIGGER_BASE_URL + route, bytes.NewReader(body))
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating GeotriggerPost for route %s. %s", route, err))
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	return post(req, responseJSON, errHandler)
}

func post(req *http.Request, responseJSON interface{}, errHandler errorHandler) (error) {
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

	if errorResponse := errorCheck(contents); errorResponse != nil {
		return errHandler(errorResponse)
	}

	return parseJSONResponse(contents, responseJSON)
}

func readResponseBody(resp *http.Response) (contents []byte, err error) {
	defer resp.Body.Close()
	contents, err = ioutil.ReadAll(resp.Body)
	return
}

func errorCheck(resp []byte) (*ErrorResponse) {
	var errorContainer ErrorResponse
	if err := json.Unmarshal(resp, &errorContainer); err != nil {
		return nil // no recognized error present
	}

	return &errorContainer
}

func parseJSONResponse(resp []byte, responseJSON interface{}) (error) {
	t := reflect.TypeOf(responseJSON)
	if t.Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("Provided responseJSON interface should be a pointer (to struct or map)."))
	}

	if err := json.Unmarshal(resp, responseJSON); err != nil {
		return errors.New(fmt.Sprintf("Error parsing response: %s  Error: %s", string(resp), err))
	}

	return nil
}
