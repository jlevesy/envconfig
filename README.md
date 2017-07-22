# EnvConfig

[![Build Status](https://travis-ci.org/jlevesy/envconfig.svg?branch=master)](https://travis-ci.org/jlevesy/envconfig)
[![Go Report Card](https://goreportcard.com/badge/github.com/jlevesy/envconfig)](https://goreportcard.com/report/github.com/jlevesy/envconfig)
[![codecov](https://codecov.io/gh/jlevesy/envconfig/branch/master/graph/badge.svg)](https://codecov.io/gh/jlevesy/envconfig)


EnvConfig is a go library which enables you to populate a struct according to
the process environment. It infers environment variables names according to struct
field names.

It fully supports complex structures involving maps, arrays and slices, also it
deals with allocation.

However, at the moment it doesn't support type loops, ie (TypeA => TypeB => TypeA ...)

## Getting Started

Here's a basic example.

```go
package example

import (
	"github.com/jlevesy/envconfig/parser"
	"github.com/jlevesy/envconfig"
)

const (
  AppPrefix = "TEST"
  Separator = "_"
)

//Configuration is a struct which contains all differents type to field
type Configuration struct {
	IntField     int
	StringField  string
	PointerField *PointerSubConfiguration
}

//PointerSubConfiguration is a SubStructure Configuration
type PointerSubConfiguration struct {
	BoolField  bool
	FloatField float64
}
```

Let's initialize it:

```go
 func main() {
        config := &Configuration{}

        // load your structure
	if err := envconfig.New(AppPrefix, Separator).Load(config); err != nil {
		fmt.Println("Failed to load config, got: ", err)
		os.Exit(1)
	}

        // Use your filled structure
}
```

Now if I run...

```
  TEST_INT_FIELD=10 TEST_POINTER_FIELD_BOOL_FIELD=1 go run main.go
```

... `config.IntField` will be set to 10, and `config.PointerField.BoolField` to
true !

And that's pretty much it ! If you need more details there is a detailed
[example](https://github.com/jlevesy/envconfig/tree/master/example).

## Under the hood

### Initialization

It can be initalized like this :

```
        prefix := "FOO"
        speparator := "/"
        env := envconfig.New(prefix, separator)
```

It takes two arguments:

- A prefix used in order to format environment variables names to fetch, if left
  blank, no prefix will be applied to environment variables
- A separator string, if left blank it will default to the "_" string

Another constructor is available

```
        env := envconfig.NewWithParsersAndDepth(prefix, separator, parsers, maxDepth)
```

It adds two more arguments

- A parser collection which  is a `map[reflect.Type]parser.Parser` representing
  all types envConfig can write to.
- A maxdepth, setting a hard limit on structure depth to avoid type loops.

`envconfig.New(prefix, separator)`, is equivalent to `envconfig.NewWithParsersAndDepth(prefix, separator,
parser.LoadBasicTypes(), 10)`

### Environment variable name inference

Environment variable names are structured like this:

```
[PREFIX][SEP][MY][SEP][FIELD][SEP][NAME]
```

Field names are split into by words according to camelCase, we rely on
[github.com/fatih/camelcase](https://github.com/fatih/camelcase) to do this.

For instance if prefix is "MyApp" and separator is the '_' rune we'll have the following mapping:

```go
type AppStruct struct {
    MyStringField string // => MYAPP_MY_STRING_FIELD
    MyIntField    int    // => MYAPP_MY_INT_FIELD
}
```

### Embedded structures

Embedded structures are supported, and environment variable name generation for a field
will have exact same behaviour than normal struct field.

For instance if we keep our previous example confifuration, we'll obtain the
following mapping:

```go
type CommonConfig struct {
    CommonString string // => MYAPP_COMMON_STRING
}

type AppConfig struct {
    CommonConfig
}
```

### Nested structures

Nested structures are also supported, both by pointer and values. However
fields names are going to be prefixed with the field name referencing the
nested structure:

```go

type PtrNestedConfig struct {
    AnArgument string // => MY_APP_FOO_AN_ARGUMENT
}

type ValueNestedConfig struct {
    AnotherArgument string // => MY_APP_BAR_ANOTHER_ARGUMENT
}

type AppConfig struct {
    Foo   **PtrNestedConfig // => Double indirection because why not ?
    Bar   ValueNestedConfig
}

```

### Pointed values

You can also use pointer to values too in your config structs,
those fields are going to be mapped exactly as a value.

```go
type AppConfig struct {
  Groot *int32 // => MYAPP_GROOT
}
```

### Array an slices

You can affect values into array and slices using environment variables.
Index affectation is not guaranteed, but ordering is.

```go
type NestedAppConfig struct {
    BoolValue bool // => MY_APP_BAR_<INT_INDEX>_BOOL_VALUE
}

type AppConfig struct {
    Foo []string // => MY_APP_FOO_<INT_INDEX>
    Bar []*NestedAppConfig
}
```

### Maps

You can affect values into maps, just like arrays and slices, however key type
must be supported by the parser colllection.

```go
type NestedAppConfig struct {
    BoolValue bool // => MY_APP_BAR_<KEY>_BOOL_VALUE
}

type AppConfig struct {
    Foo map[int]string // => MY_APP_FOO_<KEY>
    Bar map[float64]*NestedAppConfig
}
```

### The Parser interface

EnvConfig depends on a parser collection representing all types it can
write to.

A Parser is defined by the following interface.

```
type Parser interface {
	Set(strValue string, val reflect.Value) error
}
```

If you need to support different types, for instance an IP address, feel free to
define your very own `Parser` or `ParserFunc`, and add it to your parser
collection at initialization.

## Todo

- [ ] Support custom environment variable names using tags
- [ ] Better structure loop detection

Of course, any suggestions are welcome ! :)

## Contributing
1. Fork it!
2. Create your feature branch: `git checkout -b my-new-feature`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Push to the branch: `git push origin my-new-feature`
5. Submit a pull request :D
