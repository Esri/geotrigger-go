package geotrigger_golang

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"io/ioutil"
	"net/url"
	"encoding/json"
	"strings"
)

func TestRegisterFail(t *testing.T) {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		expect(t, r.URL.Path, "/sharing/oauth2/registerDevice")
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		refute(t, r, nil)
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		expect(t, len(vals), 2)
		expect(t, vals.Get("client_id"), "bad_client_id")
		expect(t, vals.Get("f"), "json")
		fmt.Fprintln(res, `{"error":{"code":999,"message":"Unable to register device.","details":["'client_id' invalid"]}}`)
	}))
	defer agoServer.Close()

	// set the ago url to the url of our test server so we aren't hitting prod
	agoUrlRestorer, err := patch(&ago_base_url, agoServer.URL)
	if err != nil {
		fmt.Printf("Error during test setup: %s", err)
		return
	}
	defer agoUrlRestorer.restore()

	expectedErrorMessage := "Error from /sharing/oauth2/registerDevice, code: 999. Message: Unable to register device."
	_, errChan := NewDeviceClient("bad_client_id")

	error := <-errChan

	refute(t, error, nil)
	expect(t, error.Error(), expectedErrorMessage)
}

func TestRegisterSuccess(t *testing.T) {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		expect(t, r.URL.Path, "/sharing/oauth2/registerDevice")
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		refute(t, r, nil)
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		expect(t, len(vals), 2)
		expect(t, vals.Get("client_id"), "good_client_id")
		expect(t, vals.Get("f"), "json")
		fmt.Fprintln(res, `{"device":{"deviceId":"device_id","client_id":"good_client_id","apnsProdToken":null,"apnsSandboxToken":null,"gcmRegistrationId":null,"registered":1389531528000,"lastAccessed":1389531528000},"deviceToken":{"access_token":"good_access_token","expires_in":1799,"refresh_token":"good_refresh_token"}}`)
	}))
	defer agoServer.Close()

	// set the ago url to the url of our test server so we aren't hitting prod
	agoUrlRestorer, err := patch(&ago_base_url, agoServer.URL)
	if err != nil {
		fmt.Printf("Error during test setup: %s", err)
		return
	}
	defer agoUrlRestorer.restore()

	client, errChan := NewDeviceClient("good_client_id")

	error := <- errChan

	expect(t, error, nil)
	sessionInfo := client.GetSessionInfo()
	expect(t, sessionInfo["access_token"], "good_access_token")
	expect(t, sessionInfo["refresh_token"], "good_refresh_token")
	expect(t, sessionInfo["device_id"], "device_id")
	expect(t, sessionInfo["client_id"], "good_client_id")
}

func TestTokenRefresh(t *testing.T) {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		expect(t, r.URL.Path, "/sharing/oauth2/token")
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		refute(t, r, nil)
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		expect(t, len(vals), 4)
		expect(t, vals.Get("client_id"), "good_client_id")
		expect(t, vals.Get("f"), "json")
		expect(t, vals.Get("grant_type"), "refresh_token")
		expect(t, vals.Get("refresh_token"), "good_refresh_token")
		fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":1800}`)
	}))
	defer agoServer.Close()

	// set the ago url to the url of our test server so we aren't hitting prod
	agoUrlRestorer, err := patch(&ago_base_url, agoServer.URL)
	if err != nil {
		fmt.Printf("Error during test setup: %s", err)
		return
	}
	defer agoUrlRestorer.restore()

	testDevice := &device{
		clientId: "good_client_id",
		deviceId: "device_id",
		accessToken: "old_access_token",
		refreshToken: "good_refresh_token",
		expiresIn : 4,
		refreshStatusChecks: make(chan *refreshStatusCheck),
	}

	err = testDevice.refresh()
	expect(t, err, nil)
	expect(t, testDevice.expiresIn, 1800)
	expect(t, testDevice.accessToken, "refreshed_access_token")
	expect(t, testDevice.clientId, "good_client_id")
	expect(t, testDevice.refreshToken, "good_refresh_token")
}

func testExpiredTokenRefresh(t *testing.T) {
	// a test server to represent the geotrigger server
	gtServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		expect(t, r.URL.Path, "/some/route")
		expect(t, r.Header.Get("Content-Type"), "application/json")
		accessToken := r.Header.Get("Authorization")
		expect(t, strings.Index(accessToken, "Bearer "), 0)
		accessToken = strings.Split(accessToken, " ")[1]
		refute(t, r, nil)
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		var params map[string]interface{}
		_ = json.Unmarshal(contents, &params)
		expect(t, len(params), 1)
		expect(t, params["tags"], "derp")

		if accessToken == "old_access_token" {
			fmt.Fprintln(res, `{"error":{"type":"invalidHeader","message":"invalid header or header value","headers":{"Authorization":[{"type":"invalid","message":"Invalid token."}]},"code":498}}`)
		} else if accessToken == "refreshed_access_token" {
			fmt.Fprintln(res, `{"triggers":[{"triggerId":"6fd01180fa1a012f27f1705681b27197","condition":{"direction":"enter","geo":{"geocode":"920 SW 3rd Ave, Portland, OR","driveTime":600,"context":{"locality":"Portland","region":"Oregon","country":"USA","zipcode":"97204"}}},"action":{"message":"Welcome to Portland - The Mayor","callback":"http://pdx.gov/welcome"},"tags":["foodcarts","citygreetings"]}],"boundingBox":{"xmin":-122.68,"ymin":45.53,"xmax":-122.45,"ymax":45.6}}`)
		} else {
			t.Error(fmt.Sprintf("Unexpected access token: ", accessToken))
		}
	}))
	defer gtServer.Close()

	// set the geotrigger url to the url of our test server so we aren't hitting prod
	gtUrlRestorer, err := patch(&geotrigger_base_url, gtServer.URL)
	if err != nil {
		fmt.Printf("Error during test setup: %s", err)
		return
	}
	defer gtUrlRestorer.restore()

	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		refute(t, r, nil)
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		expect(t, len(vals), 4)
		expect(t, vals.Get("client_id"), "good_client_id")
		expect(t, vals.Get("f"), "json")
		expect(t, vals.Get("grant_type"), "refresh_token")
		expect(t, vals.Get("refresh_token"), "good_refresh_token")
		fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":1800}`)
	}))
	defer agoServer.Close()

	// set the ago url to the url of our test server so we aren't hitting prod
	agoUrlRestorer, err := patch(&ago_base_url, agoServer.URL)
	if err != nil {
		fmt.Printf("Error during test setup: %s", err)
		return
	}
	defer agoUrlRestorer.restore()


	testDevice := &device{
		clientId: "good_client_id",
		deviceId: "device_id",
		accessToken: "old_access_token",
		refreshToken: "good_refresh_token",
		expiresIn : 4,
		refreshStatusChecks: make(chan *refreshStatusCheck),
	}

	go testDevice.tokenManager()

	params := map[string]interface{} {
		"tags": "derp",
	}
	var responseJSON map[string]interface{}
	errorChan := make(chan error)
	go func() {
		testDevice.geotriggerAPIRequest("/some/route", params, &responseJSON, errorChan)
	}()

	err = <- errorChan
	expect(t, err, nil)
}

func testDeviceRefreshResponse(t *testing.T) {

}
