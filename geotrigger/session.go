package geotrigger

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

const (
	geotrigger_base_url = "https://geotrigger.arcgis.com"
	ago_base_url        = "https://www.arcgis.com"
	ago_token_route     = "/sharing/oauth2/token"
	ago_register_route  = "/sharing/oauth2/registerDevice"
	version             = "1.0.0"
)

var defEnv = &environment{
	geotrigger_base_url,
	ago_base_url,
}

// The Session interface obfuscates whether we are a device or an application,
// both of which implement the interface slightly differently.
type session interface {
	request(string, interface{}, interface{}) error
	info() map[string]string
	// A session is also a TokenManager
	tokenManager
	// used internally when token expires
	refresh(string) error
	// used internally for changing URLs at runtime for testing
	setEnv(*environment)
}

type environment struct {
	geotriggerURL string
	agoURL        string
}

type errorResponse struct {
	Error errorJSON `json:"error"`
}

type errorJSON struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// func type for passing in to `post`. called when we get a 498 invalid token
type refreshHandler func() (string, error)

// funcs below manage http specifically for geotrigger service and AGO credentials
func doRefresh(session session, token string) (string, error) {
	error := session.refresh(token)
	var refreshResult *tokenRequest
	var accessToken string
	if error == nil {
		accessToken = session.getAccessToken()
		refreshResult = newTokenRequest(refreshComplete, false)
	} else {
		refreshResult = newTokenRequest(refreshFailed, false)
	}

	go session.tokenRequest(refreshResult)

	return accessToken, error
}

func geotriggerPost(env *environment, session session, route string, params interface{},
	responseJSON interface{}) error {
	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("Error while marshaling params into JSON for route: %s. %s", route, err)
	}

	// This func gets a blocking call if we get a 498 from the geotrigger server
	refreshFunc := func() (string, error) {
		tr := newTokenRequest(refreshNeeded, true)
		go session.tokenRequest(tr)

		tokenResp := <-tr.tokenResponses

		if tokenResp.isAccessToken {
			// refresh request denied, another routine has already refreshed!
			// go ahead and use this access token
			return tokenResp.token, nil
		}

		// refresh request approved, get a fresh token
		return doRefresh(session, tokenResp.token)
	}

	tr := newTokenRequest(accessNeeded, true)
	go session.tokenRequest(tr)

	tokenResp := <-tr.tokenResponses

	var token string
	if tokenResp.isAccessToken {
		// we have access, go ahead and use it
		token = tokenResp.token
	} else {
		// access request denied, the token has expired. go get a fresh one
		token, err = doRefresh(session, tokenResp.token)
	}

	if err != nil {
		return fmt.Errorf("Error while trying to refresh token before hitting route: %s. %s",
			route, err)
	}

	req, err := http.NewRequest("POST", routeConcat(env.geotriggerURL, route), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("Error creating GeotriggerPost for route %s. %s", route, err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GT-Client-Name", "geotrigger-go")
	req.Header.Set("X-GT-Client-Version", version)

	return post(req, body, responseJSON, refreshFunc)
}

func agoPost(env *environment, route string, body []byte, responseJSON interface{}) error {
	req, err := http.NewRequest("POST", routeConcat(env.agoURL, route), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("Error creating AgoPost for route %s. %s", route, err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return post(req, body, responseJSON, func() (string, error) {
		return "", errors.New("Expired token response from AGO. This is basically a 500.")
	})
}

func post(req *http.Request, body []byte, responseJSON interface{}, refreshFunc refreshHandler) error {
	path := req.URL.Path

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error while posting to: %s. Error: %s", path, err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Received status code %d from %s.", resp.StatusCode, path)
	}

	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Could not read response body from %s. %s", path, err)
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
			return fmt.Errorf("Error from %s, code: %d. Message: %s",
				path, errResponse.Error.Code, errResponse.Error.Message)
		}
	}

	return parseJSONResponse(contents, responseJSON)
}

func errorCheck(resp []byte) *errorResponse {
	var errorContainer errorResponse
	if err := json.Unmarshal(resp, &errorContainer); err != nil {
		// Don't return an error here, as it is possible for the response
		// to not be parsed into an errorResponse, causing an error to be thrown, but still
		// be valid, ie: the root element of the response is an array.
		// We are just looking to see if we can spot a known server error anyway.
		return nil
	}

	// We don't inspect the value of `Error.Code` here because the server may not return a code value
	if len(errorContainer.Error.Message) > 0 {
		return &errorContainer
	}

	return nil
}

func parseJSONResponse(resp []byte, responseJSON interface{}) error {
	t := reflect.TypeOf(responseJSON)
	if t == nil || t.Kind() != reflect.Ptr {
		return fmt.Errorf("Provided responseJSON interface should be a pointer (to struct or map).")
	}

	if err := json.Unmarshal(resp, responseJSON); err != nil {
		return fmt.Errorf("Error parsing response: %s  Error: %s", string(resp), err)
	}

	return nil
}

func routeConcat(baseURL, route string) string {
	var buffer bytes.Buffer
	buffer.WriteString(baseURL)

	if route[:1] != "/" {
		buffer.WriteString("/")
	}

	buffer.WriteString(route)

	return buffer.String()
}

func testEnv(gtURL, agoURL string) *environment {
	return &environment{gtURL, agoURL}
}
