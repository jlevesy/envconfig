package parser

import (
	"reflect"
	"strconv"
	"time"
)

func setFloat(floatType int) ParserFunc {
	return ParserFunc(func(strValue string, value reflect.Value) error {
		v, err := strconv.ParseFloat(strValue, floatType)
		if err != nil {
			return err
		}

		value.SetFloat(v)

		return nil
	})
}

func setInt(intLength int) ParserFunc {
	return ParserFunc(func(strValue string, value reflect.Value) error {
		v, err := strconv.ParseInt(strValue, 0, intLength)

		if err != nil {
			return err
		}

		value.SetInt(v)

		return nil
	})
}

func setUint(uintLength int) ParserFunc {
	return ParserFunc(func(strValue string, value reflect.Value) error {
		v, err := strconv.ParseUint(strValue, 0, uintLength)

		if err != nil {
			return err
		}

		value.SetUint(v)

		return nil
	})
}

func setString(strValue string, value reflect.Value) error {
	value.SetString(strValue)
	return nil
}

func setBool(strValue string, value reflect.Value) error {
	v, err := strconv.ParseBool(strValue)

	if err != nil {
		return err
	}

	value.SetBool(v)

	return nil
}

func setTime(strValue string, value reflect.Value) error {
	v, err := time.Parse(time.RFC3339, strValue)

	if err != nil {
		return err
	}

	value.Set(reflect.ValueOf(v))

	return nil
}

func setDuration(strValue string, value reflect.Value) error {
	v, err := time.ParseDuration(strValue)

	if err != nil {
		return err
	}

	value.Set(reflect.ValueOf(v))

	return nil
}

// LoadBasicTypes returns a collection of Parser for
// golang basic types.
func LoadBasicTypes() map[reflect.Type]Parser {
	res := make(map[reflect.Type]Parser)

	// Floats
	res[reflect.TypeOf(float64(0.0))] = setFloat(64)
	res[reflect.TypeOf(float32(0.0))] = setFloat(32)

	// Ints
	res[reflect.TypeOf(int(0))] = setInt(0)
	res[reflect.TypeOf(int8(0))] = setInt(8)
	res[reflect.TypeOf(int16(0))] = setInt(16)
	res[reflect.TypeOf(int32(0))] = setInt(32)
	res[reflect.TypeOf(int64(0))] = setInt(64)

	// Uints
	res[reflect.TypeOf(uint(0))] = setUint(0)
	res[reflect.TypeOf(uint8(0))] = setUint(8)
	res[reflect.TypeOf(uint16(0))] = setUint(16)
	res[reflect.TypeOf(uint32(0))] = setUint(32)
	res[reflect.TypeOf(uint64(0))] = setUint(64)

	// Misc
	res[reflect.TypeOf("")] = ParserFunc(setString)
	res[reflect.TypeOf(true)] = ParserFunc(setBool)
	res[reflect.TypeOf(time.Time{})] = ParserFunc(setTime)
	res[reflect.TypeOf(time.Duration(0))] = ParserFunc(setDuration)

	return res
}
