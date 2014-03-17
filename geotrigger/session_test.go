package geotrigger

import (
	"github.com/Esri/geotrigger-go/geotrigger/test"
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
	test.Expect(t, errResponse, nil)

	resp = []byte(`{"object":"doesnt match", "derp":["dorp", "morp"]}`)
	errResponse = errorCheck(resp)
	test.Expect(t, errResponse, nil)

	resp = []byte(`{"error":{"code":400,"message":"Invalid token."}}`)
	errResponse = errorCheck(resp)
	test.Refute(t, errResponse, nil)
}

func TestParseJSONResponse(t *testing.T) {
	var responseJSON TriggerList
	err := parseJSONResponse(triggerListData, &responseJSON)
	test.Expect(t, err, nil)

	test.Expect(t, responseJSON.BoundingBox.Xmax, -122.45)
	test.Expect(t, len(responseJSON.Triggers), 1)
	test.Expect(t, responseJSON.Triggers[0].Condition.Geo.Context.Zipcode, "97204")

	var partialJSON TriggerList
	err = parseJSONResponse(partialTriggerListData, &partialJSON)
	test.Expect(t, err, nil)

	// xmax is missing, will be set to zero value
	test.Expect(t, partialJSON.BoundingBox.Xmax, float64(0))
	test.Expect(t, len(partialJSON.Triggers), 1)
	test.Expect(t, partialJSON.Triggers[0].Condition.Geo.Context.Zipcode, "97204")
	// tags are missing, should be nil
	test.Expect(t, partialJSON.Triggers[0].Tags, nil)
	test.Expect(t, partialJSON.Triggers[0].Condition.Geo.Context.Region, "")

	var wrongJSON WrongJSON
	err = parseJSONResponse(triggerListData, &wrongJSON)
	test.Expect(t, err, nil)
	test.Expect(t, wrongJSON.Action.Message, "")
	test.Expect(t, wrongJSON.Derp, nil)
	test.Expect(t, wrongJSON.Dorp, 0)

	var arbitraryJSON map[string]interface{}
	err = parseJSONResponse(triggerListData, &arbitraryJSON)
	test.Expect(t, err, nil)

	var badArray []interface{}
	expectedError := `Error parsing response: {"triggers":[{"triggerId":"6fd01180fa1a012f27f1705681b27197","condition":{"direction":"enter","geo":{"geocode":"920 SW 3rd Ave, Portland, OR","driveTime":600,"context":{"locality":"Portland","region":"Oregon","country":"USA","zipcode":"97204"}}},"action":{"message":"Welcome to Portland - The Mayor","callback":"http://pdx.gov/welcome"},"tags":["foodcarts","citygreetings"]}],"boundingBox":{"xmin":-122.68,"ymin":45.53,"xmax":-122.45,"ymax":45.6}}  Error: json: cannot unmarshal object into Go value of type []interface {}`
	err = parseJSONResponse(triggerListData, &badArray)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), expectedError)
	test.Expect(t, badArray, nil)

	var goodArray []interface{}
	err = parseJSONResponse(jsonArrayData, &goodArray)
	test.Expect(t, err, nil)
	test.Expect(t, len(goodArray), 3)

	var badJSON TriggerList
	expectedError = `Error parsing response: ["herp", "derp", "dorp"]  Error: json: cannot unmarshal array into Go value of type geotrigger.TriggerList`
	err = parseJSONResponse(jsonArrayData, &badJSON)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), expectedError)
	test.Expect(t, len(badJSON.Triggers), 0)
	test.Expect(t, badJSON.BoundingBox.Xmin, float64(0))

	var notAPointer TriggerList
	err = parseJSONResponse(triggerListData, notAPointer)
	test.Refute(t, err, nil)
	test.Expect(t, err.Error(), "Provided responseJSON interface should be a pointer (to struct or map).")
	test.Expect(t, len(notAPointer.Triggers), 0)
	test.Expect(t, notAPointer.BoundingBox.Xmin, float64(0))

	var notAPointer2 interface{}
	err = parseJSONResponse(triggerListData, notAPointer2)
	test.Refute(t, err, nil)
	test.Expect(t, notAPointer2, nil)
	test.Expect(t, err.Error(), "Provided responseJSON interface should be a pointer (to struct or map).")
}
