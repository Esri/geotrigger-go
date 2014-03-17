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
	"testing"
	"time"
)

func TestDeviceRegisterFail(t *testing.T) {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		test.Refute(t, r, nil)
		test.Expect(t, r.URL.Path, "/sharing/oauth2/registerDevice")
		test.Expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		test.Expect(t, len(vals), 2)
		test.Expect(t, vals.Get("client_id"), "bad_client_id")
		test.Expect(t, vals.Get("f"), "json")
		fmt.Fprintln(res, `{"error":{"code":999,"message":"Unable to register device.","details":["'client_id' invalid"]}}`)
	}))
	defer agoServer.Close()

	device := &device{
		clientID: "bad_client_id",
		env:      testEnv("", agoServer.URL),
	}
	expectedErrorMessage := "Error from /sharing/oauth2/registerDevice, code: 999. Message: Unable to register device."
	err := device.register()

	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), expectedErrorMessage)
}

func TestDeviceRegisterSuccess(t *testing.T) {
	client := getValidDeviceClient(t)
	sessionInfo := client.Info()
	test.Expect(t, sessionInfo["access_token"], "good_access_token")
	test.Expect(t, sessionInfo["refresh_token"], "good_refresh_token")
	test.Expect(t, sessionInfo["device_id"], "device_id")
	test.Expect(t, sessionInfo["client_id"], "good_client_id")
}

func TestDeviceTokenRefresh(t *testing.T) {
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
		test.Expect(t, vals.Get("grant_type"), "refresh_token")
		test.Expect(t, vals.Get("refresh_token"), "good_refresh_token")
		fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":1800}`)
	}))
	defer agoServer.Close()

	testDevice := &device{
		tokenManager: newTokenManager("old_access_token", "good_refresh_token", 1800),
		clientID:     "good_client_id",
		deviceID:     "device_id",
		env:          testEnv("", agoServer.URL),
	}
	expiresAt := time.Now().Unix() + 1800 - 60

	err := testDevice.refresh("good_refresh_token")
	test.Expect(t, err, nil)
	test.Expect(t, testDevice.getExpiresAt(), expiresAt)
	test.Expect(t, testDevice.getAccessToken(), "refreshed_access_token")
	test.Expect(t, testDevice.clientID, "good_client_id")
	test.Expect(t, testDevice.getRefreshToken(), "good_refresh_token")
}

func TestDeviceFullWorkflowWithRefresh(t *testing.T) {
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
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		test.Refute(t, r, nil)
		test.Expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))

		if r.URL.Path == ago_register_route {
			test.Expect(t, len(vals), 2)
			test.Expect(t, vals.Get("client_id"), "good_client_id")
			test.Expect(t, vals.Get("f"), "json")
			fmt.Fprintln(res, `{"device":{"deviceID":"device_id","client_id":"good_client_id","apnsProdToken":null,"apnsSandboxToken":null,"gcmRegistrationId":null,"registered":1389531528000,"lastAccessed":1389531528000},"deviceToken":{"access_token":"old_access_token","expires_in":1799,"refresh_token":"good_refresh_token"}}`)
		} else if r.URL.Path == ago_token_route {
			test.Expect(t, len(vals), 4)
			test.Expect(t, vals.Get("client_id"), "good_client_id")
			test.Expect(t, vals.Get("f"), "json")
			test.Expect(t, vals.Get("grant_type"), "refresh_token")
			test.Expect(t, vals.Get("refresh_token"), "good_refresh_token")
			fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":1800}`)
		} else {
			t.Error(fmt.Sprintf("Unexpected ago request to route: %s", r.URL.Path))
		}
	}))
	defer agoServer.Close()

	device := &device{
		clientID: "good_client_id",
		env:      testEnv(gtServer.URL, agoServer.URL),
	}
	err := device.register()
	test.Expect(t, err, nil)

	client := &Client{device}

	params := map[string]interface{}{
		"tags": "derp",
	}
	var responseJSON map[string]interface{}

	err = client.Request("/some/route", params, &responseJSON)
	test.Expect(t, err, nil)
	test.Expect(t, responseJSON["triggers"].([]interface{})[0].(map[string]interface{})["triggerId"], "6fd01180fa1a012f27f1705681b27197")
	test.Expect(t, responseJSON["boundingBox"].(map[string]interface{})["xmax"], -122.45)
}

func TestDeviceConcurrentRefreshWaitingAtAccessStep(t *testing.T) {
	// This will spawn 4 go routines making requests with bad tokens.
	// The first routine will fire away immediately, get the invalid token response
	// from the geotrigger server, ask for permission to refresh, and start refreshing the token.
	// After a delay, the other 3 routines will ask to use the access token,
	// and end up waiting because a refresh is in progress.
	// After the first routine successfully refreshes the token, the waiting
	// routines will be give the message to continue by using the new access token.
	bt, gt := testConcurrentRefresh(t, getValidDeviceClient(t), "refresh_token", "", "good_refresh_token", true, false)
	test.Expect(t, bt, 1)
	test.Expect(t, gt, 4)
}

func TestDeviceConcurrentRefreshWaitingAtRefreshStep(t *testing.T) {
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
	bt, gt := testConcurrentRefresh(t, getValidDeviceClient(t), "refresh_token", "", "good_refresh_token", false, false)
	test.Expect(t, bt, 4)
	test.Expect(t, gt, 4)
}

func TestDeviceRecoveryFromErrorDuringRefreshWithRoutinesWaitingForAccess(t *testing.T) {
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
	bt, gt := testConcurrentRefresh(t, getValidDeviceClient(t), "refresh_token", "", "good_refresh_token", true, true)
	test.Expect(t, bt, 1)
	test.Expect(t, gt, 3)
}

func TestDeviceRecoveryFromErrorDuringRefreshWithRoutinesWaitingForRefresh(t *testing.T) {
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
	bt, gt := testConcurrentRefresh(t, getValidDeviceClient(t), "refresh_token", "", "good_refresh_token", false, true)
	test.Expect(t, bt, 4)
	test.Expect(t, gt, 3)
}

func TestDeviceConcurrentTokenExpirationWaitingAtAccessStep(t *testing.T) {
	dc := getValidDeviceClient(t)
	dc.setExpiresAt(-100)

	bt, gt := testConcurrentRefresh(t, dc, "refresh_token", "", "good_refresh_token", true, false)
	test.Expect(t, bt, 0)
	test.Expect(t, gt, 4)
}

func TestDeviceConcurrentTokenExpirationWaitingAtRefreshStep(t *testing.T) {
	dc := getValidDeviceClient(t)
	dc.setExpiresAt(-100)

	bt, gt := testConcurrentRefresh(t, dc, "refresh_token", "", "good_refresh_token", false, false)
	test.Expect(t, bt, 0)
	test.Expect(t, gt, 4)
}

func TestDeviceRecoveryFromErrorDuringTokenExpirationWaitingForAccess(t *testing.T) {
	dc := getValidDeviceClient(t)
	dc.setExpiresAt(-100)

	bt, gt := testConcurrentRefresh(t, dc, "refresh_token", "", "good_refresh_token", true, true)
	test.Expect(t, bt, 0)
	test.Expect(t, gt, 3)
}

func TestDeviceRecoveryFromErrorDuringTokenExpirationWaitingForRefresh(t *testing.T) {
	dc := getValidDeviceClient(t)
	dc.setExpiresAt(-100)

	bt, gt := testConcurrentRefresh(t, dc, "refresh_token", "", "good_refresh_token", false, true)
	test.Expect(t, bt, 0)
	test.Expect(t, gt, 3)
}

func getValidDeviceClient(t *testing.T) *Client {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		test.Refute(t, r, nil)
		test.Expect(t, r.URL.Path, "/sharing/oauth2/registerDevice")
		test.Expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		test.Expect(t, len(vals), 2)
		test.Expect(t, vals.Get("client_id"), "good_client_id")
		test.Expect(t, vals.Get("f"), "json")
		fmt.Fprintln(res, `{"device":{"deviceID":"device_id","client_id":"good_client_id","apnsProdToken":null,"apnsSandboxToken":null,"gcmRegistrationId":null,"registered":1389531528000,"lastAccessed":1389531528000},"deviceToken":{"access_token":"good_access_token","expires_in":1799,"refresh_token":"good_refresh_token"}}`)
	}))
	defer agoServer.Close()

	device := &device{
		clientID: "good_client_id",
		env:      testEnv("", agoServer.URL),
	}
	err := device.register()
	test.Expect(t, err, nil)

	return &Client{device}
}
