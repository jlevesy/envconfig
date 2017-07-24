package setter

import (
	"reflect"
)

// Setter represents any kind of object able to assign
// a value as a string to a reflect.Value
// It deals with conversion and might return an error.
type Setter interface {
	Set(strValue string, val reflect.Value) error
}

// SetterFunc is a sugar enabling to define a Setter as a function
type SetterFunc func(string, reflect.Value) error

// Set calls the SetterFunc function
func (p SetterFunc) Set(strValue string, val reflect.Value) error {
	return p(strValue, val)
}
