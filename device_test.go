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
	"time"
	"sync"
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
		t.Error("Error during test setup: %s", err)
	}
	defer agoUrlRestorer.restore()

	expectedErrorMessage := "Error from /sharing/oauth2/registerDevice, code: 999. Message: Unable to register device."
	_, errChan := NewDeviceClient("bad_client_id")

	error := <-errChan

	refute(t, error, nil)
	expect(t, error.Error(), expectedErrorMessage)
}

func TestRegisterSuccess(t *testing.T) {
	client := getValidDeviceClient(t)
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
		t.Error("Error during test setup: %s", err)
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

func TestFullWorkflowWithRefresh(t *testing.T) {
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
			t.Error(fmt.Sprintf("Unexpected access token: %s", accessToken))
		}
	}))
	defer gtServer.Close()

	// set the geotrigger url to the url of our test server so we aren't hitting prod
	gtUrlRestorer, err := patch(&geotrigger_base_url, gtServer.URL)
	if err != nil {
		t.Error("Error during test setup: %s", err)
	}
	defer gtUrlRestorer.restore()

	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		refute(t, r, nil)
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))

		if r.URL.Path == ago_register_route {
			expect(t, len(vals), 2)
			expect(t, vals.Get("client_id"), "good_client_id")
			expect(t, vals.Get("f"), "json")
			fmt.Fprintln(res, `{"device":{"deviceId":"device_id","client_id":"good_client_id","apnsProdToken":null,"apnsSandboxToken":null,"gcmRegistrationId":null,"registered":1389531528000,"lastAccessed":1389531528000},"deviceToken":{"access_token":"old_access_token","expires_in":1799,"refresh_token":"good_refresh_token"}}`)
		} else if r.URL.Path == ago_token_route {
			expect(t, len(vals), 4)
			expect(t, vals.Get("client_id"), "good_client_id")
			expect(t, vals.Get("f"), "json")
			expect(t, vals.Get("grant_type"), "refresh_token")
			expect(t, vals.Get("refresh_token"), "good_refresh_token")
			fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":1800}`)
		} else {
			t.Error(fmt.Sprintf("Unexpected ago request to route: %s", r.URL.Path))
		}
	}))
	defer agoServer.Close()

	// set the ago url to the url of our test server so we aren't hitting prod
	agoUrlRestorer, err := patch(&ago_base_url, agoServer.URL)
	if err != nil {
		t.Error("Error during test setup: %s", err)
	}
	defer agoUrlRestorer.restore()

	client, errChan := NewDeviceClient("good_client_id")

	error := <- errChan
	expect(t, error, nil)

	params := map[string]interface{} {
		"tags": "derp",
	}
	var responseJSON map[string]interface{}
	errChan = client.Request("/some/route", params, &responseJSON)

	err = <- errChan
	expect(t, err, nil)
	expect(t, responseJSON["triggers"].([]interface{})[0].(map[string]interface{})["triggerId"], "6fd01180fa1a012f27f1705681b27197")
	expect(t, responseJSON["boundingBox"].(map[string]interface{})["xmax"], -122.45)

	// test GetValueFromJSONObject and GetValueFromJSONArray a bit
	var triggers []interface{}
	err = GetValueFromJSONObject(responseJSON, "triggers", &triggers)
	if err != nil {
		t.Error("Error while unpacking JSON:", err)
	}

	var trigger map[string]interface{}
	err = GetValueFromJSONArray(triggers, 0, &trigger)
	if err != nil {
		t.Error("Error while unpacking JSON:", err)
	}

	var action map[string]interface{}
	err = GetValueFromJSONObject(trigger, "action", &action)
	if err != nil {
		t.Error("Error while unpacking JSON:", err)
	}

	var callback string
	err = GetValueFromJSONObject(action, "callback", &callback)
	if err != nil {
		t.Error("Error while unpacking JSON:", err)
	}

	expect(t, callback, "http://pdx.gov/welcome")
}

func TestConcurrentRefreshWaitingAtAccessStep(t *testing.T) {
	badTokenAttempts, goodTokenAttempts := testConcurrentRefreshWaitingAtAccessStep(t, nil)
	expect(t, badTokenAttempts, 1)
	expect(t, goodTokenAttempts, 4)
}

func TestConcurrentRefreshWaitingAtRefreshStep(t *testing.T) {
	badTokenAttempts, goodTokenAttempts := testConcurrentRefreshWaitingAtRefreshStep(t, nil)
	expect(t, badTokenAttempts, 4)
	expect(t, goodTokenAttempts, 4)
}

func TestRepeatedConcurrentBatchesOfRequestsWithRefresh(t *testing.T) {
	client := getValidDeviceClient(t)
	badTokenAttempts, goodTokenAttempts := testConcurrentRefreshWaitingAtRefreshStep(t, client)
	expect(t, badTokenAttempts, 4)
	expect(t, goodTokenAttempts, 4)
	badTokenAttempts, goodTokenAttempts = testConcurrentRefreshWaitingAtAccessStep(t, client)
	expect(t, badTokenAttempts, 0)
	expect(t, goodTokenAttempts, 4)

	newClient := getValidDeviceClient(t)
	badTokenAttempts, goodTokenAttempts = testConcurrentRefreshWaitingAtAccessStep(t, newClient)
	expect(t, badTokenAttempts, 1)
	expect(t, goodTokenAttempts, 4)
	badTokenAttempts, goodTokenAttempts = testConcurrentRefreshWaitingAtRefreshStep(t, newClient)
	expect(t, badTokenAttempts, 0)
	expect(t, goodTokenAttempts, 4)

	anotherClient := getValidDeviceClient(t)
	var totalBadTokenAttempts, totalGoodTokenAttempts int
	var w sync.WaitGroup
	w.Add(6)
	go func() {
		bt, gt := testConcurrentRefreshWaitingAtAccessStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	go func() {
		bt, gt := testConcurrentRefreshWaitingAtAccessStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	go func() {
		bt, gt := testConcurrentRefreshWaitingAtRefreshStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	go func() {
		bt, gt := testConcurrentRefreshWaitingAtRefreshStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	go func() {
		bt, gt := testConcurrentRefreshWaitingAtAccessStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	go func() {
		bt, gt := testConcurrentRefreshWaitingAtRefreshStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	w.Wait()

	// 15 here because, starting with routine at the top, we have: 1 + 1 + 4 + 4 + 1 + 4
	// everything runs at once, which means we can add up the expectations as if
	// we were running each of these 4 routines separately with separate clients
	// as the timing is not a factor as it is in the first batch of sequential runs above.
	expect(t, totalBadTokenAttempts, 15)
	// 24 here because 6 * 4 (each routine uses a good token 4 times)
	expect(t, totalGoodTokenAttempts, 24)
}

func testConcurrentRefreshWaitingAtAccessStep(t *testing.T, client *Client) (int, int) {
	// This will spawn 4 go routines making requests with bad tokens.
	// The first routine will fire away immediately, get the invalid token response
	// from the geotrigger server, ask for permission to refresh, and start refreshing the token.
	// After a 1 second delay, the other 3 routines will ask to use the access token,
	// and end up waiting because a refresh is in progress.
	// After the first routine successfully refreshes the token, the waiting
	// routines will be give the message to continue by using the new access token.
	return testConcurrentRefresh(t, client, true)
}

func testConcurrentRefreshWaitingAtRefreshStep(t *testing.T, client *Client) (int, int) {
	// This will spawn 4 go routines making requests with bad tokens.
	// Each routine will get permissions to present the access token to
	// the geotrigger server.
	// The first routine to receive the invalid token response will then ask for
	// permission to refresh the token, and be granted that permission.
	// The other 3 routines will also ask for permission to refresh, and instead
	// end up waiting for a reply to that request.
	// After the first routine successfully refreshes the token, the waiting
	// routines will be give the message to continue, but not refresh, and
	// instead use the new access token.
	return testConcurrentRefresh(t, client, false)
}

// A big ugly func, separated out to avoid duplicating it
func testConcurrentRefresh(t *testing.T, client *Client, pauseAfterFirstReq bool) (int, int){
	if client == nil {
		client = getValidDeviceClient(t)
	}

	var refreshCount int
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		refreshCount++

		if refreshCount > 1 {
			t.Error("Too many refresh attempts! Should have only been 1.")
		}

		time.Sleep(3 * time.Second)
		expect(t, r.URL.Path, ago_token_route)
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
		t.Error("Error during test setup: %s", err)
	}
	defer agoUrlRestorer.restore()

	var oldAccessTokenUse, refreshedAccessTokenUse int
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

	// set the geotrigger url to the url of our test server so we aren't hitting prod
	gtUrlRestorer, err := patch(&geotrigger_base_url, gtServer.URL)
	if err != nil {
		t.Error("Error during test setup: %s", err)
	}
	defer gtUrlRestorer.restore()

	params1 := map[string]interface{} {
		"tags": "derp",
	}
	var responseJSON1 map[string]interface{}
	params2 := map[string]interface{} {
		"tags": "derp",
	}
	var responseJSON2 map[string]interface{}
	params3 := map[string]interface{} {
		"tags": "derp",
	}
	var responseJSON3 map[string]interface{}
	params4 := map[string]interface{} {
		"tags": "derp",
	}
	var responseJSON4 map[string]interface{}

	errChan1 := client.Request("/some/route", params1, &responseJSON1)
	if pauseAfterFirstReq {
		time.Sleep(1 * time.Second)
	}
	errChan2 := client.Request("/some/route", params2, &responseJSON2)
	errChan3 := client.Request("/some/route", params3, &responseJSON3)
	errChan4 := client.Request("/some/route", params4, &responseJSON4)

	var w sync.WaitGroup
	w.Add(4)
	go func() {
		error := <- errChan1
		expect(t, error, nil)
		w.Done()
	}()
	go func() {
		error := <- errChan2
		expect(t, error, nil)
		w.Done()
	}()
	go func() {
		error := <- errChan3
		expect(t, error, nil)
		w.Done()
	}()
	go func() {
		error := <- errChan4
		expect(t, error, nil)
		w.Done()
	}()
	w.Wait()

	return oldAccessTokenUse, refreshedAccessTokenUse
}

// separated out to avoid some duplication where possible
func getValidDeviceClient(t *testing.T) *Client {
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
		t.Error("Error during test setup: %s", err)
	}
	defer agoUrlRestorer.restore()

	client, errChan := NewDeviceClient("good_client_id")

	error := <- errChan

	expect(t, error, nil)
	return client
}
