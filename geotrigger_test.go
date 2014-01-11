package geotrigger_golang

import (
	"testing"
	"reflect"
	"net/http/httptest"
	"net/http"
	"fmt"
	"errors"
)

/* Functions for mocking by replacing package vars at runtime */
// Restorer holds a function that can be used
// to restore some previous state.
type Restorer func()

// Restore restores some previous state.
func (r Restorer) Restore() {
	r()
}

// Patch sets the value pointed to by the given destination to the given
// value, and returns a function to restore it to its original value.  The
// value must be assignable to the element type of the destination.
func Patch(destination, v interface{}) (Restorer, error) {
	destType := reflect.TypeOf(destination)
	if reflect.TypeOf(destination).Kind() != reflect.Ptr {
		return nil, errors.New("Bad destination, please provide a pointer.")
	}

	// we know it's a pointer, so get the type of value being pointed to
	destType = destType.Elem()
	// compare that type to the type of v
	providedType := reflect.TypeOf(v)
	if destType != providedType {
		return nil, errors.New(fmt.Sprintf("Provided value of type %s does not match destination type: %s.",
			providedType, destType))
	}

	// we know destination is a pointer, let's get the value being pointed to
	destValue := reflect.ValueOf(destination).Elem()

	// reflect.New creates a new pointer value to provided type, elem gets the pointed to value again
	oldValue := reflect.New(destValue.Type()).Elem()
	// we then set that value to the current destination value to hold onto it
	oldValue.Set(destValue)

	// the value of the provided... value...
	value := reflect.ValueOf(v)
	if !value.IsValid() {
		// This should be a rare occurrence.
		// the value provided could not be reflected, and we have an invalid Value type here
		// so just attempt to use the zero value for the destination type.
		value = reflect.Zero(destValue.Type())
	}

	// replace the destination's current val with the provided v
	// this shouldn't panic, because we have already checked that they are the same type
	destValue.Set(value)
	return func() {
		// restore the destination's value to its original
		destValue.Set(oldValue)
	}, nil
}

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func TestHttpStuff(t *testing.T) {
	// test setup for all the tests that need to mock http
	var response []byte
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		res.Write(response)
	}))
	defer ts.Close()
	gtUrlRestore, err := Patch(&GEOTRIGGER_BASE_URL, ts.URL)
	if err != nil {
		fmt.Printf("Error test during setup: %s", err)
		return
	}
	defer gtUrlRestore.Restore()
	agoUrlRestore, err := Patch(&AGO_BASE_URL, ts.URL)
	if err != nil {
		fmt.Printf("Error test during setup: %s", err)
		return
	}
	defer agoUrlRestore.Restore()

	// now run all the tests!
	testDeviceRegisterFail(t, &response)
}

func testDeviceRegisterFail(t *testing.T, response *[]byte) {
	*response = []byte(`{"error":{"code":500,"message":"Unable to register device.","details":["'client_id' invalid"]}}`)
	expectedErrorMessage := "Error from AGO, code: 500. Message: Unable to register device."
	_, err := NewDeviceClient("bad_client_id")
	if err == nil {
		t.Error("Expected an error, but instead got a client!\n")
	} else if err.Error() != expectedErrorMessage {
		t.Error("Got an error (good!) but not the right error (bad!).\n")
	} else {
		fmt.Printf("SUCCESS, got expected error: %s\n", err)
	}
}

func testSessionGeotriggerError(t *testing.T, responseByte *[]byte) {
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
