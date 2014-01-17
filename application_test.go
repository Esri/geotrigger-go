package geotrigger_golang

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func TestApplicationAccessRequestFail(t *testing.T) {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		refute(t, r, nil)
		expect(t, r.URL.Path, "/sharing/oauth2/token")
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		expect(t, len(vals), 4)
		expect(t, vals.Get("client_id"), "bad_client_id")
		expect(t, vals.Get("f"), "json")
		expect(t, vals.Get("grant_type"), "client_credentials")
		expect(t, vals.Get("client_secret"), "bad_client_secret")
		fmt.Fprintln(res, `{"error":{"code":999,"error":"invalid_request","error_description":"Invalid client_id","message":"invalid_request","details":[]}}`)
	}))
	defer agoServer.Close()

	// set the ago url to the url of our test server so we aren't hitting prod
	agoUrlRestorer, err := patch(&ago_base_url, agoServer.URL)
	if err != nil {
		t.Error("Error during test setup: %s", err)
	}
	defer agoUrlRestorer.restore()

	expectedErrorMessage := "Error from /sharing/oauth2/token, code: 999. Message: invalid_request"
	_, errChan := NewApplicationClient("bad_client_id", "bad_client_secret")

	error := <-errChan

	refute(t, error, nil)
	expect(t, error.Error(), expectedErrorMessage)
}

func TestApplicationRegisterSuccess(t *testing.T) {
	client := getValidApplicationClient(t)
	sessionInfo := client.GetSessionInfo()
	expect(t, sessionInfo["access_token"], "good_access_token")
	expect(t, sessionInfo["client_id"], "good_client_id")
	expect(t, sessionInfo["client_secret"], "good_client_secret")
}

func TestApplicationTokenRefresh(t *testing.T) {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		refute(t, r, nil)
		expect(t, r.URL.Path, "/sharing/oauth2/token")
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		expect(t, len(vals), 4)
		expect(t, vals.Get("client_id"), "good_client_id")
		expect(t, vals.Get("f"), "json")
		expect(t, vals.Get("grant_type"), "client_credentials")
		expect(t, vals.Get("client_secret"), "good_client_secret")
		fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":7200}`)
	}))
	defer agoServer.Close()

	// set the ago url to the url of our test server so we aren't hitting prod
	agoUrlRestorer, err := patch(&ago_base_url, agoServer.URL)
	if err != nil {
		t.Error("Error during test setup: %s", err)
	}
	defer agoUrlRestorer.restore()

	testApplication := &application{
		tokenManager: newTokenManager("old_access_token", ""),
		clientId:     "good_client_id",
		clientSecret: "good_client_secret",
		expiresIn:    4,
	}

	err = testApplication.refresh("")
	expect(t, err, nil)
	expect(t, testApplication.expiresIn, 7200)
	expect(t, testApplication.getAccessToken(), "refreshed_access_token")
	expect(t, testApplication.clientSecret, "good_client_secret")
	expect(t, testApplication.getRefreshToken(), "")
}

func TestApplicationFullWorkflowWithRefresh(t *testing.T) {
	// a test server to represent the geotrigger server
	gtServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		refute(t, r, nil)
		expect(t, r.URL.Path, "/some/route")
		expect(t, r.Header.Get("Content-Type"), "application/json")
		expect(t, r.Header.Get("X-GT-Client-Name"), "geotrigger_golang")
		expect(t, r.Header.Get("X-GT-Client-Version"), version)
		accessToken := r.Header.Get("Authorization")
		expect(t, strings.Index(accessToken, "Bearer "), 0)
		accessToken = strings.Split(accessToken, " ")[1]
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
	var tokenReqCount int
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		tokenReqCount++
		refute(t, r, nil)
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		expect(t, len(vals), 4)
		expect(t, vals.Get("client_id"), "good_client_id")
		expect(t, vals.Get("f"), "json")
		expect(t, vals.Get("grant_type"), "client_credentials")
		expect(t, vals.Get("client_secret"), "good_client_secret")

		if tokenReqCount == 1 {
			fmt.Fprintln(res, `{"access_token":"old_access_token","expires_in":1800}`)
		} else if tokenReqCount == 2 {
			fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":1800}`)
		} else {
			t.Error("Too many requests for application token (should only have been 2).")
		}
	}))
	defer agoServer.Close()

	// set the ago url to the url of our test server so we aren't hitting prod
	agoUrlRestorer, err := patch(&ago_base_url, agoServer.URL)
	if err != nil {
		t.Error("Error during test setup: %s", err)
	}
	defer agoUrlRestorer.restore()

	client, errChan := NewApplicationClient("good_client_id", "good_client_secret")

	error := <-errChan
	expect(t, error, nil)

	params := map[string]interface{}{
		"tags": "derp",
	}
	var responseJSON map[string]interface{}
	errChan = client.Request("/some/route", params, &responseJSON)

	err = <-errChan
	expect(t, err, nil)
	expect(t, responseJSON["triggers"].([]interface{})[0].(map[string]interface{})["triggerId"], "6fd01180fa1a012f27f1705681b27197")
	expect(t, responseJSON["boundingBox"].(map[string]interface{})["xmax"], -122.45)
}

func TestApplicationConcurrentRefreshWaitingAtAccessStep(t *testing.T) {
	badTokenAttempts, goodTokenAttempts := testApplicationConcurrentRefreshWaitingAtAccessStep(t,
		getValidApplicationClient(t))
	expect(t, badTokenAttempts, 1)
	expect(t, goodTokenAttempts, 4)
}

func TestApplicationConcurrentRefreshWaitingAtRefreshStep(t *testing.T) {
	badTokenAttempts, goodTokenAttempts := testApplicationConcurrentRefreshWaitingAtRefreshStep(t,
		getValidApplicationClient(t))
	expect(t, badTokenAttempts, 4)
	expect(t, goodTokenAttempts, 4)
}

func TestApplicationRepeatedConcurrentBatchesOfRequestsWithRefresh(t *testing.T) {
	client := getValidApplicationClient(t)
	badTokenAttempts, goodTokenAttempts := testApplicationConcurrentRefreshWaitingAtRefreshStep(t, client)
	expect(t, badTokenAttempts, 4)
	expect(t, goodTokenAttempts, 4)
	badTokenAttempts, goodTokenAttempts = testApplicationConcurrentRefreshWaitingAtAccessStep(t, client)
	expect(t, badTokenAttempts, 0)
	expect(t, goodTokenAttempts, 4)

	newClient := getValidApplicationClient(t)
	badTokenAttempts, goodTokenAttempts = testApplicationConcurrentRefreshWaitingAtAccessStep(t, newClient)
	expect(t, badTokenAttempts, 1)
	expect(t, goodTokenAttempts, 4)
	badTokenAttempts, goodTokenAttempts = testApplicationConcurrentRefreshWaitingAtRefreshStep(t, newClient)
	expect(t, badTokenAttempts, 0)
	expect(t, goodTokenAttempts, 4)

	anotherClient := getValidApplicationClient(t)
	var totalBadTokenAttempts, totalGoodTokenAttempts int
	var w sync.WaitGroup

	// don't go beyond 4 here, as the test server will start closing connections
	// if the request rate gets too high, resulting in test failures.  16 total is
	// enough to prove the test and isn't overwhelming the server when running on my MBA.
	w.Add(4)
	go func() {
		bt, gt := testApplicationConcurrentRefreshWaitingAtAccessStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	go func() {
		bt, gt := testApplicationConcurrentRefreshWaitingAtRefreshStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	go func() {
		bt, gt := testApplicationConcurrentRefreshWaitingAtRefreshStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	go func() {
		bt, gt := testApplicationConcurrentRefreshWaitingAtAccessStep(t, anotherClient)
		totalBadTokenAttempts += bt
		totalGoodTokenAttempts += gt
		w.Done()
	}()
	w.Wait()

	// 10 here because, starting with routine at the top, we have: 1 + 4 + 4 + 1
	// everything runs at once, which means we can add up the expectations as if
	// we were running each of these 4 routines separately with separate clients
	// as the timing is not a factor as it is in the first batch of sequential runs above.
	expect(t, totalBadTokenAttempts, 10)
	// 16 here because 4 * 4 (each routine uses a good token 4 times)
	expect(t, totalGoodTokenAttempts, 16)
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
	expect(t, bt, 2)
	expect(t, gt, 3)
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
	expect(t, bt, 4)
	expect(t, gt, 3)
}

func testApplicationConcurrentRefreshWaitingAtAccessStep(t *testing.T, client *Client) (int, int) {
	// This will spawn 4 go routines making requests with bad tokens.
	// The first routine will fire away immediately, get the invalid token response
	// from the geotrigger server, ask for permission to refresh, and start refreshing the token.
	// After a delay, the other 3 routines will ask to use the access token,
	// and end up waiting because a refresh is in progress.
	// After the first routine successfully refreshes the token, the waiting
	// routines will be give the message to continue by using the new access token.
	return testConcurrentRefresh(t, client, "client_credentials", "good_client_secret", "", true, false)
}

func testApplicationConcurrentRefreshWaitingAtRefreshStep(t *testing.T, client *Client) (int, int) {
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
	return testConcurrentRefresh(t, client, "client_credentials", "good_client_secret", "", false, false)
}

func getValidApplicationClient(t *testing.T) *Client {
	// a test server to represent AGO
	agoServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		refute(t, r, nil)
		expect(t, r.URL.Path, "/sharing/oauth2/token")
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		expect(t, len(vals), 4)
		expect(t, vals.Get("client_id"), "good_client_id")
		expect(t, vals.Get("f"), "json")
		expect(t, vals.Get("grant_type"), "client_credentials")
		expect(t, vals.Get("client_secret"), "good_client_secret")
		fmt.Fprintln(res, `{"access_token":"good_access_token","expires_in":7200}`)
	}))
	defer agoServer.Close()

	// set the ago url to the url of our test server so we aren't hitting prod
	agoUrlRestorer, err := patch(&ago_base_url, agoServer.URL)
	if err != nil {
		t.Error("Error during test setup: %s", err)
	}
	defer agoUrlRestorer.restore()

	client, errChan := NewApplicationClient("good_client_id", "good_client_secret")

	error := <-errChan

	expect(t, error, nil)
	return client
}
