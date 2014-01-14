package geotrigger_golang

import (
	"encoding/json"
	"testing"
)

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

func TestValueGetters(t *testing.T) {
	data := []byte(`{"triggers":[{"triggerId":"6fd01180fa1a012f27f1705681b27197","condition":{"direction":"enter","geo":{"geocode":"920 SW 3rd Ave, Portland, OR","driveTime":600,"context":{"locality":"Portland","region":"Oregon","country":"USA","zipcode":"97204"}}},"action":{"message":"Welcome to Portland - The Mayor","callback":"http://pdx.gov/welcome"},"tags":["foodcarts","citygreetings"]}],"boundingBox":{"xmin":-122.68,"ymin":45.53,"xmax":-122.45,"ymax":45.6}}`)
	var responseJSON map[string]interface{}
	err := json.Unmarshal(data, &responseJSON)
	expect(t, err, nil)
}
