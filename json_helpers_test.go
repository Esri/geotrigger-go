package geotrigger_golang

import (
	"testing"
)

func TestErrorCheck(t *testing.T) {
	resp := []byte(`["array", "root", "element"]`)
	errResponse := errorCheck(resp)
	mustBeNil(t, errResponse)

	resp = []byte(`{"object":"doesnt match", "derp":["dorp", "morp"]}`)
	errResponse = errorCheck(resp)
	mustBeNil(t, errResponse)

	var derp [5]byte
	mustBeNil(t, derp)
}
