package geotrigger_golang

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// mustBeNil()/mustNotBeNil() isNillable() author: courtf
func mustBeNil(t *testing.T, a interface{}) {
	tp := reflect.TypeOf(a)

	if tp != nil && (!isNillable(tp.Kind()) || !reflect.ValueOf(a).IsNil()) {
		t.Errorf("Expected %v (type %v) to be nil", a, tp)
	}
}

func mustNotBeNil(t *testing.T, a interface{}) {
	tp := reflect.TypeOf(a)

	if tp == nil || (isNillable(tp.Kind()) && reflect.ValueOf(a).IsNil()) {
		t.Errorf("Expected %v (type %v) to not be nil", a, tp)
	}
}

func isNillable(k reflect.Kind) (nillable bool) {
	kinds := []reflect.Kind{
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Ptr,
		reflect.Slice,
	}

	for i := 0; i < len(kinds); i++ {
		if kinds[i] == k {
			nillable = true
			break
		}
	}

	return
}

// restorer and patch adapted from https://gist.github.com/imosquera/6716490
// thanks imosquera!
// Restorer holds a function that can be used
// to restore some previous state.
type restorer func()

// Restore restores some previous state.
func (r restorer) restore() {
	r()
}

// Patch sets the value pointed to by the given destination to the given
// value, and returns a function to restore it to its original value.  The
// value must be assignable to the element type of the destination.
func patch(destination, v interface{}) (restorer, error) {
	destType := reflect.TypeOf(destination)
	if reflect.TypeOf(destination).Kind() != reflect.Ptr {
		return nil, errors.New("Bad destination, please provide a pointer.")
	}

	// we know destination is a pointer, so get the type of value being pointed to
	destType = destType.Elem()
	// compare that type to the type of v
	providedType := reflect.TypeOf(v)
	if destType != providedType {
		return nil, errors.New(fmt.Sprintf("Provided value of type %s does not match destination type: %s.",
			providedType, destType))
	}

	// get the value being pointed to
	destValue := reflect.ValueOf(destination).Elem()
	// reflect.New creates a new pointer value to provided type, elem gets the pointed to value again
	oldValue := reflect.New(destType).Elem()
	// we then set that value to the current destination value to hold onto it
	oldValue.Set(destValue)

	// the value of the provided... value...
	value := reflect.ValueOf(v)
	if !value.IsValid() {
		// This should be a rare occurrence.
		// the value provided could not be reflected, and we have an invalid Value here
		// so just attempt to use the zero value for the destination type.
		value = reflect.Zero(destValue.Type())
	}

	// replace the destination's current value with the value of the provided v
	// this shouldn't panic, because we have already checked that they are the same type
	destValue.Set(value)
	return func() {
		// restore the destination's value to its original
		destValue.Set(oldValue)
	}, nil
}

// https://github.com/codegangsta/martini/blob/master/martini_test.go
// thanks codegangsta for these lil guys ;)
func expect(t *testing.T, a interface{}, b interface{}) {
	if b == nil {
		mustBeNil(t, a)
	} else if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if b == nil {
		mustNotBeNil(t, a)
	} else if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
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

		refute(t, r, nil)
		time.Sleep(80 * time.Millisecond)
		expect(t, r.URL.Path, ago_token_route)
		expect(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		contents, _ := ioutil.ReadAll(r.Body)
		refute(t, len(contents), 0)
		vals, _ := url.ParseQuery(string(contents))
		expect(t, len(vals), 4)
		expect(t, vals.Get("client_id"), "good_client_id")
		expect(t, vals.Get("f"), "json")
		expect(t, vals.Get("grant_type"), grantType)
		expect(t, vals.Get("client_secret"), clientSecret)
		expect(t, vals.Get("refresh_token"), refreshToken)
		if refreshCount == 1 && errorOnFirstRefresh {
			fmt.Fprintln(res, `{"error":{"code":498,"message":"Invalid token."}}`)
		} else {
			fmt.Fprintln(res, `{"access_token":"refreshed_access_token","expires_in":1800}`)
		}
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

	errChan1 := client.Request("/some/route", params1, &responseJSON1)
	if pauseAfterFirstReq {
		time.Sleep(20 * time.Millisecond)
	}
	errChan2 := client.Request("/some/route", params2, &responseJSON2)
	errChan3 := client.Request("/some/route", params3, &responseJSON3)
	errChan4 := client.Request("/some/route", params4, &responseJSON4)

	var w sync.WaitGroup
	var errorCount int
	w.Add(4)
	go func() {
		error := <-errChan1
		if error != nil {
			errorCount++
		}
		w.Done()
	}()
	go func() {
		error := <-errChan2
		if error != nil {
			errorCount++
		}
		w.Done()
	}()
	go func() {
		error := <-errChan3
		if error != nil {
			errorCount++
		}
		w.Done()
	}()
	go func() {
		error := <-errChan4
		if error != nil {
			errorCount++
		}
		w.Done()
	}()
	w.Wait()

	if errorOnFirstRefresh {
		expect(t, errorCount, 1)
	} else {
		expect(t, errorCount, 0)
	}

	return oldAccessTokenUse, refreshedAccessTokenUse
}
