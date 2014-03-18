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
)

func TestClientWithNewApplication(t *testing.T) {
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

	// set the ago url to the url of our test server so we aren't hitting prod
	agoURLRestorer, err := test.Patch(defEnv, *testEnv("", agoServer.URL))
	if err != nil {
		t.Error("Error during test setup: %s", err)
	}
	defer agoURLRestorer.Restore()

	client, err := NewApplication("good_client_id", "good_client_secret")
	test.Expect(t, err, nil)
	test.Refute(t, client, nil)
}

func TestClientWithNewDevice(t *testing.T) {
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

	// set the ago url to the url of our test server so we aren't hitting prod
	agoURLRestorer, err := test.Patch(defEnv, *testEnv("", agoServer.URL))
	if err != nil {
		t.Error("Error during test setup: %s", err)
	}
	defer agoURLRestorer.Restore()

	client, err := NewDevice("good_client_id")
	test.Expect(t, err, nil)
	test.Refute(t, client, nil)
}

func TestClientWithExistingDevice(t *testing.T) {
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
		test.Expect(t, accessToken, "good_access_token")
		contents, _ := ioutil.ReadAll(r.Body)
		test.Refute(t, len(contents), 0)
		var params map[string]interface{}
		_ = json.Unmarshal(contents, &params)
		test.Expect(t, len(params), 1)
		test.Expect(t, params["tags"], "derp")
		fmt.Fprintln(res, `{}`)
	}))
	defer gtServer.Close()

	client := ExistingDevice("good_client_id", "device_id", "good_access_token", 1800, "good_refresh_token")
	client.session.setEnv(testEnv(gtServer.URL, ""))

	params := map[string]interface{}{
		"tags": "derp",
	}
	var responseJSON map[string]interface{}

	err := client.Request("/some/route", params, &responseJSON)
	test.Expect(t, err, nil)
}
