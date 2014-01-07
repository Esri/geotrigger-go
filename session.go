package geotrigger_golang

import (
	"net/http"
	"bytes"
	"fmt"
	"errors"
	"io/ioutil"
	"encoding/json"
)

const GEOTRIGGER_BASE_URL = "https://geotrigger.arcgis.com/"
const AGO_BASE_URL = "https://www.arcgis.com/"
const AGO_TOKEN_ROUTE = "sharing/oauth2/token"
const AGO_REGISTER_ROUTE = "sharing/oauth2/registerDevice"

type Session interface {
	RequestAccess() (error)
	GeotriggerAPIRequest(string, map[string]interface{}) (map[string]interface{}, error)
}

func agoPost(route string, body []byte) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", AGO_BASE_URL + route, bytes.NewReader(body))
	if err != nil { return nil, errors.New(fmt.Sprintf("Error creating AgoPost for route %s. %s", route, err)) }
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return http.DefaultClient.Do(req)
}

func parseJSONResponse(resp *http.Response, reqName string) (data map[string]interface{}, err error) {
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
