package test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// mustBeNil()/mustNotBeNil() isNillable() author: courtf
func MustBeNil(t *testing.T, a interface{}) {
	tp := reflect.TypeOf(a)

	if tp != nil && (!IsNillable(tp.Kind()) || !reflect.ValueOf(a).IsNil()) {
		t.Errorf("Got nil, but expected %v (type %v)", a, tp)
	}
}

func MustNotBeNil(t *testing.T, a interface{}) {
	tp := reflect.TypeOf(a)

	if tp == nil || (IsNillable(tp.Kind()) && reflect.ValueOf(a).IsNil()) {
		t.Errorf("Got nil and did not expect nil.")
	}
}

func IsNillable(k reflect.Kind) (nillable bool) {
	kinds := []reflect.Kind{
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Ptr,
		reflect.Slice,
	}

	for i := 0; i < len(kinds); i++ {
		if kinds[i] == k {
			nillable = true
			break
		}
	}

	return
}

// restorer and patch adapted from https://gist.github.com/imosquera/6716490
// thanks imosquera!
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

	// we know destination is a pointer, so get the type of value being pointed to
	destType = destType.Elem()
	// compare that type to the type of v
	providedType := reflect.TypeOf(v)
	if !destType.AssignableTo(providedType) {
		return nil, fmt.Errorf("Provided value of type %s cannot be assigned to type: %s.",
			providedType, destType)
	}

	// get the value being pointed to
	destValue := reflect.ValueOf(destination).Elem()
	// reflect.New creates a new pointer value to provided type, elem gets the pointed to value again
	oldValue := reflect.New(destType).Elem()
	// we then set that value to the current destination value to hold onto it
	oldValue.Set(destValue)

	// the value of the provided interface{}
	value := reflect.ValueOf(v)
	if !value.IsValid() {
		// This should be a rare occurrence.
		// the value provided could not be reflected, and we have an invalid Value here
		// so just attempt to use the zero value for the destination type.
		value = reflect.Zero(destType)
	}

	// replace the destination's current value with the value of the provided v
	// this shouldn't panic, because we have already checked that they are the same type
	destValue.Set(value)
	return func() {
		// restore the destination's value to its original
		destValue.Set(oldValue)
	}, nil
}

// Warning: directly comparing functions is unreliable
func Expect(t *testing.T, a interface{}, b interface{}) {
	btype := reflect.TypeOf(b)
	if b == nil {
		MustBeNil(t, a)
	} else if btype.Kind() == reflect.Func {
		if reflect.ValueOf(a).Pointer() != reflect.ValueOf(b).Pointer() {
			t.Errorf("Expected func %v (type %v) to equal func %v (type %v).", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
		}
	} else if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

// Warning: directly comparing functions is unreliable
func Refute(t *testing.T, a interface{}, b interface{}) {
	btype := reflect.TypeOf(b)
	if b == nil {
		MustNotBeNil(t, a)
	} else if btype.Kind() == reflect.Func {
		if reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() {
			t.Errorf("Expected func %v (type %v) to not equal func %v (type %v).", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
		}
	} else if reflect.DeepEqual(a, b) {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}
