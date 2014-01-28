package geotrigger_golang

import (
	"encoding/json"
	"testing"
)

type TriggerList struct {
	Triggers    []Trigger   `json:"triggers"`
	BoundingBox BoundingBox `json:"boundingBox"`
}

type Trigger struct {
	TriggerID string    `json:"triggerId"`
	Condition Condition `json:"condition"`
	Action    Action    `json:"action"`
	Tags      []string  `json:"tags"`
}

type Condition struct {
	Direction string `json:"direction"`
	Geo       Geo    `json:"geo"`
}

type Geo struct {
	Geocode   string  `json:"geocode"`
	DriveTime int     `json:"drivetime"`
	Context   Context `json:"context"`
}

type Context struct {
	Locality string `json:"locality"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Zipcode  string `json:"zipcode"`
}

type Action struct {
	Message  string `json:"message"`
	Callback string `json:"callback"`
}

type BoundingBox struct {
	Xmin float64 `json:"xmin"`
	Ymin float64 `json:"ymin"`
	Xmax float64 `json:"xmax"`
	Ymax float64 `json:"ymax"`
}

type WrongJSON struct {
	Derp   []string `json"derp"`
	Dorp   int      `json:"dorp"`
	Action Action   `json:"action"`
}

/* editing these will break tests */
var triggerListData = []byte(`{"triggers":[{"triggerId":"6fd01180fa1a012f27f1705681b27197","condition":{"direction":"enter","geo":{"geocode":"920 SW 3rd Ave, Portland, OR","driveTime":600,"context":{"locality":"Portland","region":"Oregon","country":"USA","zipcode":"97204"}}},"action":{"message":"Welcome to Portland - The Mayor","callback":"http://pdx.gov/welcome"},"tags":["foodcarts","citygreetings"]}],"boundingBox":{"xmin":-122.68,"ymin":45.53,"xmax":-122.45,"ymax":45.6}}`)

// missing geo[drivetime], context[region], trigger[tags] and boundingBox[xmax]
var partialTriggerListData = []byte(`{"triggers":[{"triggerId":"6fd01180fa1a012f27f1705681b27197","condition":{"direction":"enter","geo":{"geocode":"920 SW 3rd Ave, Portland, OR","context":{"locality":"Portland","country":"USA","zipcode":"97204"}}},"action":{"message":"Welcome to Portland - The Mayor","callback":"http://pdx.gov/welcome"}}],"boundingBox":{"xmin":-122.68,"ymin":45.53,"ymax":45.6}}`)
var jsonArrayData = []byte(`["herp", "derp", "dorp"]`)

func TestErrorCheck(t *testing.T) {
	resp := []byte(`["array", "root", "element"]`)
	errResponse := errorCheck(resp)
	expect(t, errResponse, nil)

	resp = []byte(`{"object":"doesnt match", "derp":["dorp", "morp"]}`)
	errResponse = errorCheck(resp)
	expect(t, errResponse, nil)

	resp = []byte(`{"error":{"code":400,"message":"Invalid token."}}`)
	errResponse = errorCheck(resp)
	refute(t, errResponse, nil)
}

func TestParseJSONResponse(t *testing.T) {
	var responseJSON TriggerList
	err := parseJSONResponse(triggerListData, &responseJSON)
	expect(t, err, nil)

	expect(t, responseJSON.BoundingBox.Xmax, -122.45)
	expect(t, len(responseJSON.Triggers), 1)
	expect(t, responseJSON.Triggers[0].Condition.Geo.Context.Zipcode, "97204")

	var partialJSON TriggerList
	err = parseJSONResponse(partialTriggerListData, &partialJSON)
	expect(t, err, nil)

	// xmax is missing, will be set to zero value
	expect(t, partialJSON.BoundingBox.Xmax, float64(0))
	expect(t, len(partialJSON.Triggers), 1)
	expect(t, partialJSON.Triggers[0].Condition.Geo.Context.Zipcode, "97204")
	// tags are missing, should be nil
	expect(t, partialJSON.Triggers[0].Tags, nil)
	expect(t, partialJSON.Triggers[0].Condition.Geo.Context.Region, "")

	var wrongJSON WrongJSON
	err = parseJSONResponse(triggerListData, &wrongJSON)
	expect(t, err, nil)
	expect(t, wrongJSON.Action.Message, "")
	expect(t, wrongJSON.Derp, nil)
	expect(t, wrongJSON.Dorp, 0)

	var arbitraryJSON map[string]interface{}
	err = parseJSONResponse(triggerListData, &arbitraryJSON)
	expect(t, err, nil)

	var badArray []interface{}
	expectedError := `Error parsing response: {"triggers":[{"triggerId":"6fd01180fa1a012f27f1705681b27197","condition":{"direction":"enter","geo":{"geocode":"920 SW 3rd Ave, Portland, OR","driveTime":600,"context":{"locality":"Portland","region":"Oregon","country":"USA","zipcode":"97204"}}},"action":{"message":"Welcome to Portland - The Mayor","callback":"http://pdx.gov/welcome"},"tags":["foodcarts","citygreetings"]}],"boundingBox":{"xmin":-122.68,"ymin":45.53,"xmax":-122.45,"ymax":45.6}}  Error: json: cannot unmarshal object into Go value of type []interface {}`
	err = parseJSONResponse(triggerListData, &badArray)
	refute(t, err, nil)
	expect(t, err.Error(), expectedError)
	expect(t, badArray, nil)

	var goodArray []interface{}
	err = parseJSONResponse(jsonArrayData, &goodArray)
	expect(t, err, nil)
	expect(t, len(goodArray), 3)

	var badJSON TriggerList
	expectedError = `Error parsing response: ["herp", "derp", "dorp"]  Error: json: cannot unmarshal array into Go value of type geotrigger_golang.TriggerList`
	err = parseJSONResponse(jsonArrayData, &badJSON)
	refute(t, err, nil)
	expect(t, err.Error(), expectedError)
	expect(t, len(badJSON.Triggers), 0)
	expect(t, badJSON.BoundingBox.Xmin, float64(0))

	var notAPointer TriggerList
	err = parseJSONResponse(triggerListData, notAPointer)
	refute(t, err, nil)
	expect(t, err.Error(), "Provided responseJSON interface should be a pointer (to struct or map).")
	expect(t, len(notAPointer.Triggers), 0)
	expect(t, notAPointer.BoundingBox.Xmin, float64(0))

	var notAPointer2 interface{}
	err = parseJSONResponse(triggerListData, notAPointer2)
	refute(t, err, nil)
	expect(t, notAPointer2, nil)
	expect(t, err.Error(), "Provided responseJSON interface should be a pointer (to struct or map).")
}

func TestValueGetters(t *testing.T) {
	var responseJSON map[string]interface{}
	err := json.Unmarshal(triggerListData, &responseJSON)
	expect(t, err, nil)

	// test GetValueFromJSONObject and GetValueFromJSONArray a bit
	var triggers []interface{}
	err = GetValueFromJSONObject(responseJSON, "triggers", &triggers)
	expect(t, err, nil)

	var trigger map[string]interface{}
	err = GetValueFromJSONArray(triggers, 0, &trigger)
	expect(t, err, nil)

	var tags []interface{}
	err = GetValueFromJSONObject(trigger, "tags", &tags)
	expect(t, err, nil)
	expect(t, tags[0], "foodcarts")

	var action map[string]interface{}
	err = GetValueFromJSONObject(trigger, "action", &action)
	expect(t, err, nil)

	var callback string
	err = GetValueFromJSONObject(action, "callback", &callback)
	expect(t, err, nil)

	expect(t, callback, "http://pdx.gov/welcome")

	// works with empty interface
	var emptyInterface interface{}
	err = GetValueFromJSONObject(trigger, "triggerId", &emptyInterface)
	expect(t, err, nil)
	expect(t, emptyInterface, "6fd01180fa1a012f27f1705681b27197")

	// provide nil array/object
	var failureInterface interface{}
	err = GetValueFromJSONObject(nil, "derp", &failureInterface)
	expect(t, failureInterface, nil)
	refute(t, err, nil)
	expect(t, err.Error(), "Attempt to get value from a nil JSON object.")

	err = GetValueFromJSONArray(nil, 1, &failureInterface)
	expect(t, failureInterface, nil)
	refute(t, err, nil)
	expect(t, err.Error(), "Attempt to get value from a nil JSON aray.")

	// empty key
	err = GetValueFromJSONObject(responseJSON, "", &failureInterface)
	expect(t, failureInterface, nil)
	refute(t, err, nil)
	expect(t, err.Error(), "Attempt to pull value for empty key from JSON object.")

	// no matching value in map
	err = GetValueFromJSONObject(responseJSON, "horse with hands", &failureInterface)
	expect(t, failureInterface, nil)
	refute(t, err, nil)
	expect(t, err.Error(), "No value found for key: horse with hands")

	// index out of range
	err = GetValueFromJSONArray(triggers, -3, &failureInterface)
	expect(t, failureInterface, nil)
	refute(t, err, nil)
	expect(t, err.Error(), "Provided index -3 was out of range.")

	err = GetValueFromJSONArray(triggers, 17, &failureInterface)
	expect(t, failureInterface, nil)
	refute(t, err, nil)
	expect(t, err.Error(), "Provided index 17 was out of range.")

	// not a pointer
	err = GetValueFromJSONArray(triggers, 0, failureInterface)
	expect(t, failureInterface, nil)
	refute(t, err, nil)
	expect(t, err.Error(), "Provided value is of invalid type (must be pointer).")

	var notAPointer2 Trigger
	err = GetValueFromJSONArray(triggers, 0, notAPointer2)
	refute(t, err, nil)
	expect(t, err.Error(), "Provided value is of invalid type (must be pointer).")
	expect(t, len(notAPointer2.Tags), 0)
	expect(t, notAPointer2.Condition.Geo.DriveTime, 0)

	var notAPointer3 BoundingBox
	err = GetValueFromJSONObject(responseJSON, "boundingBox", notAPointer3)
	refute(t, err, nil)
	expect(t, err.Error(), "Provided value is of invalid type (must be pointer).")
	expect(t, notAPointer3.Ymin, float64(0))

	// wrong value type!
	var wrongType1 BoundingBox
	err = GetValueFromJSONObject(responseJSON, "triggers", &wrongType1)
	refute(t, err, nil)
	expect(t, err.Error(), "Provided reference is to a value of type geotrigger_golang.BoundingBox that cannot be assigned to type found in JSON: []interface {}.")
	expect(t, notAPointer3.Xmin, float64(0))

	var wrongType2 []interface{}
	err = GetValueFromJSONArray(triggers, 0, &wrongType2)
	refute(t, err, nil)
	expect(t, err.Error(), "Provided reference is to a value of type []interface {} that cannot be assigned to type found in JSON: map[string]interface {}.")
	expect(t, notAPointer3.Xmin, float64(0))
}
