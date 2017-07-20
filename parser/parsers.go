package parser

import (
	"reflect"
)

// Parser represents any kind of object able to assign
// a value as a string to a reflect.Value
// It deals with conversion and might return an error.
type Parser interface {
	Set(strValue string, val reflect.Value) error
}

// ParserFunc is a sugar enabling to define a Parser as a function
type ParserFunc func(string, reflect.Value) error

// Set calls the ParserFunc function
func (p ParserFunc) Set(strValue string, val reflect.Value) error {
	return p(strValue, val)
}
