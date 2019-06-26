package fcf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func TestString(t *testing.T) {
	testVal := "foo"
	fcfVal := Value{
		Fields: map[string]interface{}{
			"Field": map[string]interface{}{"stringValue": testVal},
		},
	}

	userVal := &struct {
		Field string
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if userVal.Field != testVal {
		t.Errorf("expected %q, got %q", testVal, userVal.Field)
	}
}

func TestMissingFields(t *testing.T) {
	fcfVal := Value{Fields: map[string]interface{}{
		"otherField": map[string]interface{}{"stringValue": "foo"},
	}}

	userVal := &struct {
		Field string
		Ptr   *bool
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if userVal.Field != "" {
		t.Errorf("expected %q, got %q", "", userVal.Field)
	}

	if userVal.Ptr != nil {
		t.Errorf("expected %v, got %v", nil, *userVal.Ptr)
	}
}

func TestTag(t *testing.T) {
	testVal := "foo"
	fcfVal := Value{
		Fields: map[string]interface{}{
			"otherName": map[string]interface{}{"stringValue": testVal},
		},
	}

	userVal := &struct {
		Field string `fcf:"otherName"`
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if userVal.Field != testVal {
		t.Errorf("expected %q, got %q", testVal, userVal.Field)
	}
}

func TestReference(t *testing.T) {
	testVal := "/col1/doc1/col2/doc2"
	fullVal := "projects/project-name/databases/(default)/documents" + testVal
	fcfVal := Value{
		Fields: map[string]interface{}{
			"Field": map[string]interface{}{"referenceValue": fullVal},
		},
	}

	userVal := &struct {
		Field string
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if userVal.Field != testVal {
		t.Errorf("expected %q, got %q", testVal, userVal.Field)
	}
}

func TestBool(t *testing.T) {
	testVal := true
	fcfVal := Value{
		Fields: map[string]interface{}{
			"Field": map[string]interface{}{"booleanValue": testVal},
		},
	}

	userVal := &struct {
		Field bool
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if userVal.Field != testVal {
		t.Errorf("expected %v, got %v", testVal, userVal.Field)
	}
}

func TestBoolPtr(t *testing.T) {
	testVal := true
	fcfVal := Value{
		Fields: map[string]interface{}{
			"Field": map[string]interface{}{"booleanValue": testVal},
		},
	}

	userVal := &struct {
		Field *bool
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if *userVal.Field != testVal {
		t.Errorf("expected %v, got %v", testVal, userVal.Field)
	}
}

func TestDynamic(t *testing.T) {
	testString := "foo"
	testInt := 3
	fcfVal := Value{
		Fields: map[string]interface{}{
			"String": map[string]interface{}{"stringValue": testString},
			"Int":    map[string]interface{}{"integerValue": strconv.Itoa(testInt)},
		},
	}

	userVal := &struct {
		String interface{}
		Int    interface{}
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}

	if userVal.String.(string) != testString {
		t.Errorf("expected %q, got %q", testString, userVal.String)
	}
	if userVal.Int.(int) != testInt {
		t.Errorf("expected %q, got %q", testInt, userVal.Int)
	}
}

func TestInteger(t *testing.T) {
	testVal := 42
	testVal8 := 8
	testVal16 := 16
	testVal32 := 32
	testVal64 := 64
	testValPtr := 1337
	fcfVal := Value{
		Fields: map[string]interface{}{
			"Int":     map[string]interface{}{"integerValue": strconv.Itoa(testVal)},
			"Int8":    map[string]interface{}{"integerValue": strconv.Itoa(testVal8)},
			"Int16":   map[string]interface{}{"integerValue": strconv.Itoa(testVal16)},
			"Int32":   map[string]interface{}{"integerValue": strconv.Itoa(testVal32)},
			"Int64":   map[string]interface{}{"integerValue": strconv.Itoa(testVal64)},
			"Uint":    map[string]interface{}{"integerValue": strconv.Itoa(testVal)},
			"Uint8":   map[string]interface{}{"integerValue": strconv.Itoa(testVal8)},
			"Uint16":  map[string]interface{}{"integerValue": strconv.Itoa(testVal16)},
			"Uint32":  map[string]interface{}{"integerValue": strconv.Itoa(testVal32)},
			"Uint64":  map[string]interface{}{"integerValue": strconv.Itoa(testVal64)},
			"Uintptr": map[string]interface{}{"integerValue": strconv.Itoa(testValPtr)},
		},
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
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
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

	fcfVal := Value{
		Fields: map[string]interface{}{
			"Float32": map[string]interface{}{"doubleValue": float64(testFloat32)},
			"Float64": map[string]interface{}{"doubleValue": testFloat64},
			"Int32":   map[string]interface{}{"integerValue": strconv.Itoa(testInt32)},
			"Int64":   map[string]interface{}{"integerValue": strconv.Itoa(testInt64)},
		},
	}
	userVal := &struct {
		Float32 float32
		Float64 float64
		Int32   float32
		Int64   float64
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
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
	fcfVal := Value{
		Fields: map[string]interface{}{
			"Field": map[string]interface{}{"timestampValue": testVal.Format(time.RFC3339Nano)},
		},
	}

	userVal := &struct {
		Field time.Time
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if userVal.Field != testVal {
		t.Errorf("expected %q, got %q", testVal, userVal.Field)
	}
}

func TestGeoPoint(t *testing.T) {
	testLat, testLong := 26.357896, 127.783809
	fcfVal := Value{
		Fields: map[string]interface{}{
			"Field": map[string]interface{}{
				"geoPointValue": map[string]interface{}{
					"latitude":  testLat,
					"longitude": testLong,
				},
			},
		},
	}

	userVal := &struct {
		Field  GeoPoint
		Custom struct {
			Lat  float32 `fcf:"latitude"`
			Long float32 `fcf:"longitude"`
		} `fcf:"Field"`
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if userVal.Field.Latitude != testLat {
		t.Errorf("expected latitude %v, got %v", testLat, userVal.Field.Latitude)
	}
	if userVal.Field.Longitude != testLong {
		t.Errorf("expected longitude %v, got %v", testLong, userVal.Field.Longitude)
	}
	if userVal.Custom.Lat != float32(testLat) {
		t.Errorf("expected latitude %v, got %v", testLat, userVal.Field.Latitude)
	}
	if userVal.Custom.Long != float32(testLong) {
		t.Errorf("expected longitude %v, got %v", testLong, userVal.Field.Longitude)
	}
}

func TestBytes(t *testing.T) {
	testVal := []byte("foobar")
	fcfVal := Value{
		Fields: map[string]interface{}{
			"Field": map[string]interface{}{"bytesValue": base64.StdEncoding.EncodeToString(testVal)},
		},
	}

	userVal := &struct {
		Field []byte
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(userVal.Field, testVal) != 0 {
		t.Errorf("expected %q, got %q", testVal, userVal.Field)
	}
}

func TestNull(t *testing.T) {
	fcfVal := Value{
		Fields: map[string]interface{}{
			"Field": map[string]interface{}{"nullValue": nil},
			"Ptr":   map[string]interface{}{"nullValue": nil},
		},
	}

	s := "bar"
	userVal := &struct {
		Field string
		Ptr   *string
	}{
		Field: "foo",
		Ptr:   &s,
	}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}
	if userVal.Field != "" {
		t.Errorf("expected empty string, got %q", userVal.Field)
	}
	if userVal.Ptr != nil {
		t.Errorf("expected nil, got %q (%v)", *userVal.Ptr, userVal.Ptr)
	}
}

func TestMapAsStruct(t *testing.T) {
	key0, key1, key2 := "Elem0", "Elem1", "Elem2"
	val0, val1, val2 := "foo", "bar", "baz"

	fcfVal := Value{
		Fields: map[string]interface{}{
			"Struct": map[string]interface{}{
				"mapValue": map[string]interface{}{
					"fields": map[string]interface{}{
						key0: map[string]interface{}{"stringValue": val0},
						key1: map[string]interface{}{"stringValue": val1},
						key2: map[string]interface{}{"stringValue": val2},
					},
				},
			},
		},
	}

	userVal := &struct {
		Struct struct {
			Elem0 string
			Elem1 string
			Elem2 string
		}
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}

	if val0 != userVal.Struct.Elem0 {
		t.Errorf("%s: expected %q, got %q", key0, val0, userVal.Struct.Elem0)
	}
	if val1 != userVal.Struct.Elem1 {
		t.Errorf("%s: expected %q, got %q", key1, val1, userVal.Struct.Elem1)
	}
	if val2 != userVal.Struct.Elem2 {
		t.Errorf("%s: expected %q, got %q", key2, val2, userVal.Struct.Elem2)
	}
}

func TestMapStatic(t *testing.T) {
	key0, key1, key2 := "Elem0", "Elem1", "Elem2"
	val0, val1, val2 := "foo", "bar", "baz"

	fcfVal := Value{
		Fields: map[string]interface{}{
			"Map": map[string]interface{}{
				"mapValue": map[string]interface{}{
					"fields": map[string]interface{}{
						key0: map[string]interface{}{"stringValue": val0},
						key1: map[string]interface{}{"stringValue": val1},
						key2: map[string]interface{}{"stringValue": val2},
					},
				},
			},
		},
	}

	userVal := &struct {
		Map map[string]string
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}

	if 3 != len(userVal.Map) {
		t.Fatalf("Map length mismatch: expected %v, got %v", 3, userVal.Map)
	}

	if val0 != userVal.Map[key0] {
		t.Errorf("%s: expected %q, got %q", key0, val0, userVal.Map[key0])
	}
	if val1 != userVal.Map[key1] {
		t.Errorf("%s: expected %q, got %q", key1, val1, userVal.Map[key1])
	}
	if val2 != userVal.Map[key2] {
		t.Errorf("%s: expected %q, got %q", key2, val2, userVal.Map[key2])
	}
}

func TestMapDynamic(t *testing.T) {
	key0, key1, key2 := "Elem0", "Elem1", "Elem2"
	val0, val1, val2 := "foo", 3, true

	fcfVal := Value{
		Fields: map[string]interface{}{
			"Map": map[string]interface{}{
				"mapValue": map[string]interface{}{
					"fields": map[string]interface{}{
						key0: map[string]interface{}{"stringValue": val0},
						key1: map[string]interface{}{"integerValue": strconv.Itoa(val1)},
						key2: map[string]interface{}{"booleanValue": val2},
					},
				},
			},
		},
	}

	userVal := &struct {
		Map map[string]interface{}
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}

	if 3 != len(userVal.Map) {
		t.Fatalf("Map length mismatch: expected %v, got %v", 3, userVal.Map)
	}

	if val0 != userVal.Map[key0] {
		t.Errorf("%s: expected %q, got %q", key0, val0, userVal.Map[key0])
	}
	if val1 != userVal.Map[key1] {
		t.Errorf("%s: expected %v, got %v", key1, val1, userVal.Map[key1])
	}
	if val2 != userVal.Map[key2] {
		t.Errorf("%s: expected %v, got %v", key2, val2, userVal.Map[key2])
	}
}

func TestMapAsInterface(t *testing.T) {
	key0, key1, key2 := "Elem0", "Elem1", "Elem2"
	val0, val1, val2 := "foo", 3, true

	fcfVal := Value{
		Fields: map[string]interface{}{
			"Map": map[string]interface{}{
				"mapValue": map[string]interface{}{
					"fields": map[string]interface{}{
						key0: map[string]interface{}{"stringValue": val0},
						key1: map[string]interface{}{"integerValue": strconv.Itoa(val1)},
						key2: map[string]interface{}{"booleanValue": val2},
					},
				},
			},
		},
	}

	userVal := &struct {
		Map interface{}
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}

	userMap, ok := userVal.Map.(map[string]interface{})
	if !ok {
		t.Fatalf("type assertion failed: expected %v, got %v", userMap, userVal.Map)
	}

	if 3 != len(userMap) {
		t.Fatalf("Map length mismatch: expected %v, got %v", 3, userMap)
	}

	if val0 != userMap[key0] {
		t.Errorf("%s: expected %q, got %q", key0, val0, userMap[key0])
	}
	if val1 != userMap[key1] {
		t.Errorf("%s: expected %v, got %v", key1, val1, userMap[key1])
	}
	if val2 != userMap[key2] {
		t.Errorf("%s: expected %v, got %v", key2, val2, userMap[key2])
	}
}

// func TestStructInMap(t *testing.T) {

// 	val0, val1, val2 := "foo", "bar", "baz"
// 	fcfVal := Value{
// 		Fields: map[string]interface{}{
// 			"Outer": map[string]interface{}{
// 				"mapValue": map[string]interface{}{
// 					"fields": map[string]interface{}{
// 						"Inner": map[string]interface{}{
// 							"mapValue": map[string]interface{}{
// 								"fields": map[string]interface{}{
// 									"Elem0": map[string]interface{}{"stringValue": val0},
// 									"Elem1": map[string]interface{}{"stringValue": val1},
// 									"Elem2": map[string]interface{}{"stringValue": val2},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// }

func TestMapNested(t *testing.T) {
	key0, key1, key2 := "Elem0", "Elem1", "Elem2"
	val0, val1, val2 := "foo", "bar", "baz"

	fcfVal := Value{
		Fields: map[string]interface{}{
			"Outer": map[string]interface{}{
				"mapValue": map[string]interface{}{
					"fields": map[string]interface{}{
						"Inner": map[string]interface{}{
							"mapValue": map[string]interface{}{
								"fields": map[string]interface{}{
									key0: map[string]interface{}{"stringValue": val0},
									key1: map[string]interface{}{"stringValue": val1},
									key2: map[string]interface{}{"stringValue": val2},
									// "foo": map[string]interface{}{"decimalValue": 2.5},
								},
							},
						},
						// "Inner2": map[string]interface{}{
						// 	"mapValue": map[string]interface{}{
						// 		"fields": map[string]interface{}{
						// 			key0: map[string]interface{}{"stringValue": val0},
						// 			key1: map[string]interface{}{"stringValue": val1},
						// 			key2: map[string]interface{}{"stringValue": val2},
						// 		},
						// 	},
						// },
					},
				},
			},
		},
	}

	type elems struct {
		Elem0 string
		Elem1 string
		Elem2 string
	}
	userVal := &struct {
		S struct {
			Inner elems
			I1    interface{}            `fcf:"Inner"`
			I2    map[string]interface{} `fcf:"Inner"`
			I3    map[string]string      `fcf:"Inner"`
		} `fcf:"Outer"`
		O1 interface{}                       `fcf:"Outer"`
		O2 map[string]interface{}            `fcf:"Outer"`
		O3 map[string]map[string]interface{} `fcf:"Outer"`
		O4 map[string]map[string]string      `fcf:"Outer"`
		// O5 map[string]elems                  `fcf:"Outer"`
		O6 map[string]*elems `fcf:"Outer"`
	}{}
	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}

	// dump("O5", userVal.O5)
	dump("O6", userVal.O6)

	if val0 != userVal.S.Inner.Elem0 {
		t.Errorf("S.Inner.%s: expected %q, got %q", key0, val0, userVal.S.Inner.Elem0)
	}
	if val1 != userVal.S.Inner.Elem1 {
		t.Errorf("S.Inner.%s: expected %q, got %q", key1, val1, userVal.S.Inner.Elem1)
	}
	if val2 != userVal.S.Inner.Elem2 {
		t.Errorf("S.Inner.%s: expected %q, got %q", key2, val2, userVal.S.Inner.Elem2)
	}
	testInner := map[string]string{
		key0: val0,
		key1: val1,
		key2: val2,
	}

	compareL1 := func(name string, testMap map[string]string, usrMap map[string]interface{}) {
		if len(testMap) != len(usrMap) {
			t.Errorf("%s: expected %v, got %v", name, testMap, usrMap)
		} else {
			for key, tVal := range testMap {
				if tVal != usrMap[key] {
					t.Errorf("%s[%s]: expected %q, got %q", name, key, tVal, usrMap[key])
				}
			}
		}
	}

	compareL1("I1", testInner, userVal.S.I1.(map[string]interface{}))
	compareL1("I2", testInner, userVal.S.I2)
	// convert to map[string]interface{}
	i3 := map[string]interface{}{}
	for k, v := range userVal.S.I3 {
		i3[k] = v
	}
	compareL1("I3", testInner, i3)

	testOuter := map[string]map[string]string{
		"Inner": testInner,
		// "Inner2": testInner,
	}

	compareL2 := func(name string, testMap map[string]map[string]string, usrMap map[string]interface{}) {
		if len(testOuter) != len(usrMap) {
			t.Errorf("%s: expected %v, got %v", name, testInner, usrMap)
		} else {
			for key, tVal := range testOuter {
				compareL1(name, tVal, usrMap[key].(map[string]interface{}))
			}
		}
	}
	compareL2("O1", testOuter, userVal.O1.(map[string]interface{}))
	compareL2("O2", testOuter, userVal.O2)
	// convert to map[string]interface{}
	o3 := map[string]interface{}{}
	for k, v := range userVal.O3 {
		o3[k] = v
	}
	compareL2("O3", testOuter, o3)
	// convert to map[string]interface{}
	o4 := map[string]interface{}{}
	for k1, v1 := range userVal.O4 {
		inner := map[string]interface{}{}
		for k2, v2 := range v1 {
			inner[k2] = v2
		}
		o4[k1] = inner
	}
	compareL2("O4", testOuter, o4)
}

func dump(prefix string, v interface{}) {
	for _, line := range strings.Split(spew.Sdump(v), "\n") {
		if line != "" {
			fmt.Printf("%s: %s\n", prefix, line)
		}
	}
}

func TestArray(t *testing.T) {
	testVal := []string{"elem0", "elem1", "elem2"}

	fcfVal := Value{
		Fields: map[string]interface{}{
			"Field": map[string]interface{}{
				"arrayValue": map[string]interface{}{
					"values": []interface{}{
						map[string]interface{}{"stringValue": testVal[0]},
						map[string]interface{}{"stringValue": testVal[1]},
						map[string]interface{}{"stringValue": testVal[2]},
					},
				},
			},
		},
	}

	userVal := &struct {
		Field []string
		F1    []interface{} `fcf:"Field"`
		F2    interface{}   `fcf:"Field"`
	}{}

	err := fcfVal.Decode(userVal)
	if err != nil {
		t.Fatal(err)
	}

	if len(testVal) != len(userVal.Field) {
		t.Fatalf("array length mismatch: expected %v, got %v", testVal, userVal.Field)
	}
	for i, tVal := range testVal {
		uVal := userVal.Field[i]
		if tVal != uVal {
			t.Errorf("idx %v: expected %q, got %q", i, tVal, uVal)
		}
	}
	for i, tVal := range testVal {
		uVal := userVal.F1[i]
		if tVal != uVal {
			t.Errorf("idx %v: expected %q, got %q", i, tVal, uVal)
		}
	}
	f2 := userVal.F2.([]interface{})
	for i, tVal := range testVal {
		uVal := f2[i]
		if tVal != uVal {
			t.Errorf("idx %v: expected %q, got %q", i, tVal, uVal)
		}
	}
}
