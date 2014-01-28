package geotrigger_golang

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// The following are vars so that they can be changed by tests
var (
	geotrigger_base_url = "https://geotrigger.arcgis.com"
	ago_base_url        = "https://www.arcgis.com"
)

const (
	ago_token_route    = "/sharing/oauth2/token"
	ago_register_route = "/sharing/oauth2/registerDevice"
	version            = "0.1.0"
)

// The Session interface obfuscates whether we are a device or an application,
// both of which implement the interface slightly differently.
type session interface {
	request(string, map[string]interface{}, interface{}) error
	info() map[string]string
	// A session is also a TokenManager
	tokenManager
	// used internally when token expires
	refresh(string) error
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

func geotriggerPost(session session, route string, params map[string]interface{}, responseJSON interface{}) error {
	body, err := json.Marshal(params)
	if err != nil {
		return errors.New(fmt.Sprintf("Error while marshaling params into JSON for route: %s. %s",
			route, err))
	}

	// This func gets a blocking call if we get a 498 from the geotrigger server
	var refreshFunc refreshHandler = func() (string, error) {
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
		return errors.New(fmt.Sprintf("Error while trying to refresh token before hitting route: %s. %s",
			route, err))
	}

	req, err := http.NewRequest("POST", geotrigger_base_url+route, bytes.NewReader(body))
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating GeotriggerPost for route %s. %s", route, err))
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GT-Client-Name", "geotrigger_golang")
	req.Header.Set("X-GT-Client-Version", version)

	return post(req, body, responseJSON, refreshFunc)
}

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

func post(req *http.Request, body []byte, responseJSON interface{}, refreshFunc refreshHandler) error {
	path := req.URL.Path

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New(fmt.Sprintf("Error while posting to: %s.  Error: %s", path, err))
	}

	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Received status code %d from %s.", resp.StatusCode, path))
	}

	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
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
