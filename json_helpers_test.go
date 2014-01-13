package geotrigger_golang

import (
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
