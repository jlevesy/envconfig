package envconfig

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/jlevesy/envconfig/setter"

	"github.com/fatih/camelcase"
)

const (
	// DefaultDepth is the default maximum depth allowed for a struct
	DefaultDepth = 10

	envConfigTag = "envconfig"
	noExpand     = "noexpand"
)

// ConfigLoader interface is an object that can be used to Loader
// data into a configuration structure
type ConfigLoader interface {
	Load(config interface{}) error
}

// envConfig implements ConfigLoader
// Enables to populate configuration struct with informations extracted from
// process's environment variables.
// Variables names are like %PREFIX%%SEP%%FIELD_NAME%
type envConfig struct {
	prefix    string
	separator string
	setters   map[reflect.Type]setter.Setter
	maxDepth  int
}

// NewWithSettersAndDepth constructs a new instance of envConfig
// It allows to setup prefix, separator supported setters and maximum structure depth.
func NewWithSettersAndDepth(prefix, separator string, setters map[reflect.Type]setter.Setter, maxDepth int) ConfigLoader {
	return &envConfig{prefix, separator, setters, maxDepth}
}

// New returns a new instance of envConfig with given prefix and separator.
func New(prefix, separator string) ConfigLoader {
	return NewWithSettersAndDepth(prefix, separator, setter.LoadBasicTypes(), DefaultDepth)
}

// Load loads environment data into given configuration structure
func (e *envConfig) Load(config interface{}) error {

	configVal := reflect.ValueOf(config)

	if configVal.Kind() != reflect.Ptr {
		return errors.New("Passing by value isn't supported, please provide a pointer")
	}

	configVal = configVal.Elem()
	configType := configVal.Type()

	values, err := e.analyzeStruct(configType, []string{})

	if err != nil {
		return err
	}

	return e.assignValues(configVal, configType, values)
}

// path represents path to a value in a struct
type path []string

func (p path) clone() path {
	res := make(path, len(p))
	copy(res, p)
	return res
}

func (p path) popBack() (string, path) {
	if len(p) == 1 {
		return p[0], path{}
	}
	return p[0], p[1:]
}

// envValue represents a defined string value at a path
type envValue struct {
	StrValue string
	Path     path
}

// Recursively scan the given config structure type information
// and look for defined environment variables.
// Returns discovered values as a slice of *envValue
func (e *envConfig) analyzeStruct(configType reflect.Type, currentPath path) ([]*envValue, error) {
	res := []*envValue{}

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		if field.Type.Kind() == reflect.Ptr && indirectedType(field.Type) == configType {
			return []*envValue{}, fmt.Errorf("Recursive type detected %v in field %s", field.Type, field.Name)
		}

		// If we're facing an embedded struct
		if field.Anonymous {

			// Silently ignore interface types
			if field.Type.Kind() == reflect.Interface {
				continue
			}
			values, err := e.analyzeStruct(field.Type, currentPath)

			if err != nil {
				return []*envValue{}, err
			}

			res = append(res, values...)
			continue
		}

		fieldPath := append(currentPath, field.Name)

		if t, ok := field.Tag.Lookup(envConfigTag); ok {
			if t == noExpand {
				if v := e.loadValue(fieldPath); v != nil {
					res = append(res, v)
				}
			}

			continue
		}

		values, err := e.analyzeValue(field.Type, fieldPath)

		if err != nil {
			return []*envValue{}, err
		}

		res = append(res, values...)
	}

	return res, nil
}

func (e *envConfig) analyzeValue(valType reflect.Type, fieldPath path) ([]*envValue, error) {
	var (
		res []*envValue
		err error
	)

	if len(fieldPath) > e.maxDepth {
		return res, errors.New("Maxdepth exceeded, you might have a type loop in your structure")
	}

	switch valType.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		res, err = e.analyzeIndexedType(valType, fieldPath)
	case reflect.Ptr:
		res, err = e.analyzeValue(valType.Elem(), fieldPath)
	case reflect.Struct:
		res, err = e.analyzeStruct(valType, fieldPath)
	case reflect.Invalid, reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer:
		err = fmt.Errorf("type %s is not supported by EnvSource", valType.Name())
	default:
		if v := e.loadValue(fieldPath); v != nil {
			res = append(res, v)
		}
	}

	return res, err
}

func (e *envConfig) analyzeIndexedType(valType reflect.Type, fieldPath path) ([]*envValue, error) {
	var (
		res []*envValue
	)

	prefix := e.envVarFromPath(fieldPath)
	vars := e.envVarsWithPrefix(prefix)
	nextKeys := unique(e.nextLevelKeys(prefix, vars))

	for _, varName := range nextKeys {
		key := e.keyFromEnvVar(varName, prefix)

		// If we're on an Int based key, we need to be able to convert
		// detected key to an int
		if valType.Kind() == reflect.Array ||
			valType.Kind() == reflect.Slice {
			index, err := strconv.ParseUint(key, 10, 64)

			if err != nil {
				return res, fmt.Errorf(
					"Key [%s] is not usable as an int index in [%s]",
					key,
					varName,
				)

			}

			if valType.Kind() == reflect.Array &&
				int(index) >= valType.Len() {
				return res, fmt.Errorf(
					"Detected key (%s) from variable %s is >= to array length %d",
					key,
					varName,
					valType.Len(),
				)
			}
		}

		valPath := append(fieldPath, key)
		keyValues, err := e.analyzeValue(valType.Elem(), valPath)
		if err != nil {
			return res, err
		}

		res = append(res, keyValues...)
	}

	return res, nil
}

func (e *envConfig) loadValue(fieldPath path) *envValue {
	variableName := e.envVarFromPath(fieldPath)

	value, ok := os.LookupEnv(variableName)

	if !ok {
		return nil
	}

	return &envValue{value, fieldPath.clone()}
}

func (e *envConfig) assignValues(configVal reflect.Value, configType reflect.Type, values []*envValue) error {
	for _, v := range values {
		if err := e.assignValue(configVal, configType, v.Path, v.StrValue); err != nil {
			return err
		}
	}
	return nil
}

func (e *envConfig) assignValue(val reflect.Value, valType reflect.Type, currentPath path, strValue string) error {
	var err error
	switch valType.Kind() {
	case reflect.Ptr:
		val, valType, err = e.allocate(val, valType)
		if err != nil {
			return err
		}

		err = e.assignValue(val, valType, currentPath, strValue)
	case reflect.Struct:
		err = e.assignToStruct(val, valType, currentPath, strValue)
	case reflect.Slice:
		err = e.assignToSlice(val, valType, currentPath, strValue)
	case reflect.Array:
		err = e.assignToArray(val, valType, currentPath, strValue)
	case reflect.Map:
		err = e.assignToMap(val, valType, currentPath, strValue)
	case reflect.Invalid, reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer:
		err = fmt.Errorf("type %s is not supported by EnvSource", valType.Name())
	default:
		err = e.setValue(val, strValue)
	}

	return err
}

func (e *envConfig) assignToStruct(val reflect.Value, valType reflect.Type, currentPath path, strValue string) error {
	fieldName, currentPath := currentPath.popBack()

	structField, ok := valType.FieldByName(fieldName)

	if !ok {
		return fmt.Errorf("Unexpected error: failed to get field [%s] in config struct [%v]", fieldName, valType)
	}

	valType = structField.Type
	val = val.FieldByName(fieldName)

	// If we're dealing with a noexpand struct
	// Directly perform allocation then intent to set value
	if t, ok := structField.Tag.Lookup(envConfigTag); ok {
		if t == noExpand {
			val, _, err := e.allocate(val, valType)
			if err != nil {
				return err
			}
			return e.setValue(val, strValue)
		}
	}

	return e.assignValue(val, valType, currentPath, strValue)
}

func (e *envConfig) assignToSlice(slice reflect.Value, sliceType reflect.Type, currentPath path, strValue string) error {
	key, currentPath := currentPath.popBack()

	indexU64, err := strconv.ParseUint(key, 10, 64)

	if err != nil {
		return err
	}

	index := int(indexU64)

	var elemValue reflect.Value
	elemType := sliceType.Elem()

	if index < slice.Len() {
		elemValue = slice.Index(index)
	} else {
		elemValue = reflect.New(elemType).Elem()
	}

	if err := e.assignValue(elemValue, elemType, currentPath, strValue); err != nil {
		return err
	}

	if index >= slice.Len() {
		slice.Set(reflect.Append(slice, elemValue))
	}

	return nil
}

func (e *envConfig) assignToArray(array reflect.Value, arrayType reflect.Type, currentPath path, strValue string) error {
	key, currentPath := currentPath.popBack()

	indexU64, err := strconv.ParseUint(key, 10, 64)

	if err != nil {
		return err
	}

	index := int(indexU64)

	var elemValue reflect.Value
	elemType := arrayType.Elem()

	if index >= array.Len() {
		return fmt.Errorf("Index [%d] is overflowing array of length [%d]", index, array.Len())
	}

	elemValue = array.Index(index)

	return e.assignValue(elemValue, elemType, currentPath, strValue)
}

func (e *envConfig) assignToMap(mapValue reflect.Value, mapType reflect.Type, currentPath path, strValue string) error {
	keyString, currentPath := currentPath.popBack()

	keyValue := reflect.New(mapType.Key()).Elem()

	if err := e.setValue(keyValue, keyString); err != nil {
		return err
	}

	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapType))
	}

	elemValue := mapValue.MapIndex(keyValue)
	elemType := mapType.Elem()

	if !elemValue.IsValid() {
		elemValue = reflect.New(elemType).Elem()
	}

	if err := e.assignValue(elemValue, elemType, currentPath, strValue); err != nil {
		return err
	}

	mapValue.SetMapIndex(keyValue, elemValue)

	return nil
}

func (e *envConfig) allocate(val reflect.Value, valType reflect.Type) (reflect.Value, reflect.Type, error) {
	if valType.Kind() != reflect.Ptr {
		return val, valType, nil
	}

	if !val.IsValid() {
		return val, valType, fmt.Errorf("Cannot allocate to an invalid value")
	}

	if !val.IsNil() {
		return val.Elem(), valType.Elem(), nil
	}

	val.Set(reflect.New(valType.Elem()))

	if valType.Elem().Kind() == reflect.Ptr {
		return e.allocate(val.Elem(), valType.Elem())
	}

	return val.Elem(), valType.Elem(), nil
}

func (e *envConfig) setValue(value reflect.Value, strValue string) error {
	if !value.CanSet() {
		return fmt.Errorf("Value [%v] cannot be set", value)
	}

	setter, ok := e.setters[value.Type()]

	if !ok {
		return fmt.Errorf(
			"Unsupported type [%s], please consider adding custom setter",
			value.Type().String(),
		)
	}

	return setter.Set(strValue, value)
}

func (e *envConfig) nextLevelKeys(prefix string, envVars []string) []string {
	res := make([]string, 0, len(envVars))

	for _, envVar := range envVars {
		nextKey := strings.Split(
			strings.TrimPrefix(envVar, prefix+e.separator),
			e.separator,
		)[0]
		res = append(res, prefix+e.separator+nextKey)

	}

	return res
}

func (e *envConfig) envVarsWithPrefix(prefix string) []string {
	res := []string{}

	for _, rawVar := range os.Environ() {
		varName := strings.Split(rawVar, "=")[0]
		if strings.HasPrefix(varName, prefix) {
			res = append(res, varName)
		}
	}

	return res
}

func (e *envConfig) keyFromEnvVar(fullVar, prefix string) string {
	return strings.ToLower(
		strings.Split(
			strings.TrimPrefix(fullVar, prefix+e.separator),
			e.separator,
		)[0],
	)
}

func (e *envConfig) envVarFromPath(currentPath []string) string {
	if e.prefix != "" {
		currentPath = append([]string{e.prefix}, currentPath...)
	}
	s := make([]string, 0, len(currentPath))

	for _, word := range currentPath {
		s = append(s, camelcase.Split(word)...)
	}

	return strings.ToUpper(strings.Join(s, e.separator))
}

func unique(in []string) []string {
	collector := map[string]struct{}{}
	res := []string{}

	for _, v := range in {
		if _, ok := collector[v]; ok {
			continue
		}

		collector[v] = struct{}{}
		res = append(res, v)
	}

	return res
}

func indirectedType(elemType reflect.Type) reflect.Type {
	if elemType.Kind() == reflect.Ptr {
		return indirectedType(elemType.Elem())
	}
	return elemType
}
