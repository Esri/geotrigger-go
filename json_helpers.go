package geotrigger_golang

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// GetValueFromJSONObject is a helper function for unpacking values out of
// arbitrary JSON objects. `value` should be a pointer to a value of the type
// you expect to retrieve. Inner objects are of type `map[string]interface{}`.
// Inner arrays are of type `[]interface{}`. You can pass down a pointer to an
// `interface{}`, but then you are really better off not using these helpers, as
// they use reflection to try and match types.
func GetValueFromJSONObject(jsonObject map[string]interface{}, key string, value interface{}) error {
	if jsonObject == nil {
		return errors.New("Attempt to get value from a nil JSON object.")
	}

	if len(key) == 0 {
		return errors.New("Attempt to pull value for empty key from JSON object.")
	}

	jsonVal, gotVal := jsonObject[key]
	if !gotVal {
		return fmt.Errorf("No value found for key: %s", key)
	}

	return setVal(value, jsonVal)
}

// GetValueFromJSONArray is a helper function for unpacking values out of
// arbitrary JSON arrays. `value` should be a pointer to a value of the type you
// expect to retrieve. Inner objects are of type `map[string]interface{}`. Inner
// arrays are of type `[]interface{}`. You can pass down a pointer to an
// `interface{}`, but then you are really better off not using these helpers, as
// they use reflection to try and match types.
func GetValueFromJSONArray(jsonArray []interface{}, index int, value interface{}) error {
	if jsonArray == nil {
		return errors.New("Attempt to get value from a nil JSON aray.")
	}

	if index < 0 || index >= len(jsonArray) {
		return fmt.Errorf("Provided index %d was out of range.", index)
	}

	jsonVal := jsonArray[index]

	return setVal(value, jsonVal)
}

func setVal(value interface{}, jsonVal interface{}) (err error) {
	// make sure the interface provided is a pointer, so that we can modify the value
	expectedType := reflect.TypeOf(value)
	if expectedType == nil || expectedType.Kind() != reflect.Ptr {
		return errors.New("Provided value is of invalid type (must be pointer).")
	}

	// we know it's a pointer, so get the type of value being pointed to
	expectedType = expectedType.Elem()

	// compare that type to the type pulled from the JSON
	actualType := reflect.TypeOf(jsonVal)

	if !actualType.AssignableTo(expectedType) {
		return fmt.Errorf(
			"Provided reference is to a value of type %s that cannot be assigned to type found in JSON: %s.",
			expectedType, actualType)
	}

	// recover from any panics that might occur below, although we should be safe
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Panic during assignment from JSON to provided interface value.")
			}
		}
	}()

	// Time to set the new value being pointed to by the passed in interface.
	// We know it's a pointer, so its value will be a reference to the value
	// we are actually interested in changing.
	pv := reflect.ValueOf(value)
	// Elem() gets the value being pointed to,
	v := pv.Elem()
	// and we can set it directly to what we found in the JSON, since we
	// have already checked that they the JSON value is assignable.
	v.Set(reflect.ValueOf(jsonVal))
	return
}

func errorCheck(resp []byte) *errorResponse {
	var errorContainer errorResponse
	if err := json.Unmarshal(resp, &errorContainer); err != nil {
		// Don't return an error here, as it is possible for the response
		// to not be parsed into an errorResponse, causing an error to be thrown, but still
		// be valid, ie: the root element of the response is an array.
		// We are just looking to see if we can spot a known server error anyway.
		return nil
	}

	if errorContainer.Error.Code > 0 && len(errorContainer.Error.Message) > 0 {
		return &errorContainer
	}

	return nil
}

func parseJSONResponse(resp []byte, responseJSON interface{}) error {
	t := reflect.TypeOf(responseJSON)
	if t == nil || t.Kind() != reflect.Ptr {
		return fmt.Errorf("Provided responseJSON interface should be a pointer (to struct or map).")
	}

	if err := json.Unmarshal(resp, responseJSON); err != nil {
		return fmt.Errorf("Error parsing response: %s  Error: %s", string(resp), err)
	}

	return nil
}
