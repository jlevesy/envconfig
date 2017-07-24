package envconfig

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jlevesy/envconfig/setter"
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
	subject := &envConfig{"", "_", map[reflect.Type]setter.Setter{}, 10}

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
				{"FOOO", path{"StringValue"}},
				{"10", path{"IntValue"}},
				{"true", path{"BoolValue"}},
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
				{"FOOO", path{"StringValue"}},
				{"10", path{"IntValue"}},
				{"true", path{"BoolValue"}},
				{"42.1", path{"FloatValue"}},
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
				{"FOOO", path{"StringValue"}},
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
				{"FOOO", path{"Config", "StringValue"}},
				{"10", path{"Config", "IntValue"}},
				{"true", path{"Config", "BoolValue"}},
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
				{"FOOO", path{"Nested", "Config", "StringValue"}},
				{"10", path{"Nested", "Config", "IntValue"}},
				{"true", path{"Nested", "Config", "BoolValue"}},
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
				{"FOOO", path{"Config", "StringValue"}},
				{"10", path{"Config", "IntValue"}},
				{"true", path{"Config", "BoolValue"}},
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
				{"FOOO", path{"Nested", "Config", "StringValue"}},
				{"10", path{"Nested", "Config", "IntValue"}},
				{"true", path{"Nested", "Config", "BoolValue"}},
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
				{"FOOO", path{"Nested", "Config", "StringValue"}},
				{"10", path{"Nested", "Config", "IntValue"}},
				{"true", path{"Nested", "Config", "BoolValue"}},
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
				{"10", path{"IntValue"}},
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
				{"10", path{"Config", "IntValue"}},
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
				{"10", path{"Config", "IntValue"}},
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
				{"10", path{"Config"}},
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
				{"FOOO", path{"Config", "StringValue"}},
				{"10", path{"Config", "IntValue"}},
				{"true", path{"Config", "BoolValue"}},
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
				{"FOO", path{"Config", "foo"}},
				{"MEH", path{"Config", "bar"}},
				{"BAR", path{"Config", "biz"}},
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
				{"FOO", path{"Config", "foo", "StringValue"}},
				{"MEH", path{"Config", "bar", "StringValue"}},
				{"BAR", path{"Config", "biz", "StringValue"}},
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
				{"FOO", path{"Config", "foo", "StringValue"}},
				{"MEH", path{"Config", "bar", "StringValue"}},
				{"BAR", path{"Config", "biz", "StringValue"}},
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
				{"FOO", path{"Config", "0", "foo", "StringValue"}},
				{"MEH", path{"Config", "1", "bar", "StringValue"}},
				{"BAR", path{"Config", "0", "biz", "StringValue"}},
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
				{"FOOO", path{"Config", "0"}},
				{"10", path{"Config", "1"}},
				{"true", path{"Config", "2"}},
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
				{"FOOO", path{"Config", "0"}},
				{"10", path{"Config", "1"}},
				{"true", path{"Config", "2"}},
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
				{"FOOO", path{"Config", "0"}},
				{"10", path{"Config", "1"}},
				{"true", path{"Config", "2"}},
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
				{"FOOO", path{"Config", "0", "StringValue"}},
				{"10", path{"Config", "0", "IntValue"}},
				{"MIMI", path{"Config", "1", "StringValue"}},
				{"15", path{"Config", "1", "IntValue"}},
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
				{"FOOO", path{"Config", "0", "0", "StringValue"}},
				{"10", path{"Config", "0", "0", "IntValue"}},
				{"MIMI", path{"Config", "1", "1", "StringValue"}},
				{"15", path{"Config", "1", "1", "IntValue"}},
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
				{"FOOO", path{"Config", "0", "foo", "StringValue"}},
				{"10", path{"Config", "0", "foo", "IntValue"}},
				{"MIMI", path{"Config", "1", "bar", "StringValue"}},
				{"15", path{"Config", "1", "bar", "IntValue"}},
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
				map[reflect.Type]setter.Setter{},
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
	subject := &envConfig{"", "_", map[reflect.Type]setter.Setter{}, 10}
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

	subject := &envConfig{"", "_", map[reflect.Type]setter.Setter{}, 10}

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
	subject := &envConfig{"", "_", map[reflect.Type]setter.Setter{}, 10}
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
		setter.LoadBasicTypes(),
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
				{"FOO", path{"StringValue"}},
				{"BAR", path{"OtherStringValue"}},
			},
			&testAppConfig{StringValue: "FOO", OtherStringValue: "BAR"},
			assignShouldSucceed,
		},
		{
			"NestedValue",
			&testAppConfig{},
			[]*envValue{
				{"FOO", path{"NestedValue"}},
				{"BAR", path{"OtherStringValue"}},
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
				{"FOO", path{"PtrToValue"}},
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
				{"FOO", path{"PtrPtrToValue"}},
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
				{"FOO", path{"StructValue", "StringValue"}},
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
				{"FOO", path{"PtrToStruct", "StringValue"}},
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
				{"FOO", path{"PtrToStruct", "StringValue"}},
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
				{"FOO", path{"PtrPtrPtrToStruct", "StringValue"}},
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
				{"FOO", path{"PtrToStruct", "PtrPtrPtrToStruct", "PtrPtrToValue"}},
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
				{"FOO", path{"SliceToValue", "0"}},
				{"BAR", path{"SliceToValue", "1"}},
				{"BIZ", path{"SliceToValue", "2"}},
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
				{"FOO", path{"SliceToStructValue", "0", "StringValue"}},
				{"BAR", path{"SliceToStructValue", "1", "StringValue"}},
				{"BIZ", path{"SliceToStructValue", "2", "StringValue"}},
			},
			&testAppConfig{
				SliceToStructValue: []basicAppConfig{
					{StringValue: "FOO"},
					{StringValue: "BAR"},
					{StringValue: "BIZ"},
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
				{"FOO", path{"SliceToStructPtr", "0", "StringValue"}},
				{"BAR", path{"SliceToStructPtr", "1", "StringValue"}},
				{"BIZ", path{"SliceToStructPtr", "2", "StringValue"}},
			},
			&testAppConfig{
				SliceToStructPtr: []*testAppConfig{
					{StringValue: "FOO"},
					{StringValue: "BAR"},
					{StringValue: "BIZ"},
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
					{StringValue: "FOO"},
					{StringValue: "BAR"},
					{StringValue: "BUZ"},
				},
			},
			[]*envValue{
				{"BIZ", path{"SliceToStructPtr", "2", "StringValue"}},
			},
			&testAppConfig{
				SliceToStructPtr: []*testAppConfig{
					{StringValue: "FOO"},
					{StringValue: "BAR"},
					{StringValue: "BIZ"},
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
				{"BIZ", path{"SliceToStructPtr", "NotInt", "StringValue"}},
			},
			&testAppConfig{
				SliceToStructPtr: []*testAppConfig{
					{StringValue: "FOO"},
					{StringValue: "BAR"},
					{StringValue: "BIZ"},
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
				{"FOO", path{"ArrayToValue", "0"}},
				{"BAR", path{"ArrayToValue", "1"}},
				{"BIZ", path{"ArrayToValue", "2"}},
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
				{"FOO", path{"ArrayToValue", "0"}},
				{"BAR", path{"ArrayToValue", "1"}},
				{"BIZ", path{"ArrayToValue", "20"}},
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
				{"FOO", path{"ArrayToValue", "0"}},
				{"BAR", path{"ArrayToValue", "Foo"}},
				{"BIZ", path{"ArrayToValue", "2"}},
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
				{"FOO", path{"ArrayToValue", "0"}},
				{"BAR", path{"ArrayToValue", "-1"}},
				{"BIZ", path{"ArrayToValue", "2"}},
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
				{"FOO", path{"MapToStructPtr", "0", "StringValue"}},
				{"BAR", path{"MapToStructPtr", "1", "StringValue"}},
				{"BIZ", path{"MapToStructPtr", "2", "StringValue"}},
			},
			&testAppConfig{
				MapToStructPtr: map[int]*testAppConfig{
					0: {StringValue: "FOO"},
					1: {StringValue: "BAR"},
					2: {StringValue: "BIZ"},
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
					0: {StringValue: "BOO"},
					1: {StringValue: "FAR"},
					2: {StringValue: "FIZ"},
				},
			},
			[]*envValue{
				{"FOO", path{"MapToStructPtr", "0", "StringValue"}},
				{"BAR", path{"MapToStructPtr", "1", "StringValue"}},
				{"BIZ", path{"MapToStructPtr", "2", "StringValue"}},
			},
			&testAppConfig{
				MapToStructPtr: map[int]*testAppConfig{
					0: {StringValue: "FOO"},
					1: {StringValue: "BAR"},
					2: {StringValue: "BIZ"},
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
	subject := &envConfig{"", "_", setter.LoadBasicTypes(), 10}

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

type yetAnotherConfigStruct struct {
	Date        time.Time            `envconfig:"noexpand"`
	PtrDate     *time.Time           `envconfig:"noexpand"`
	Items       []string             `envconfig:"noexpand"`
	Configs     []*grootConfig       `envconfig:"noexpand"`
	OtherConfig *anotherConfigStruct `envconfig:"noexpand"`
}

type grootConfig struct {
	IamGroot string
}

func sliceOfGrootSetter(strValue string, value reflect.Value) error {
	split := strings.Split(strValue, ",")
	res := make([]*grootConfig, len(split))

	for i, s := range split {
		res[i] = &grootConfig{s}
	}

	value.Set(reflect.ValueOf(res))
	return nil
}

func sliceOfStringSetter(strValue string, value reflect.Value) error {
	value.Set(reflect.ValueOf(strings.Split(strValue, ",")))
	return nil
}

func TestLoadConfigNoExpand(t *testing.T) {
	setters := setter.LoadBasicTypes()

	setters[reflect.TypeOf([]string{})] = setter.SetterFunc(sliceOfStringSetter)
	setters[reflect.TypeOf([]*grootConfig{})] = setter.SetterFunc(sliceOfGrootSetter)

	subject := &envConfig{"", "_", setters, 10}

	testCases := []struct {
		Label       string
		Result      *yetAnotherConfigStruct
		Expectation *yetAnotherConfigStruct
		Env         map[string]string
		Then        func(t *testing.T, expectation, result *yetAnotherConfigStruct, err error)
	}{
		{
			"WithValueStruct",
			&yetAnotherConfigStruct{},
			&yetAnotherConfigStruct{
				Date: func() time.Time {
					res, _ := time.Parse(time.RFC3339, "2009-08-25T00:00:00Z")
					return res
				}(),
			},
			map[string]string{
				"DATE": "2009-08-25T00:00:00Z",
			},
			func(t *testing.T, expectation, result *yetAnotherConfigStruct, err error) {
				if err != nil {
					t.Log("Wasn't expecting an error, got :", err)
					t.FailNow()
				}

				if result.Date != expectation.Date {
					t.Logf("Invalid assignation, expected %v got %v", expectation, result)
					t.Fail()
				}
			},
		},
		{
			"WithPtrStruct",
			&yetAnotherConfigStruct{},
			&yetAnotherConfigStruct{
				PtrDate: func() *time.Time {
					res, _ := time.Parse(time.RFC3339, "2009-08-25T00:00:00Z")
					return &res
				}(),
			},
			map[string]string{
				"PTR_DATE": "2009-08-25T00:00:00Z",
			},
			func(t *testing.T, expectation, result *yetAnotherConfigStruct, err error) {
				if err != nil {
					t.Log("Wasn't expecting an error, got :", err)
					t.FailNow()
				}

				if *(result.PtrDate) != *(expectation.PtrDate) {
					t.Logf("Invalid assignation, expected %v got %v", expectation, result)
					t.Fail()
				}
			},
		},
		{
			"WithCollection",
			&yetAnotherConfigStruct{},
			&yetAnotherConfigStruct{
				Items: []string{
					"foo",
					"bar",
					"buz",
				},
			},
			map[string]string{
				"ITEMS": "foo,bar,buz",
			},
			func(t *testing.T, expectation, result *yetAnotherConfigStruct, err error) {
				if err != nil {
					t.Log("Wasn't expecting an error, got :", err)
					t.FailNow()
				}

				if len(result.Items) != len(expectation.Items) {
					t.Logf("Assignation failed, expected length of %d got %d", len(expectation.Items), len(result.Items))
					t.FailNow()
				}

				if result.Items[0] != expectation.Items[0] ||
					result.Items[1] != expectation.Items[1] ||
					result.Items[2] != expectation.Items[2] {
					t.Logf("Invalid assignation, expected %v got %v", expectation, result)
					t.FailNow()
				}
			},
		},
		{
			"WithCollectionOfStructs",
			&yetAnotherConfigStruct{},
			&yetAnotherConfigStruct{
				Configs: []*grootConfig{
					{"I"},
					{"AM"},
					{"GROOT"},
				},
			},
			map[string]string{
				"CONFIGS": "I,AM,GROOT",
			},
			func(t *testing.T, expectation, result *yetAnotherConfigStruct, err error) {
				if err != nil {
					t.Log("Wasn't expecting an error, got :", err)
					t.FailNow()
				}

				if len(result.Configs) != len(expectation.Configs) {
					t.Logf("Assignation failed, expected length of %d got %d", len(expectation.Configs), len(result.Configs))
					t.FailNow()
				}

				if *(result.Configs[0]) != *(expectation.Configs[0]) ||
					*(result.Configs[1]) != *(expectation.Configs[1]) ||
					*(result.Configs[2]) != *(expectation.Configs[2]) {
					t.Logf("Invalid assignation, expected %v got %v", expectation, result)
					t.FailNow()
				}
			},
		},
		{
			"WithUnknownSetter",
			&yetAnotherConfigStruct{},
			nil,
			map[string]string{
				"OTHER_CONFIG": "I,AM,GROOT",
			},
			func(t *testing.T, expectation, result *yetAnotherConfigStruct, err error) {
				if err == nil {
					t.Log("Expecting an error, got nothing :(")
					t.FailNow()
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
