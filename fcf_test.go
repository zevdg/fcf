package fcf

import (
	"math"
	"strconv"
	"testing"
	"time"
)

func TestString(t *testing.T) {
	testVal := "foo"
	fcfVal := &struct {
		Fields map[string]interface{}
	}{make(map[string]interface{})}
	fcfVal.Fields["Field"] = map[string]interface{}{
		"stringValue": testVal,
	}

	userVal := &struct {
		Field string
	}{}
	err := unmarshalMap(fcfVal, userVal)
	if err != nil {
		t.Error(err)
	}
	if userVal.Field != testVal {
		t.Errorf("expected %q, got %q", testVal, userVal.Field)
	}
}

func TestMissingFields(t *testing.T) {
	fcfVal := &struct {
		Fields map[string]interface{}
	}{make(map[string]interface{})}

	userVal := &struct {
		Field string
		Ptr   *bool
	}{}
	err := unmarshalMap(fcfVal, userVal)
	if err != nil {
		t.Error(err)
	}
	if userVal.Field != "" {
		t.Errorf("expected %q, got %q", "", userVal.Field)
	}

	if userVal.Ptr != nil {
		t.Errorf("expected %v, got %v", nil, *userVal.Ptr)
	}
}

func TestReference(t *testing.T) {
	testVal := "/col1/doc1/col2/doc2"
	fullVal := "projects/project-name/databases/(default)/documents" + testVal
	fcfVal := &struct {
		Fields map[string]interface{}
	}{make(map[string]interface{})}
	fcfVal.Fields["Field"] = map[string]interface{}{
		"referenceValue": fullVal,
	}

	userVal := &struct {
		Field string
	}{}
	err := unmarshalMap(fcfVal, userVal)
	if err != nil {
		t.Error(err)
	}
	if userVal.Field != testVal {
		t.Errorf("expected %q, got %q", testVal, userVal.Field)
	}
}

func TestBool(t *testing.T) {
	testVal := true
	fcfVal := &struct {
		Fields map[string]interface{}
	}{make(map[string]interface{})}
	fcfVal.Fields["Field"] = map[string]interface{}{
		"booleanValue": testVal,
	}

	userVal := &struct {
		Field bool
	}{}
	err := unmarshalMap(fcfVal, userVal)
	if err != nil {
		t.Error(err)
	}
	if userVal.Field != testVal {
		t.Errorf("expected %v, got %v", testVal, userVal.Field)
	}
}

func TestBoolPtr(t *testing.T) {
	testVal := true
	fcfVal := &struct {
		Fields map[string]interface{}
	}{make(map[string]interface{})}
	fcfVal.Fields["Field"] = map[string]interface{}{
		"booleanValue": testVal,
	}

	userVal := &struct {
		Field *bool
	}{}
	err := unmarshalMap(fcfVal, userVal)
	if err != nil {
		t.Error(err)
	}
	if *userVal.Field != testVal {
		t.Errorf("expected %v, got %v", testVal, userVal.Field)
	}
}

func TestInteger(t *testing.T) {
	testVal := 42
	testVal8 := 8
	testVal16 := 16
	testVal32 := 32
	testVal64 := 64
	testValPtr := 1337
	fcfVal := &struct {
		Fields map[string]interface{}
	}{make(map[string]interface{})}

	fcfVal.Fields["Int"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal),
	}
	fcfVal.Fields["Int8"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal8),
	}
	fcfVal.Fields["Int16"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal16),
	}
	fcfVal.Fields["Int32"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal32),
	}
	fcfVal.Fields["Int64"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal64),
	}
	fcfVal.Fields["Uint"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal),
	}
	fcfVal.Fields["Uint8"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal8),
	}
	fcfVal.Fields["Uint16"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal16),
	}
	fcfVal.Fields["Uint32"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal32),
	}
	fcfVal.Fields["Uint64"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testVal64),
	}
	fcfVal.Fields["Uintptr"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testValPtr),
	}
	userVal := &struct {
		Int     int
		Int8    int8
		Int16   int16
		Int32   int32
		Int64   int64
		Uint    uint
		Uint8   uint8
		Uint16  uint16
		Uint32  uint32
		Uint64  uint64
		Uintptr uintptr
	}{}
	err := unmarshalMap(fcfVal, userVal)
	if err != nil {
		t.Error(err)
	}
	if userVal.Int != testVal {
		t.Errorf("Int field failed: expected %v, got %v", testVal, userVal.Int)
	}
	if int(userVal.Int8) != testVal8 {
		t.Errorf("Int8 field failed: expected %v, got %v", testVal8, userVal.Int8)
	}
	if int(userVal.Int16) != testVal16 {
		t.Errorf("Int16 field failed: expected %v, got %v", testVal16, userVal.Int16)
	}
	if int(userVal.Int32) != testVal32 {
		t.Errorf("Int32 field failed: expected %v, got %v", testVal32, userVal.Int32)
	}
	if int(userVal.Int64) != testVal64 {
		t.Errorf("Int64 field failed: expected %v, got %v", testVal64, userVal.Int64)
	}
	if int(userVal.Uint) != testVal {
		t.Errorf("Uint field failed: expected %v, got %v", testVal, userVal.Uint)
	}
	if int(userVal.Uint8) != testVal8 {
		t.Errorf("Uint8 field failed: expected %v, got %v", testVal8, userVal.Uint8)
	}
	if int(userVal.Uint16) != testVal16 {
		t.Errorf("Uint16 field failed: expected %v, got %v", testVal16, userVal.Uint16)
	}
	if int(userVal.Uint32) != testVal32 {
		t.Errorf("Uint32 field failed: expected %v, got %v", testVal32, userVal.Uint32)
	}
	if int(userVal.Uint64) != testVal64 {
		t.Errorf("Uint64 field failed: expected %v, got %v", testVal64, userVal.Uint64)
	}
	if int(userVal.Uintptr) != testValPtr {
		t.Errorf("Uint64 field failed: expected %v, got %v", testValPtr, userVal.Uint64)
	}
}

func TestDecimal(t *testing.T) {
	testFloat32 := 3.2
	testFloat64 := 3.4
	testInt32 := 32
	testInt64 := 64

	fcfVal := &struct {
		Fields map[string]interface{}
	}{make(map[string]interface{})}
	fcfVal.Fields["Float32"] = map[string]interface{}{
		"doubleValue": float64(testFloat32),
	}
	fcfVal.Fields["Float64"] = map[string]interface{}{
		"doubleValue": testFloat64,
	}
	fcfVal.Fields["Int32"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testInt32),
	}
	fcfVal.Fields["Int64"] = map[string]interface{}{
		"integerValue": strconv.Itoa(testInt64),
	}
	userVal := &struct {
		Float32 float32
		Float64 float64
		Int32   float32
		Int64   float64
	}{}
	err := unmarshalMap(fcfVal, userVal)
	if err != nil {
		t.Error(err)
	}
	if math.Round(float64(userVal.Float32)-testFloat32) != 0 {
		t.Errorf("expected %f, got %f", float64(testFloat32), userVal.Float32)
	}
	if math.Round(userVal.Float64-testFloat64) != 0 {
		t.Errorf("expected %v, got %v", testFloat64, userVal.Float64)
	}
	if math.Round(float64(userVal.Int32)-float64(testInt32)) != 0 {
		t.Errorf("expected %v, got %v - %v", float32(testInt32), userVal.Int32, userVal.Int32 == float32(testInt32))
	}
	if math.Round(userVal.Int64-float64(testInt64)) != 0 {
		t.Errorf("expected %v, got %v", testInt64, userVal.Int64)
	}
}

func TestTimestamp(t *testing.T) {
	testVal := time.Date(2019, time.February, 3, 1, 7, 5, 565000000, time.UTC)
	fcfVal := &struct {
		Fields map[string]interface{}
	}{make(map[string]interface{})}
	fcfVal.Fields["Field"] = map[string]interface{}{
		"timestampValue": testVal.Format(time.RFC3339Nano),
	}

	userVal := &struct {
		Field time.Time
	}{}
	err := unmarshalMap(fcfVal, userVal)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%v, %v, %v", testVal, testVal.Format(time.RFC3339Nano), userVal.Field)
	if userVal.Field != testVal {
		t.Errorf("expected %q, got %q", testVal, userVal.Field)
	}
}

// func TestBytes(t *testing.T) {
// 	testVal := []byte("foo")
// 	fcfVal := &struct {
// 		Fields map[string]interface{}
// 	}{make(map[string]interface{})}
// 	fcfVal.Fields["Field"] = map[string]interface{}{
// 		"bytesValue": testVal,
// 	}

// 	userVal := &struct {
// 		Field []byte
// 	}{}
// 	err := unmarshalMap(fcfVal, userVal)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if userVal.Field != testVal {
// 		t.Errorf("expected %q, got %q", testVal, userVal.Field)
// 	}
// }
