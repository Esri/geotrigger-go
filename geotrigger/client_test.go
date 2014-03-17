package geotrigger

import (
	"encoding/json"
	"fmt"
	"github.com/Esri/geotrigger-go/geotrigger/test"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
