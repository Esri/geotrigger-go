package geotrigger

import (
	"encoding/json"
	"fmt"
	"github.com/Esri/geotrigger-go/geotrigger/test"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestApplicationAccessRequestFail(t *testing.T) {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		test.Refute(t, r, nil)
		test.Expect(t, r.URL.Path, "/sharing/oauth2/token")
		test.Expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		test.Expect(t, len(vals), 4)
		test.Expect(t, vals.Get("client_id"), "bad_client_id")
		test.Expect(t, vals.Get("f"), "json")
		test.Expect(t, vals.Get("grant_type"), "client_credentials")
		test.Expect(t, vals.Get("client_secret"), "bad_client_secret")
		fmt.Fprintln(res, `{"error":{"code":999,"error":"invalid_request","error_description":"Invalid client_id","message":"invalid_request","details":[]}}`)
	}))
	defer agoServer.Close()

	application := &application{
		clientID:     "bad_client_id",
		clientSecret: "bad_client_secret",
		env:          testEnv("", agoServer.URL),
	}

	expectedErrorMessage := "Error from /sharing/oauth2/token, code: 999. Message: invalid_request"
	err := application.requestAccess()
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), expectedErrorMessage)
}

func TestApplicationRegisterSuccess(t *testing.T) {
	client := getValidApplicationClient(t)
	sessionInfo := client.Info()
	test.Expect(t, sessionInfo["access_token"], "good_access_token")
	test.Expect(t, sessionInfo["client_id"], "good_client_id")
	test.Expect(t, sessionInfo["client_secret"], "good_client_secret")
}

func TestApplicationTokenRefresh(t *testing.T) {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		test.Refute(t, r, nil)
		test.Expect(t, r.URL.Path, "/sharing/oauth2/token")
		test.Expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		test.Expect(t, len(vals), 4)
		test.Expect(t, vals.Get("client_id"), "good_client_id")
		test.Expect(t, vals.Get("f"), "json")
		test.Expect(t, vals.Get("grant_type"), "client_credentials")
		test.Expect(t, vals.Get("client_secret"), "good_client_secret")
		fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":7200}`)
	}))
	defer agoServer.Close()

	testApplication := &application{
		tokenManager: newTokenManager("old_access_token", "", 7200),
		clientID:     "good_client_id",
		clientSecret: "good_client_secret",
		env:          testEnv("", agoServer.URL),
	}
	expiresAt := time.Now().Unix() + 7200 - 60

	err := testApplication.refresh("")
	test.Expect(t, err, nil)
	test.Expect(t, testApplication.getExpiresAt(), expiresAt)
	test.Expect(t, testApplication.getAccessToken(), "refreshed_access_token")
	test.Expect(t, testApplication.clientSecret, "good_client_secret")
	test.Expect(t, testApplication.getRefreshToken(), "")
}

func TestApplicationFullWorkflowWithRefresh(t *testing.T) {
	// a test server to represent the geotrigger server
	gtServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		test.Refute(t, r, nil)
		test.Expect(t, r.URL.Path, "/some/route")
		test.Expect(t, r.Header.Get("Content-Type"), "application/json")
		test.Expect(t, r.Header.Get("X-GT-Client-Name"), "geotrigger-go")
		test.Expect(t, r.Header.Get("X-GT-Client-Version"), version)
		accessToken := r.Header.Get("Authorization")
		test.Expect(t, strings.Index(accessToken, "Bearer "), 0)
		accessToken = strings.Split(accessToken, " ")[1]
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		var params map[string]interface{}
		_ = json.Unmarshal(contents, &params)
		test.Expect(t, len(params), 1)
		test.Expect(t, params["tags"], "derp")

		if accessToken == "old_access_token" {
			fmt.Fprintln(res, `{"error":{"type":"invalidHeader","message":"invalid header or header value","headers":{"Authorization":[{"type":"invalid","message":"Invalid token."}]},"code":498}}`)
		} else if accessToken == "refreshed_access_token" {
			fmt.Fprintln(res, `{"triggers":[{"triggerId":"6fd01180fa1a012f27f1705681b27197","condition":{"direction":"enter","geo":{"geocode":"920 SW 3rd Ave, Portland, OR","driveTime":600,"context":{"locality":"Portland","region":"Oregon","country":"USA","zipcode":"97204"}}},"action":{"message":"Welcome to Portland - The Mayor","callback":"http://pdx.gov/welcome"},"tags":["foodcarts","citygreetings"]}],"boundingBox":{"xmin":-122.68,"ymin":45.53,"xmax":-122.45,"ymax":45.6}}`)
		} else {
			t.Error(fmt.Sprintf("Unexpected access token: %s", accessToken))
		}
	}))
	defer gtServer.Close()

	// a test server to represent AGO
	var tokenReqCount int
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		tokenReqCount++
		test.Refute(t, r, nil)
		test.Expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		test.Expect(t, len(vals), 4)
		test.Expect(t, vals.Get("client_id"), "good_client_id")
		test.Expect(t, vals.Get("f"), "json")
		test.Expect(t, vals.Get("grant_type"), "client_credentials")
		test.Expect(t, vals.Get("client_secret"), "good_client_secret")

		if tokenReqCount == 1 {
			fmt.Fprintln(res, `{"access_token":"old_access_token","expires_in":1800}`)
		} else if tokenReqCount == 2 {
			fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":1800}`)
		} else {
			t.Error("Too many requests for application token (should only have been 2).")
		}
	}))
	defer agoServer.Close()

	application := &application{
		clientID:     "good_client_id",
		clientSecret: "good_client_secret",
		env:          testEnv(gtServer.URL, agoServer.URL),
	}
	err := application.requestAccess()
	test.Expect(t, err, nil)

	client := &Client{application}

	params := map[string]interface{}{
		"tags": "derp",
	}
	var responseJSON map[string]interface{}

	err = client.Request("/some/route", params, &responseJSON)
	test.Expect(t, err, nil)
	test.Expect(t, responseJSON["triggers"].([]interface{})[0].(map[string]interface{})["triggerId"], "6fd01180fa1a012f27f1705681b27197")
	test.Expect(t, responseJSON["boundingBox"].(map[string]interface{})["xmax"], -122.45)
}

func TestApplicationConcurrentRefreshWaitingAtAccessStep(t *testing.T) {
	// This will spawn 4 go routines making requests with bad tokens.
	// The first routine will fire away immediately, get the invalid token response
	// from the geotrigger server, ask for permission to refresh, and start refreshing the token.
	// After a delay, the other 3 routines will ask to use the access token,
	// and end up waiting because a refresh is in progress.
	// After the first routine successfully refreshes the token, the waiting
	// routines will be give the message to continue by using the new access token.
	bt, gt := testConcurrentRefresh(t, getValidApplicationClient(t), "client_credentials", "good_client_secret", "", true, false)
	test.Expect(t, bt, 1)
	test.Expect(t, gt, 4)
}

func TestApplicationConcurrentRefreshWaitingAtRefreshStep(t *testing.T) {
	// This will spawn 4 go routines making requests with bad tokens.
	// Each routine will get permissions to present the access token to
	// the geotrigger server.
	// Whichever routine arrives first will receive the invalid token response will then ask for
	// permission to refresh the token, and be granted that permission.
	// The other 3 routines will also ask for permission to refresh, and instead
	// end up waiting for a reply to that request.
	// After the first routine successfully refreshes the token, the waiting
	// routines will be give the message to continue, but not refresh, and
	// instead use the new access token.
	bt, gt := testConcurrentRefresh(t, getValidApplicationClient(t), "client_credentials", "good_client_secret", "", false, false)
	test.Expect(t, bt, 4)
	test.Expect(t, gt, 4)
}

func TestApplicationRecoveryFromErrorDuringRefreshWithRoutinesWaitingForAccess(t *testing.T) {
	// This will spawn 4 go routines making requests with bad tokens.
	// The first routine will fire away immediately, get the invalid token response
	// from the geotrigger server, ask for permission to refresh, and start refreshing the token.
	// After a delay, the other 3 routines will ask to use the access token,
	// and end up waiting because a refresh is in progress.
	// The first routine will get an error while refreshing, which it will report.
	// The token manager routine will then promote the next routine in line to continue
	// with its actions, prompting another refresh which this time will succeed.
	// That refresh will be communicated to the remaining routines waiting for a token,
	// and they will go ahead and finish.
	bt, gt := testConcurrentRefresh(t, getValidApplicationClient(t), "client_credentials", "good_client_secret", "",
		true, true)
	test.Expect(t, bt, 1)
	test.Expect(t, gt, 3)
}

func TestApplicationRecoveryFromErrorDuringRefreshWithRoutinesWaitingForRefresh(t *testing.T) {
	// This will spawn 4 go routines making requests with bad tokens.
	// Each routine will get permissions to present the access token to
	// the geotrigger server.
	// Whichever routine arrive first will receive the invalid token response will then ask for
	// permission to refresh the token, and be granted that permission.
	// The other 3 routines will also ask for permission to refresh, and instead
	// end up waiting for a reply to that request.
	// The first routine will get an error while refreshing, which it will report.
	// The token manager routine will then promote the next routine in line to continue
	// with its actions, prompting another refresh which this time will succeed.
	// That refresh will be communicated to the remaining routines waiting for a token,
	// and they will go ahead and finish.
	bt, gt := testConcurrentRefresh(t, getValidApplicationClient(t), "client_credentials", "good_client_secret", "",
		false, true)
	test.Expect(t, bt, 4)
	test.Expect(t, gt, 3)
}

func TestApplicationConcurrentTokenExpirationWaitingAtAccessStep(t *testing.T) {
	ac := getValidApplicationClient(t)
	ac.setExpiresAt(-100)

	bt, gt := testConcurrentRefresh(t, ac, "client_credentials", "good_client_secret", "", true, false)
	test.Expect(t, bt, 0)
	test.Expect(t, gt, 4)
}

func TestApplicationConcurrentTokenExpirationWaitingAtRefreshStep(t *testing.T) {
	ac := getValidApplicationClient(t)
	ac.setExpiresAt(-100)

	bt, gt := testConcurrentRefresh(t, ac, "client_credentials", "good_client_secret", "", false, false)
	test.Expect(t, bt, 0)
	test.Expect(t, gt, 4)
}

func TestApplicationRecoveryFromErrorDuringTokenExpirationWaitingForAccess(t *testing.T) {
	ac := getValidApplicationClient(t)
	ac.setExpiresAt(-100)

	bt, gt := testConcurrentRefresh(t, ac, "client_credentials", "good_client_secret", "", true, true)
	test.Expect(t, bt, 0)
	test.Expect(t, gt, 3)
}

func TestApplicationRecoveryFromErrorDuringTokenExpirationWaitingForRefresh(t *testing.T) {
	ac := getValidApplicationClient(t)
	ac.setExpiresAt(-100)

	bt, gt := testConcurrentRefresh(t, ac, "client_credentials", "good_client_secret", "", false, true)
	test.Expect(t, bt, 0)
	test.Expect(t, gt, 3)
}

func getValidApplicationClient(t *testing.T) *Client {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		test.Refute(t, r, nil)
		test.Expect(t, r.URL.Path, "/sharing/oauth2/token")
		test.Expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		test.Expect(t, len(vals), 4)
		test.Expect(t, vals.Get("client_id"), "good_client_id")
		test.Expect(t, vals.Get("f"), "json")
		test.Expect(t, vals.Get("grant_type"), "client_credentials")
		test.Expect(t, vals.Get("client_secret"), "good_client_secret")
		fmt.Fprintln(res, `{"access_token":"good_access_token","expires_in":7200}`)
	}))
	defer agoServer.Close()

	application := &application{
		clientID:     "good_client_id",
		clientSecret: "good_client_secret",
		env:          testEnv("", agoServer.URL),
	}
	err := application.requestAccess()
	test.Expect(t, err, nil)

	return &Client{application}
}

// A big ugly func that gets called many times for tests. Separated out to avoid duplicating it.
func testConcurrentRefresh(t *testing.T, client *Client, grantType string, clientSecret string,
	refreshToken string, pauseAfterFirstReq bool, errorOnFirstRefresh bool) (int, int) {
	var refreshCount int
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		refreshCount++

		if !errorOnFirstRefresh && refreshCount > 1 {
			t.Error("Too many refresh attempts! Should have only been 1.")
		} else if errorOnFirstRefresh && refreshCount > 2 {
			t.Error("Too many refresh attempts! Should have only been 2.")
		}

		test.Refute(t, r, nil)
		time.Sleep(80 * time.Millisecond)
		test.Expect(t, r.URL.Path, ago_token_route)
		test.Expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		test.Expect(t, len(vals), 4)
		test.Expect(t, vals.Get("client_id"), "good_client_id")
		test.Expect(t, vals.Get("f"), "json")
		test.Expect(t, vals.Get("grant_type"), grantType)
		test.Expect(t, vals.Get("client_secret"), clientSecret)
		test.Expect(t, vals.Get("refresh_token"), refreshToken)
		if refreshCount == 1 && errorOnFirstRefresh {
			fmt.Fprintln(res, `{"error":{"code":498,"message":"Invalid token."}}`)
		} else {
			fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":1800}`)
		}
	}))
	defer agoServer.Close()

	var oldAccessTokenUse, refreshedAccessTokenUse int
	// a test server to represent the geotrigger server
	gtServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		test.Refute(t, r, nil)
		test.Expect(t, r.URL.Path, "/some/route")
		test.Expect(t, r.Header.Get("Content-Type"), "application/json")
		test.Expect(t, r.Header.Get("X-GT-Client-Name"), "geotrigger-go")
		test.Expect(t, r.Header.Get("X-GT-Client-Version"), version)
		accessToken := r.Header.Get("Authorization")
		test.Expect(t, strings.Index(accessToken, "Bearer "), 0)
		accessToken = strings.Split(accessToken, " ")[1]
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		var params map[string]interface{}
		_ = json.Unmarshal(contents, &params)
		test.Expect(t, len(params), 1)
		test.Expect(t, params["tags"], "derp")

		if accessToken == "good_access_token" {
			oldAccessTokenUse++
			fmt.Fprintln(res, `{"error":{"type":"invalidHeader","message":"invalid header or header value","headers":{"Authorization":[{"type":"invalid","message":"Invalid token."}]},"code":498}}`)
		} else if accessToken == "refreshed_access_token" {
			refreshedAccessTokenUse++
			fmt.Fprintln(res, `{"triggers":[{"triggerId":"6fd01180fa1a012f27f1705681b27197","condition":{"direction":"enter","geo":{"geocode":"920 SW 3rd Ave, Portland, OR","driveTime":600,"context":{"locality":"Portland","region":"Oregon","country":"USA","zipcode":"97204"}}},"action":{"message":"Welcome to Portland - The Mayor","callback":"http://pdx.gov/welcome"},"tags":["foodcarts","citygreetings"]}],"boundingBox":{"xmin":-122.68,"ymin":45.53,"xmax":-122.45,"ymax":45.6}}`)
		} else {
			t.Error(fmt.Sprintf("Unexpected access token: %s", accessToken))
		}
	}))
	defer gtServer.Close()

	client.session.setEnv(testEnv(gtServer.URL, agoServer.URL))

	params1 := map[string]interface{}{
		"tags": "derp",
	}
	var responseJSON1 map[string]interface{}
	params2 := map[string]interface{}{
		"tags": "derp",
	}
	var responseJSON2 map[string]interface{}
	params3 := map[string]interface{}{
		"tags": "derp",
	}
	var responseJSON3 map[string]interface{}
	params4 := map[string]interface{}{
		"tags": "derp",
	}
	var responseJSON4 map[string]interface{}

	var w sync.WaitGroup
	var errorCount int
	w.Add(4)
	go func() {
		err := client.Request("/some/route", params1, &responseJSON1)
		if err != nil {
			errorCount++
		}
		w.Done()
	}()
	if pauseAfterFirstReq {
		time.Sleep(20 * time.Millisecond)
	}
	go func() {
		err := client.Request("/some/route", params2, &responseJSON2)
		if err != nil {
			errorCount++
		}
		w.Done()
	}()
	go func() {
		err := client.Request("/some/route", params3, &responseJSON3)
		if err != nil {
			errorCount++
		}
		w.Done()
	}()
	go func() {
		err := client.Request("/some/route", params4, &responseJSON4)
		if err != nil {
			errorCount++
		}
		w.Done()
	}()
	w.Wait()

	if errorOnFirstRefresh {
		test.Expect(t, errorCount, 1)
	} else {
		test.Expect(t, errorCount, 0)
	}

	return oldAccessTokenUse, refreshedAccessTokenUse
}
