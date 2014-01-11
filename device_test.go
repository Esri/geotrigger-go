package geotrigger_golang

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A test that does some setup and then calls all the http related tests
func TestHttpStuff(t *testing.T) {
	// a pointer to these bytes is given to each sub-test so that they can define the expected response
	// and the server will serve that up
	var response []byte
	// a test server is set up
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		res.Write(response)
	}))
	defer ts.Close()
	// set the geotrigger base url to the url of the test server
	gtUrlRestorer, err := patch(&geotrigger_base_url, ts.URL)
	if err != nil {
		fmt.Printf("Error during test setup: %s", err)
		return
	}
	// after this test (and all sub-tests) complete, set the base url back to original value
	defer gtUrlRestorer.restore()
	// do the same for the AGO base url
	agoUrlRestorer, err := patch(&ago_base_url, ts.URL)
	if err != nil {
		fmt.Printf("Error during test setup: %s", err)
		return
	}
	defer agoUrlRestorer.restore()

	// now run all the tests!
	testDeviceRegisterFail(t, &response)
}

func testDeviceRegisterFail(t *testing.T, response *[]byte) {
	*response = []byte(`{"error":{"code":400,"message":"Unable to register device.","details":["'client_id' invalid"]}}`)
	expectedErrorMessage := "Error from /sharing/oauth2/registerDevice, code: 400. Message: Unable to register device."
	_, errChan := NewDeviceClient("bad_client_id")

	error := <-errChan

	refute(t, error, nil)
	expect(t, error.Error(), expectedErrorMessage)
}

func testDeviceRegisterSuccess(t *testing.T, responseByte *[]byte) {
	//	geotriggerErrorResponse := []byte(
	//	`{"error":{"type":"invalidHeader","message":"invalid header or header value","headers":{"Authorization":
	//	[{"type":"invalid","message":"Invalid token."}]},"code":498}}`)
}

func testSessionPostSuccess(t *testing.T) {

}

func testDeviceRegisterResponse(t *testing.T) {

}

func testDeviceRefreshResponse(t *testing.T) {

}
