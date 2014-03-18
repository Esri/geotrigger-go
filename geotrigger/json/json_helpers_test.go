package json

import (
	"encoding/json"
	"testing"
	"github.com/Esri/geotrigger-go/geotrigger/test"
)

/* editing these will break tests */
var triggerListData = []byte(`{"triggers":[{"triggerId":"6fd01180fa1a012f27f1705681b27197","condition":{"direction":"enter","geo":{"geocode":"920 SW 3rd Ave, Portland, OR","driveTime":600,"context":{"locality":"Portland","region":"Oregon","country":"USA","zipcode":"97204"}}},"action":{"message":"Welcome to Portland - The Mayor","callback":"http://pdx.gov/welcome"},"tags":["foodcarts","citygreetings"]}],"boundingBox":{"xmin":-122.68,"ymin":45.53,"xmax":-122.45,"ymax":45.6}}`)

func TestValueGetters(t *testing.T) {
	var responseJSON map[string]interface{}
	err := json.Unmarshal(triggerListData, &responseJSON)
	test.Expect(t, err, nil)

	// test GetValueFromJSONObject and GetValueFromJSONArray a bit
	var triggers []interface{}
	err = GetValueFromJSONObject(responseJSON, "triggers", &triggers)
	test.Expect(t, err, nil)

	var trigger map[string]interface{}
	err = GetValueFromJSONArray(triggers, 0, &trigger)
	test.Expect(t, err, nil)

	var tags []interface{}
	err = GetValueFromJSONObject(trigger, "tags", &tags)
	test.Expect(t, err, nil)
	test.Expect(t, tags[0], "foodcarts")

	var action map[string]interface{}
	err = GetValueFromJSONObject(trigger, "action", &action)
	test.Expect(t, err, nil)

	var callback string
	err = GetValueFromJSONObject(action, "callback", &callback)
	test.Expect(t, err, nil)

	test.Expect(t, callback, "http://pdx.gov/welcome")

	// works with empty interface
	var emptyInterface interface{}
	err = GetValueFromJSONObject(trigger, "triggerId", &emptyInterface)
	test.Expect(t, err, nil)
	test.Expect(t, emptyInterface, "6fd01180fa1a012f27f1705681b27197")

	// provide nil array/object
	var failureInterface interface{}
	err = GetValueFromJSONObject(nil, "derp", &failureInterface)
	test.Expect(t, failureInterface, nil)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Attempt to get value from a nil JSON object.")

	err = GetValueFromJSONArray(nil, 1, &failureInterface)
	test.Expect(t, failureInterface, nil)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Attempt to get value from a nil JSON aray.")

	// empty key
	err = GetValueFromJSONObject(responseJSON, "", &failureInterface)
	test.Expect(t, failureInterface, nil)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Attempt to pull value for empty key from JSON object.")

	// no matching value in map
	err = GetValueFromJSONObject(responseJSON, "horse with hands", &failureInterface)
	test.Expect(t, failureInterface, nil)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "No value found for key: horse with hands")

	// index out of range
	err = GetValueFromJSONArray(triggers, -3, &failureInterface)
	test.Expect(t, failureInterface, nil)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Provided index -3 was out of range.")

	err = GetValueFromJSONArray(triggers, 17, &failureInterface)
	test.Expect(t, failureInterface, nil)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Provided index 17 was out of range.")

	// not a pointer
	err = GetValueFromJSONArray(triggers, 0, failureInterface)
	test.Expect(t, failureInterface, nil)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Provided value is of invalid type (must be pointer).")

	var notAPointer2 Trigger
	err = GetValueFromJSONArray(triggers, 0, notAPointer2)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Provided value is of invalid type (must be pointer).")
	test.Expect(t, len(notAPointer2.Tags), 0)
	test.Expect(t, notAPointer2.Condition.Geo.DriveTime, 0)

	var notAPointer3 BoundingBox
	err = GetValueFromJSONObject(responseJSON, "boundingBox", notAPointer3)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Provided value is of invalid type (must be pointer).")
	test.Expect(t, notAPointer3.Ymin, float64(0))

	// wrong value type!
	var wrongType1 BoundingBox
	err = GetValueFromJSONObject(responseJSON, "triggers", &wrongType1)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Provided reference is to a value of type geotrigger.BoundingBox that cannot be assigned to type found in JSON: []interface {}.")
	test.Expect(t, notAPointer3.Xmin, float64(0))

	var wrongType2 []interface{}
	err = GetValueFromJSONArray(triggers, 0, &wrongType2)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Provided reference is to a value of type []interface {} that cannot be assigned to type found in JSON: map[string]interface {}.")
	test.Expect(t, notAPointer3.Xmin, float64(0))
}
