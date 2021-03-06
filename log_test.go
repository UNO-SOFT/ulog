// Copyright 2019 The Antilog Authors.
//
// SPDX-License-Identifier: MIT

package ulog_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/UNO-SOFT/ulog"
	"github.com/stretchr/testify/require"
)

func parseLogLine(b []byte) (v map[string]interface{}) {
	err := json.Unmarshal(b, &v)
	if err != nil {
		panic(fmt.Errorf("%q: %w", string(b), err))
	}
	return
}

func parseTime(i interface{}) (t time.Time) {
	s := i.(string)
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return
}

func TestHasTimestampAndMessage(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	logger.Write("this is a test")

	logLine := parseLogLine(buffer.Bytes())

	require.Len(t, logLine, 2)
	require.WithinDuration(t, time.Now(), parseTime(logLine[ulog.DefaultTimestampKey]), 1*time.Second)
	require.Equal(t, "this is a test", logLine[ulog.DefaultMessageKey])
}

func TestHandlesBasicTypes(t *testing.T) {
	for test, value := range map[string]interface{}{
		"bool":   true,
		"int":    1234,
		"float":  123.456,
		"string": "wibble",
		"array":  []interface{}{"woo", "yay", "houpla"},
		"map":    map[string]interface{}{"woo": "yay", "houpla": "panowie"},
	} {
		t.Run(test, func(t *testing.T) {
			var buffer bytes.Buffer
			logger := ulog.WithWriter(&buffer)

			logger.Write("this is a test", test, value)
			logLine := parseLogLine(buffer.Bytes())

			t.Log(buffer.String())
			require.EqualValues(t, value, logLine[test])
		})
	}
}

func TestOmitsUnknownTypes(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	logger.Write("this is a test", "ulog", logger)
	logLine := parseLogLine(buffer.Bytes())

	expected := map[string]interface{}{
		"Writer": map[string]interface{}{},
	}
	require.EqualValues(t, expected, logLine["ulog"])
}

func TestIncludesContextFields(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer).With("test", "hello")

	logger.Write("this is a test")
	logLine := parseLogLine(buffer.Bytes())

	require.EqualValues(t, "hello", logLine["test"])
}

func TestAppendsLoggedFieldsToContextFields(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer).With("test", "hello")

	logger.Write("this is a test", "tomato", "banana")
	logLine := parseLogLine(buffer.Bytes())

	require.EqualValues(t, "banana", logLine["tomato"])
}

func TestPicksLastDuplicateValue(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	logger.Write("this is a test", "tomato", 1, "potato", 2, "pineapple", 3, "potato", 4)
	require.NotContains(t, buffer.String(), `"potato": 2`)
	require.Contains(t, buffer.String(), `"potato": 4`)

	logLine := parseLogLine(buffer.Bytes())
	require.EqualValues(t, 4, logLine["potato"])
}

func TestOverridesContextValue(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer).With("potato", 2)

	logger.Write("this is a test", "tomato", 1, "pineapple", 3, "potato", 4)
	require.NotContains(t, buffer.String(), `"potato": 2`)
	require.Contains(t, buffer.String(), `"potato": 4`)

	logLine := parseLogLine(buffer.Bytes())
	require.EqualValues(t, 4, logLine["potato"])
}

func TestReplacesContextValue(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer).With("potato", 2)

	logger = logger.With("potato", 4)

	logger.Write("this is a test", "tomato", 1, "pineapple", 3)
	require.NotContains(t, buffer.String(), `"potato": 2`)
	require.Contains(t, buffer.String(), `"potato": 4`)

	logLine := parseLogLine(buffer.Bytes())
	require.EqualValues(t, 4, logLine["potato"])
}

func TestLogsErrors(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	logger.Write("this is a test", "error", errors.New("an error occurred"))
	logLine := parseLogLine(buffer.Bytes())

	t.Logf("line: %q", buffer.String())
	require.EqualValues(t, "an error occurred", logLine["error"])
}

func TestLogsNilErrors(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	var err error
	logger.Write("this is a test", "error", err)
	logLine := parseLogLine(buffer.Bytes())

	require.EqualValues(t, nil, logLine["error"])
}

type OuterStruct struct {
	Name         string
	AnotherField string
	Inner        InnerStruct
}

type InnerStruct struct {
	Tag          string
	ArrayOfStuff []LeafStruct
}

type LeafStruct struct {
	Key   string
	Value string
}

func TestHandlesNestedStructs(t *testing.T) {
	inputStructure := OuterStruct{
		Name: "Test struct",
		Inner: InnerStruct{
			Tag: "something",
			ArrayOfStuff: []LeafStruct{
				{"a key", "a value"},
				{"another", "with another value"},
				{"one more", "for luck"},
			},
		},
		AnotherField: "what is this?",
	}

	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	logger.Write("this is a test", "struct test", inputStructure)

	var actual struct {
		OuterStruct OuterStruct `json:"struct test"`
	}

	err := json.Unmarshal(buffer.Bytes(), &actual)
	require.NoError(t, err)

	require.EqualValues(t, inputStructure, actual.OuterStruct)
}

type TaggedStruct struct {
	Name        string `json:"name"`
	NotIncluded string `json:"-"`
	Age         int    `json:"age"`
}

func TestHandlesStructTags(t *testing.T) {
	inputStructure := TaggedStruct{
		Name:        "Jim",
		Age:         42,
		NotIncluded: "blah",
	}

	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	logger.Write("this is a test", "struct test", inputStructure)

	expected := map[string]interface{}{
		"name": "Jim",
		"age":  float64(42),
	}
	actual := map[string]interface{}{}
	err := json.Unmarshal(buffer.Bytes(), &actual)
	require.NoError(t, err)

	require.EqualValues(t, expected, actual["struct test"])
}

func TestHandlesDeeplyNestedTypes(t *testing.T) {
	inputStructure := map[string]interface{}{
		"array_with_various_types": []interface{}{
			"string",
			123.456,
			[]interface{}{
				"another",
				"array",
				"inside",
			},
			map[string]interface{}{
				"a map": "nested in the array",
			},
		},
		"map_with_various_types": map[string]interface{}{
			"string": "a string",
			"number": 1234.0,
			"bool":   false,
			"an array!": []interface{}{
				"with",
				"mixed",
				false,
				"types",
				map[string]interface{}{
					"including": "a map",
				},
			},
			"another map": map[string]interface{}{
				"with its own values": "like this",
			},
		},
	}

	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	logger.Write("this is a test", "a deep structure", inputStructure)
	logLine := parseLogLine(buffer.Bytes())

	require.EqualValues(t, inputStructure, logLine["a deep structure"])
}

func TestAlteringMapsDoesNotChangeLog(t *testing.T) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	values := map[string]string{
		"woo": "yay",
	}
	logger = logger.With("values", values)

	values["woo"] = "no"
	values["yay"] = "yes"

	logger.Write("this is a test")
	logLine := parseLogLine(buffer.Bytes())

	require.EqualValues(t, map[string]interface{}{"woo": "yay"}, logLine["values"])
}

func BenchmarkLogWithNoFields(b *testing.B) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)

	for n := 0; n < b.N; n++ {
		logger.Write("a message")
	}
}

func BenchmarkLogWithComplexFields(b *testing.B) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer)
	inputStructure := map[string]interface{}{
		"array_with_various_types": []interface{}{
			"string",
			123.456,
			[]interface{}{
				"another",
				"array",
				"inside",
			},
			map[string]interface{}{
				"a map": "nested in the array",
			},
		},
		"map_with_various_types": map[string]interface{}{
			"string": "a string",
			"number": 1234.0,
			"bool":   false,
			"an array!": []interface{}{
				"with",
				"mixed",
				false,
				"types",
				map[string]interface{}{
					"including": "a map",
				},
			},
			"another map": map[string]interface{}{
				"with its own values": "like this",
			},
		},
		"a struct of all things": struct {
			Name string
			Age  int
		}{"Mr Blobby", 48},
	}

	for n := 0; n < b.N; n++ {
		logger.Write("a message", "complex field", inputStructure)
	}
}

func BenchmarkLogWithComplexFieldsInContext(b *testing.B) {
	var buffer bytes.Buffer
	logger := ulog.WithWriter(&buffer).With("complex field", map[string]interface{}{
		"array_with_various_types": []interface{}{
			"string",
			123.456,
			[]interface{}{
				"another",
				"array",
				"inside",
			},
			map[string]interface{}{
				"a map": "nested in the array",
			},
		},
		"map_with_various_types": map[string]interface{}{
			"string": "a string",
			"number": 1234.0,
			"bool":   false,
			"an array!": []interface{}{
				"with",
				"mixed",
				false,
				"types",
				map[string]interface{}{
					"including": "a map",
				},
			},
			"another map": map[string]interface{}{
				"with its own values": "like this",
			},
		},
		"a struct of all things": struct {
			Name string
			Age  int
		}{"Mr Blobby", 48},
	})

	for n := 0; n < b.N; n++ {
		logger.Write("a message", "simple field", "test")
	}
}

/*
func TestKitLog(t *testing.T) {
	var buf bytes.Buffer
	logger := log.With(ulog.WithWriter(&buf), "a", "b")
	logger.Log("msg", "message")
	t.Log(buf.String())
}
*/

func TestError(t *testing.T) {
	var buf bytes.Buffer
	logger := ulog.WithWriter(&buf)
	f := func() error {
		return io.EOF
	}
	logger.Write("msg", "error", fmt.Errorf("deep: %w", fmt.Errorf("io: %w", ulog.WrapError(f()))))
	t.Log(buf.String())
}

func TestMarshalError(t *testing.T) {
	var buf bytes.Buffer
	logger := ulog.WithWriter(&buf)
	logger.Write("channel", "chan", make(chan int))
	t.Log(buf.String())
}

func TestTestLogger(t *testing.T) {
	logger := ulog.NewTestLogger(t)
	logger.Log("msg", "test")
}
