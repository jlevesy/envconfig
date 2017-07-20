package envconfig

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/jlevesy/envconfig/parser"
)

func setupEnv(env map[string]string) {
	for k, v := range env {
		os.Setenv(k, v)
	}

}
func cleanupEnv(env map[string]string) {
	for k := range env {
		os.Unsetenv(k)
	}
}

type basicAppConfig struct {
	StringValue string
	IntValue    int
	BoolValue   bool
}

type recursiveAppConfig struct {
	Config *****recursiveAppConfig
}

type loopStructureA struct {
	Inner *loopStructureB
}

type loopStructureB struct {
	Inner *loopStructureA
}

type Yoloer interface {
	Yolo() error
}

type sortableEnvValues []*envValue

func (s sortableEnvValues) Len() int {
	return len(s)
}

func (s sortableEnvValues) Less(i, j int) bool {
	return s[i].StrValue < s[j].StrValue
}

func (s sortableEnvValues) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type testAnalyzeStructThenHook func(t *testing.T, expectation, result sortableEnvValues, err error)

func testAnalyzeStructShouldSucceed(t *testing.T, expectation, result sortableEnvValues, err error) {
	if err != nil {
		t.Logf("Weren't expecting an error, got [%v]", err)
		t.FailNow()
	}

	if len(expectation) != len(result) {
		t.Logf("Unexpected count of values returned: Expected [%d] got [%d]", len(expectation), len(result))
		t.FailNow()
	}

	// Sort by value, according to StrValue (which might not be the best
	// idea ever), in order to ensure index based comparison consistency
	sort.Sort(expectation)
	sort.Sort(result)

	for i, v := range expectation {
		if v.StrValue != result[i].StrValue {
			t.Logf("Expected [%v] got [%v]", *v, *result[i])
			t.Fail()
		}

		if len(v.Path) != len(result[i].Path) {
			t.Logf("Expected Path length of [%v] got [%v]", len(v.Path), len(result[i].Path))
			t.FailNow()
		}

		for j, p := range v.Path {
			if p != result[i].Path[j] {
				t.Logf("Expected path term [%v] got [%v]", p, result[i].Path[j])
				t.Fail()
			}
		}

	}
}

func testAnalyzeStructShouldFail(t *testing.T, expectation, result sortableEnvValues, err error) {
	if err == nil {
		t.Logf("Expected an error, got nothing")
		t.Fail()
	}
}

func TestAnalyzeStruct(t *testing.T) {
	subject := &envConfig{"", "_", map[reflect.Type]parser.Parser{}, 10}

	testCases := []struct {
		Label       string
		Source      interface{}
		Expectation []*envValue
		Env         map[string]string
		Then        testAnalyzeStructThenHook
	}{
		{
			"WithBasicConfiguration",
			&basicAppConfig{},
			[]*envValue{
				&envValue{"FOOO", path{"StringValue"}},
				&envValue{"10", path{"IntValue"}},
				&envValue{"true", path{"BoolValue"}},
			},
			map[string]string{
				"STRING_VALUE": "FOOO",
				"INT_VALUE":    "10",
				"BOOL_VALUE":   "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithEmbeddedStruct",
			&struct {
				basicAppConfig
				FloatValue float32
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"StringValue"}},
				&envValue{"10", path{"IntValue"}},
				&envValue{"true", path{"BoolValue"}},
				&envValue{"42.1", path{"FloatValue"}},
			},
			map[string]string{
				"STRING_VALUE": "FOOO",
				"INT_VALUE":    "10",
				"BOOL_VALUE":   "true",
				"FLOAT_VALUE":  "42.1",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithEmbeddedIface",
			&struct {
				Yoloer
				StringValue string
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"StringValue"}},
			},
			map[string]string{
				"STRING_VALUE": "FOOO",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithNestedStructValue",
			&struct {
				Config basicAppConfig
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Config", "StringValue"}},
				&envValue{"10", path{"Config", "IntValue"}},
				&envValue{"true", path{"Config", "BoolValue"}},
			},
			map[string]string{
				"CONFIG_STRING_VALUE": "FOOO",
				"CONFIG_INT_VALUE":    "10",
				"CONFIG_BOOL_VALUE":   "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithDoubleNestedStructValue",
			&struct {
				Nested struct {
					Config basicAppConfig
				}
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Nested", "Config", "StringValue"}},
				&envValue{"10", path{"Nested", "Config", "IntValue"}},
				&envValue{"true", path{"Nested", "Config", "BoolValue"}},
			},
			map[string]string{
				"NESTED_CONFIG_STRING_VALUE": "FOOO",
				"NESTED_CONFIG_INT_VALUE":    "10",
				"NESTED_CONFIG_BOOL_VALUE":   "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithRecursiveStruct",
			&recursiveAppConfig{},
			[]*envValue{},
			map[string]string{},
			testAnalyzeStructShouldFail,
		},
		{
			"WithLoopStruct",
			&loopStructureA{},
			[]*envValue{},
			map[string]string{},
			testAnalyzeStructShouldFail,
		},
		{
			"WithNestedStructPtr",
			&struct {
				Config *basicAppConfig
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Config", "StringValue"}},
				&envValue{"10", path{"Config", "IntValue"}},
				&envValue{"true", path{"Config", "BoolValue"}},
			},
			map[string]string{
				"CONFIG_STRING_VALUE": "FOOO",
				"CONFIG_INT_VALUE":    "10",
				"CONFIG_BOOL_VALUE":   "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithDoubleNestedStructPtr",
			&struct {
				Nested *struct {
					Config *basicAppConfig
				}
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Nested", "Config", "StringValue"}},
				&envValue{"10", path{"Nested", "Config", "IntValue"}},
				&envValue{"true", path{"Nested", "Config", "BoolValue"}},
			},
			map[string]string{
				"NESTED_CONFIG_STRING_VALUE": "FOOO",
				"NESTED_CONFIG_INT_VALUE":    "10",
				"NESTED_CONFIG_BOOL_VALUE":   "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithDoubleNestedStructMixed",
			&struct {
				Nested *struct {
					Config basicAppConfig
				}
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Nested", "Config", "StringValue"}},
				&envValue{"10", path{"Nested", "Config", "IntValue"}},
				&envValue{"true", path{"Nested", "Config", "BoolValue"}},
			},
			map[string]string{
				"NESTED_CONFIG_STRING_VALUE": "FOOO",
				"NESTED_CONFIG_INT_VALUE":    "10",
				"NESTED_CONFIG_BOOL_VALUE":   "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithPtrValue",
			&struct {
				IntValue *int
			}{},
			[]*envValue{
				&envValue{"10", path{"IntValue"}},
			},
			map[string]string{
				"INT_VALUE": "10",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithNestedPtrValue",
			&struct {
				Config struct {
					IntValue *int
				}
			}{},
			[]*envValue{
				&envValue{"10", path{"Config", "IntValue"}},
			},
			map[string]string{
				"CONFIG_INT_VALUE": "10",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithNestedPtrValue",
			&struct {
				Config struct {
					IntValue *int
				}
			}{},
			[]*envValue{
				&envValue{"10", path{"Config", "IntValue"}},
			},
			map[string]string{
				"CONFIG_INT_VALUE": "10",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithPtrPtrToValue",
			&struct {
				Config **int
			}{},
			[]*envValue{
				&envValue{"10", path{"Config"}},
			},
			map[string]string{
				"CONFIG": "10",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithPtrPtrToStruct",
			&struct {
				Config **basicAppConfig
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Config", "StringValue"}},
				&envValue{"10", path{"Config", "IntValue"}},
				&envValue{"true", path{"Config", "BoolValue"}},
			},
			map[string]string{
				"CONFIG_STRING_VALUE": "FOOO",
				"CONFIG_INT_VALUE":    "10",
				"CONFIG_BOOL_VALUE":   "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithMapOfValues",
			&struct {
				Config map[string]string
			}{},
			[]*envValue{
				&envValue{"FOO", path{"Config", "foo"}},
				&envValue{"MEH", path{"Config", "bar"}},
				&envValue{"BAR", path{"Config", "biz"}},
			},
			map[string]string{
				"CONFIG_FOO": "FOO",
				"CONFIG_BAR": "MEH",
				"CONFIG_BIZ": "BAR",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithMapOfStructValues",
			&struct {
				Config map[string]basicAppConfig
			}{},
			[]*envValue{
				&envValue{"FOO", path{"Config", "foo", "StringValue"}},
				&envValue{"MEH", path{"Config", "bar", "StringValue"}},
				&envValue{"BAR", path{"Config", "biz", "StringValue"}},
			},
			map[string]string{
				"CONFIG_FOO_STRING_VALUE": "FOO",
				"CONFIG_BAR_STRING_VALUE": "MEH",
				"CONFIG_BIZ_STRING_VALUE": "BAR",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithMapOfStructPtr",
			&struct {
				Config map[string]*basicAppConfig
			}{},
			[]*envValue{
				&envValue{"FOO", path{"Config", "foo", "StringValue"}},
				&envValue{"MEH", path{"Config", "bar", "StringValue"}},
				&envValue{"BAR", path{"Config", "biz", "StringValue"}},
			},
			map[string]string{
				"CONFIG_FOO_STRING_VALUE": "FOO",
				"CONFIG_BAR_STRING_VALUE": "MEH",
				"CONFIG_BIZ_STRING_VALUE": "BAR",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithMapOfMapOfPtrStruct",
			&struct {
				Config map[int]map[string]*basicAppConfig
			}{},
			[]*envValue{
				&envValue{"FOO", path{"Config", "0", "foo", "StringValue"}},
				&envValue{"MEH", path{"Config", "1", "bar", "StringValue"}},
				&envValue{"BAR", path{"Config", "0", "biz", "StringValue"}},
			},
			map[string]string{
				"CONFIG_0_FOO_STRING_VALUE": "FOO",
				"CONFIG_1_BAR_STRING_VALUE": "MEH",
				"CONFIG_0_BIZ_STRING_VALUE": "BAR",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithSliceToValue",
			&struct {
				Config []int
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Config", "0"}},
				&envValue{"10", path{"Config", "1"}},
				&envValue{"true", path{"Config", "2"}},
			},
			map[string]string{
				"CONFIG_0": "FOOO",
				"CONFIG_1": "10",
				"CONFIG_2": "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithSliceToValueAndInvalidKey",
			&struct {
				Config []int
			}{},
			[]*envValue{},
			map[string]string{
				"CONFIG_0":      "FOOO",
				"CONFIG_1":      "10",
				"CONFIG_PATATE": "true",
			},
			testAnalyzeStructShouldFail,
		},
		{
			"WithAnArrayToValue",
			&struct {
				Config [10]int
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Config", "0"}},
				&envValue{"10", path{"Config", "1"}},
				&envValue{"true", path{"Config", "2"}},
			},
			map[string]string{
				"CONFIG_0": "FOOO",
				"CONFIG_1": "10",
				"CONFIG_2": "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithAnArrayAndAnOutOfBoundIndex",
			&struct {
				Config [10]int
			}{},
			[]*envValue{},
			map[string]string{
				"CONFIG_11": "10",
			},
			testAnalyzeStructShouldFail,
		},
		{
			"WithAnArrayToValue",
			&struct {
				Config [10]int
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Config", "0"}},
				&envValue{"10", path{"Config", "1"}},
				&envValue{"true", path{"Config", "2"}},
			},
			map[string]string{
				"CONFIG_0": "FOOO",
				"CONFIG_1": "10",
				"CONFIG_2": "true",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithASliceToStruct",
			&struct {
				Config []basicAppConfig
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Config", "0", "StringValue"}},
				&envValue{"10", path{"Config", "0", "IntValue"}},
				&envValue{"MIMI", path{"Config", "1", "StringValue"}},
				&envValue{"15", path{"Config", "1", "IntValue"}},
			},
			map[string]string{
				"CONFIG_0_STRING_VALUE": "FOOO",
				"CONFIG_0_INT_VALUE":    "10",
				"CONFIG_1_STRING_VALUE": "MIMI",
				"CONFIG_1_INT_VALUE":    "15",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithASliceToASliceToStruct",
			&struct {
				Config [][]basicAppConfig
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Config", "0", "0", "StringValue"}},
				&envValue{"10", path{"Config", "0", "0", "IntValue"}},
				&envValue{"MIMI", path{"Config", "1", "1", "StringValue"}},
				&envValue{"15", path{"Config", "1", "1", "IntValue"}},
			},
			map[string]string{
				"CONFIG_0_0_STRING_VALUE": "FOOO",
				"CONFIG_0_0_INT_VALUE":    "10",
				"CONFIG_1_1_STRING_VALUE": "MIMI",
				"CONFIG_1_1_INT_VALUE":    "15",
			},
			testAnalyzeStructShouldSucceed,
		},
		{
			"WithASliceToAMapToStruct",
			&struct {
				Config []map[string]basicAppConfig
			}{},
			[]*envValue{
				&envValue{"FOOO", path{"Config", "0", "foo", "StringValue"}},
				&envValue{"10", path{"Config", "0", "foo", "IntValue"}},
				&envValue{"MIMI", path{"Config", "1", "bar", "StringValue"}},
				&envValue{"15", path{"Config", "1", "bar", "IntValue"}},
			},
			map[string]string{
				"CONFIG_0_FOO_STRING_VALUE": "FOOO",
				"CONFIG_0_FOO_INT_VALUE":    "10",
				"CONFIG_1_BAR_STRING_VALUE": "MIMI",
				"CONFIG_1_BAR_INT_VALUE":    "15",
			},
			testAnalyzeStructShouldSucceed,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Label, func(t *testing.T) {
			setupEnv(testCase.Env)
			res, err := subject.analyzeStruct(
				reflect.TypeOf(testCase.Source).Elem(),
				path{},
			)
			testCase.Then(t, testCase.Expectation, res, err)
			cleanupEnv(testCase.Env)
		})
	}

}

func TestEnvVarFromPath(t *testing.T) {
	testCases := []struct {
		Label       string
		Prefix      string
		Separator   string
		Path        []string
		Expectation string
	}{
		{"BlankPrefix", "", "_", []string{"Foo"}, "FOO"},
		{"NonBlankPrefix", "YOUPI", "_", []string{"Foo"}, "YOUPI_FOO"},
		{
			"CamelCasedPathMembers",
			"YOUPI",
			"_",
			[]string{"Foo", "IamGroot", "IAmBatman"},
			"YOUPI_FOO_IAM_GROOT_I_AM_BATMAN",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Label, func(t *testing.T) {
			subject := &envConfig{
				testCase.Prefix,
				testCase.Separator,
				map[reflect.Type]parser.Parser{},
				10,
			}

			result := subject.envVarFromPath(testCase.Path)

			if result != testCase.Expectation {
				t.Logf("Expected [%s] got [%s]\n", testCase.Expectation, result)
				t.Fail()
			}
		})
	}

}

func TestNextLevelKeys(t *testing.T) {
	subject := &envConfig{"", "_", map[reflect.Type]parser.Parser{}, 10}
	testCases := []struct {
		Label       string
		Prefix      string
		EnvVars     []string
		Expectation []string
	}{
		{
			"WithPrefix",
			"CONFIG_APP",
			[]string{
				"CONFIG_APP_BATMAN_FOO",
				"CONFIG_APP_ROBIN_FOO",
				"CONFIG_APP_JOCKER_FOO",
			},
			[]string{
				"CONFIG_APP_BATMAN",
				"CONFIG_APP_ROBIN",
				"CONFIG_APP_JOCKER",
			},
		},
		{
			"WithDuplicates",
			"CONFIG_APP",
			[]string{
				"CONFIG_APP_BATMAN_FOO",
				"CONFIG_APP_ROBIN_FOO",
				"CONFIG_APP_JOCKER_FOO",
				"CONFIG_APP_BATMAN_BAR",
			},
			[]string{
				"CONFIG_APP_BATMAN",
				"CONFIG_APP_ROBIN",
				"CONFIG_APP_JOCKER",
				"CONFIG_APP_BATMAN",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Label, func(t *testing.T) {
			res := subject.nextLevelKeys(testCase.Prefix, testCase.EnvVars)
			for i, exp := range testCase.Expectation {
				if exp != res[i] {
					t.Logf("Unexpected value, expected [%s] got [%s]", exp, res[i])
					t.Fail()
				}
			}
		})
	}
}

func TestEnvVarsWithPrefix(t *testing.T) {

	subject := &envConfig{"", "_", map[reflect.Type]parser.Parser{}, 10}

	testCases := []struct {
		Label       string
		Prefix      string
		Env         map[string]string
		Expectation []string
	}{
		{
			"WithPrefix",
			"APP",
			map[string]string{
				"STRING_VALUE":   "FOOO",
				"INT_VALUE":      "10",
				"BOOL_VALUE":     "true",
				"APP_BOOL_VALUE": "true",
				"APP_BAR_VALUE":  "true",
			},
			[]string{"APP_BOOL_VALUE", "APP_BAR_VALUE"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Label, func(t *testing.T) {
			setupEnv(testCase.Env)
			res := subject.envVarsWithPrefix(testCase.Prefix)
			for i, envVar := range testCase.Expectation {
				if envVar != res[i] {
					t.Logf("Invalid env variableName, expected [%s] got [%s]", envVar, res[i])
					t.Fail()
				}
			}
			cleanupEnv(testCase.Env)
		})
	}
}

func TestUnique(t *testing.T) {
	testCases := []struct {
		Label       string
		In          []string
		Expectation []string
	}{
		{
			"WithDuplicates",
			[]string{"FOO", "BAR", "BIZ", "FOO", "BIZ"},
			[]string{"FOO", "BAR", "BIZ"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Label, func(t *testing.T) {
			res := unique(testCase.In)
			for i, val := range testCase.Expectation {
				if res[i] != val {
					t.Logf("Invalid result: expected [%s] got [%s]\n", val, res[i])
					t.Fail()
				}
			}
		})
	}
}

func TestKeyFromEnvVar(t *testing.T) {
	subject := &envConfig{"", "_", map[reflect.Type]parser.Parser{}, 10}
	testCases := []struct {
		Label       string
		Prefix      string
		EnvVar      string
		Expectation string
	}{
		{"WithPrefix", "CONFIG_APP", "CONFIG_APP_BATMAN", "batman"},
		{"WithPrefixAndSuffix", "CONFIG_APP", "CONFIG_APP_BATMAN_FOO", "batman"},
		{"WithoutPrefix", "", "BATMAN", "batman"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Label, func(t *testing.T) {
			if res := subject.keyFromEnvVar(testCase.EnvVar, testCase.Prefix); res != testCase.Expectation {
				t.Logf("Unexpected value, expected [%s] got [%s]", testCase.Expectation, res)
				t.Fail()
			}
		})
	}
}

type nestedConfig struct {
	NestedValue string
}

type testAppConfig struct {
	nestedConfig
	StringValue        string
	OtherStringValue   string
	PtrToValue         *string
	PtrPtrToValue      **string
	StructValue        basicAppConfig
	PtrToStruct        *testAppConfig
	PtrPtrPtrToStruct  ***testAppConfig
	SliceToValue       []string
	SliceToStructValue []basicAppConfig
	SliceToStructPtr   []*testAppConfig
	ArrayToValue       [10]string
	ArrayToPtrValue    [10]*string
	ArrayToPtrStruct   [10]*testAppConfig
	MapToStructPtr     map[int]*testAppConfig
}

func assignShouldSucceed(t *testing.T, expectation, value *testAppConfig, err error) {
	if err != nil {
		t.Logf("Expected no error, got %s", err.Error())
		t.Fail()
	}

	if !reflect.DeepEqual(expectation, value) {
		t.Logf("Incorrect assignation, expected %v got %v", expectation, value)
		t.Fail()
	}
}

func TestAssignValues(t *testing.T) {
	subject := &envConfig{
		"",
		"_",
		parser.LoadBasicTypes(),
		10,
	}

	testCases := []struct {
		Label       string
		Value       *testAppConfig
		Values      []*envValue
		Expectation *testAppConfig
		Then        func(t *testing.T, expectation, value *testAppConfig, err error)
	}{
		{
			"Value",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"StringValue"}},
				&envValue{"BAR", path{"OtherStringValue"}},
			},
			&testAppConfig{StringValue: "FOO", OtherStringValue: "BAR"},
			assignShouldSucceed,
		},
		{
			"NestedValue",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"NestedValue"}},
				&envValue{"BAR", path{"OtherStringValue"}},
			},
			&testAppConfig{
				nestedConfig:     nestedConfig{NestedValue: "FOO"},
				OtherStringValue: "BAR",
			},
			assignShouldSucceed,
		},
		{
			"PtrToValue",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"PtrToValue"}},
			},
			&testAppConfig{
				PtrToValue: func() *string { foo := "FOO"; return &foo }(),
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				if *(expectation.PtrToValue) != *(result.PtrToValue) {
					t.Logf("Incorrect assignation, expected %v got %v", *(expectation.PtrToValue), *(result.PtrToValue))
					t.Fail()
				}
			},
		},
		{
			"PtrPtrToValue",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"PtrPtrToValue"}},
			},
			&testAppConfig{
				PtrPtrToValue: func() **string { foo := "FOO"; ptrFoo := &foo; return &ptrFoo }(),
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				if **(expectation.PtrPtrToValue) != **(result.PtrPtrToValue) {
					t.Logf("Incorrect assignation, expected %v got %v", *(expectation.PtrToValue), *(result.PtrToValue))
					t.Fail()
				}
			},
		},
		{
			"ValueStruct",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"StructValue", "StringValue"}},
			},
			&testAppConfig{
				StructValue: basicAppConfig{
					StringValue: "FOO",
				},
			},
			assignShouldSucceed,
		},
		{
			"PtrToStruct",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"PtrToStruct", "StringValue"}},
			},
			&testAppConfig{
				PtrToStruct: &testAppConfig{
					StringValue: "FOO",
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				if expectation.PtrToStruct.StringValue != expectation.PtrToStruct.StringValue {
					t.Logf("Incorrect assignation, expected %v got %v", expectation.PtrToStruct.StringValue, result.PtrToStruct.StringValue)
					t.Fail()
				}
			},
		},
		{
			"PtrToInitializedStruct",
			&testAppConfig{
				PtrToStruct: &testAppConfig{
					StringValue: "FIZ",
				},
			},
			[]*envValue{
				&envValue{"FOO", path{"PtrToStruct", "StringValue"}},
			},
			&testAppConfig{
				PtrToStruct: &testAppConfig{
					StringValue: "FOO",
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				if expectation.PtrToStruct.StringValue != expectation.PtrToStruct.StringValue {
					t.Logf("Incorrect assignation, expected %v got %v", expectation.PtrToStruct.StringValue, result.PtrToStruct.StringValue)
					t.Fail()
				}
			},
		},
		{
			"PtrPtrPtrToStruct",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"PtrPtrPtrToStruct", "StringValue"}},
			},
			&testAppConfig{
				PtrPtrPtrToStruct: func() ***testAppConfig {
					fooRef := &testAppConfig{StringValue: "FOO"}
					fooRefRef := &fooRef
					return &fooRefRef
				}(),
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				expValue := ***expectation.PtrPtrPtrToStruct
				resValue := ***result.PtrPtrPtrToStruct
				if expValue.StringValue != resValue.StringValue {
					t.Logf("Incorrect assignation, expected %v got %v", expValue.StringValue, resValue.StringValue)
					t.Fail()
				}
			},
		},
		{
			"MixedStructPtrAndValues",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"PtrToStruct", "PtrPtrPtrToStruct", "PtrPtrToValue"}},
			},
			&testAppConfig{
				PtrToStruct: &testAppConfig{
					PtrPtrPtrToStruct: func() ***testAppConfig {
						fooRef := &testAppConfig{
							PtrPtrToValue: func() **string {
								foo := "FOO"
								fooRef := &foo
								return &fooRef
							}(),
						}
						fooRefRef := &fooRef
						return &fooRefRef
					}(),
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				extract := func(config *testAppConfig) string {
					s := *(config.PtrToStruct)
					s2 := ***(s.PtrPtrPtrToStruct)
					return **(s2.PtrPtrToValue)
				}

				expVal := extract(expectation)
				resVal := extract(result)

				if expVal != resVal {
					t.Logf("Invalid assignation, expected [%s] got [%s]", expVal, resVal)
					t.Fail()
				}
			},
		},
		{
			"SliceToValue",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"SliceToValue", "0"}},
				&envValue{"BAR", path{"SliceToValue", "1"}},
				&envValue{"BIZ", path{"SliceToValue", "2"}},
			},
			&testAppConfig{
				SliceToValue: []string{
					"FOO",
					"BAR",
					"BIZ",
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				for i, s := range expectation.SliceToValue {
					if s != result.SliceToValue[i] {
						t.Logf("Invalid assignation, expexted %s got %s", s, result.SliceToValue[i])
						t.Fail()
					}
				}
			},
		},
		{
			"SliceToStructValue",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"SliceToStructValue", "0", "StringValue"}},
				&envValue{"BAR", path{"SliceToStructValue", "1", "StringValue"}},
				&envValue{"BIZ", path{"SliceToStructValue", "2", "StringValue"}},
			},
			&testAppConfig{
				SliceToStructValue: []basicAppConfig{
					basicAppConfig{StringValue: "FOO"},
					basicAppConfig{StringValue: "BAR"},
					basicAppConfig{StringValue: "BIZ"},
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				for i, s := range expectation.SliceToStructValue {
					if s != result.SliceToStructValue[i] {
						t.Logf("Invalid assignation, expexted %s got %s", s.StringValue, result.SliceToStructValue[i].StringValue)
						t.Fail()
					}
				}
			},
		},
		{
			"SliceToStructPtr",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"SliceToStructPtr", "0", "StringValue"}},
				&envValue{"BAR", path{"SliceToStructPtr", "1", "StringValue"}},
				&envValue{"BIZ", path{"SliceToStructPtr", "2", "StringValue"}},
			},
			&testAppConfig{
				SliceToStructPtr: []*testAppConfig{
					&testAppConfig{StringValue: "FOO"},
					&testAppConfig{StringValue: "BAR"},
					&testAppConfig{StringValue: "BIZ"},
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				for i, s := range expectation.SliceToStructPtr {
					if s.StringValue != result.SliceToStructPtr[i].StringValue {
						t.Logf("Invalid assignation, expexted %s got %s", s.StringValue, result.SliceToStructPtr[i].StringValue)
						t.Fail()
					}
				}
			},
		},
		{
			"InitializedSliceToStructPtr",
			&testAppConfig{
				SliceToStructPtr: []*testAppConfig{
					&testAppConfig{StringValue: "FOO"},
					&testAppConfig{StringValue: "BAR"},
					&testAppConfig{StringValue: "BUZ"},
				},
			},
			[]*envValue{
				&envValue{"BIZ", path{"SliceToStructPtr", "2", "StringValue"}},
			},
			&testAppConfig{
				SliceToStructPtr: []*testAppConfig{
					&testAppConfig{StringValue: "FOO"},
					&testAppConfig{StringValue: "BAR"},
					&testAppConfig{StringValue: "BIZ"},
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				for i, s := range expectation.SliceToStructPtr {
					if s.StringValue != result.SliceToStructPtr[i].StringValue {
						t.Logf("Invalid assignation, expexted %s got %s", s.StringValue, result.SliceToStructPtr[i].StringValue)
						t.Fail()
					}
				}
			},
		},
		{
			"SliceToStructPtrWithInvalidIndex",
			&testAppConfig{},
			[]*envValue{
				&envValue{"BIZ", path{"SliceToStructPtr", "NotInt", "StringValue"}},
			},
			&testAppConfig{
				SliceToStructPtr: []*testAppConfig{
					&testAppConfig{StringValue: "FOO"},
					&testAppConfig{StringValue: "BAR"},
					&testAppConfig{StringValue: "BIZ"},
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err == nil {
					t.Log("Expected an error, got nothing!")
					t.Fail()
				}
			},
		},
		{
			"ArrayToValue",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"ArrayToValue", "0"}},
				&envValue{"BAR", path{"ArrayToValue", "1"}},
				&envValue{"BIZ", path{"ArrayToValue", "2"}},
			},
			&testAppConfig{
				ArrayToValue: [10]string{
					"FOO",
					"BAR",
					"BIZ",
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				for i, s := range expectation.ArrayToValue {
					if s != result.ArrayToValue[i] {
						t.Logf("Invalid assignation, expexted %s got %s", s, result.ArrayToValue[i])
						t.Fail()
					}
				}
			},
		},
		{
			"ArrayToValueWithOverflow",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"ArrayToValue", "0"}},
				&envValue{"BAR", path{"ArrayToValue", "1"}},
				&envValue{"BIZ", path{"ArrayToValue", "20"}},
			},
			&testAppConfig{},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err == nil {
					t.Log("Expected an error, got nothing !")
					t.Fail()
				}
			},
		},
		{
			"ArrayToValueWithBadIndex",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"ArrayToValue", "0"}},
				&envValue{"BAR", path{"ArrayToValue", "Foo"}},
				&envValue{"BIZ", path{"ArrayToValue", "2"}},
			},
			&testAppConfig{},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err == nil {
					t.Log("Expected an error, got nothing !")
					t.Fail()
				}
			},
		},
		{
			"ArrayToValueWithNegativeIndex",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"ArrayToValue", "0"}},
				&envValue{"BAR", path{"ArrayToValue", "-1"}},
				&envValue{"BIZ", path{"ArrayToValue", "2"}},
			},
			&testAppConfig{},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err == nil {
					t.Log("Expected an error, got nothing !")
					t.Fail()
				}
			},
		},
		{
			"MapToStructPtr",
			&testAppConfig{},
			[]*envValue{
				&envValue{"FOO", path{"MapToStructPtr", "0", "StringValue"}},
				&envValue{"BAR", path{"MapToStructPtr", "1", "StringValue"}},
				&envValue{"BIZ", path{"MapToStructPtr", "2", "StringValue"}},
			},
			&testAppConfig{
				MapToStructPtr: map[int]*testAppConfig{
					0: &testAppConfig{StringValue: "FOO"},
					1: &testAppConfig{StringValue: "BAR"},
					2: &testAppConfig{StringValue: "BIZ"},
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				for k, v := range expectation.MapToStructPtr {
					if v.StringValue != result.MapToStructPtr[k].StringValue {
						t.Logf("Invalid assignation, expexted %s got %s", v.StringValue, result.MapToStructPtr[k].StringValue)
						t.Fail()
					}
				}
			},
		},
		{
			"InitializedMapToStructPtr",
			&testAppConfig{
				MapToStructPtr: map[int]*testAppConfig{
					0: &testAppConfig{StringValue: "BOO"},
					1: &testAppConfig{StringValue: "FAR"},
					2: &testAppConfig{StringValue: "FIZ"},
				},
			},
			[]*envValue{
				&envValue{"FOO", path{"MapToStructPtr", "0", "StringValue"}},
				&envValue{"BAR", path{"MapToStructPtr", "1", "StringValue"}},
				&envValue{"BIZ", path{"MapToStructPtr", "2", "StringValue"}},
			},
			&testAppConfig{
				MapToStructPtr: map[int]*testAppConfig{
					0: &testAppConfig{StringValue: "FOO"},
					1: &testAppConfig{StringValue: "BAR"},
					2: &testAppConfig{StringValue: "BIZ"},
				},
			},
			func(t *testing.T, expectation, result *testAppConfig, err error) {
				if err != nil {
					t.Logf("Expected no error, got %s", err.Error())
					t.FailNow()
				}

				for k, v := range expectation.MapToStructPtr {
					if v.StringValue != result.MapToStructPtr[k].StringValue {
						t.Logf("Invalid assignation, expexted %s got %s", v.StringValue, result.MapToStructPtr[k].StringValue)
						t.Fail()
					}
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Label, func(t *testing.T) {
			value := reflect.ValueOf(testCase.Value).Elem()
			valueType := value.Type()
			err := subject.assignValues(value, valueType, testCase.Values)
			testCase.Then(t, testCase.Expectation, testCase.Value, err)
		})
	}
}

type embeddedConfig struct {
	EmbeddedValue string
}

type anotherConfigStruct struct {
	embeddedConfig
	StringValue string
	IntValue    int
}

func TestLoadConfig(t *testing.T) {
	subject := &envConfig{"", "_", parser.LoadBasicTypes(), 10}

	testCases := []struct {
		Label       string
		Result      *anotherConfigStruct
		Expectation *anotherConfigStruct
		Env         map[string]string
		Then        func(t *testing.T, expectation, result *anotherConfigStruct, err error)
	}{
		{
			"WithValues",
			&anotherConfigStruct{},
			&anotherConfigStruct{
				embeddedConfig: embeddedConfig{
					EmbeddedValue: "BIZ",
				},
				StringValue: "FOO",
				IntValue:    10,
			},
			map[string]string{
				"EMBEDDED_VALUE": "BIZ",
				"STRING_VALUE":   "FOO",
				"INT_VALUE":      "10",
			},
			func(t *testing.T, expectation, result *anotherConfigStruct, err error) {
				if err != nil {
					t.Log("Wasn't expecting an error, got :", err)
					t.FailNow()
				}

				if *result != *expectation {
					t.Logf("Invalid assignation, expected %v got %v", expectation, result)
					t.Fail()
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Label, func(t *testing.T) {
			setupEnv(testCase.Env)
			err := subject.Load(testCase.Result)
			testCase.Then(t, testCase.Expectation, testCase.Result, err)
			cleanupEnv(testCase.Env)
		})
	}
}
