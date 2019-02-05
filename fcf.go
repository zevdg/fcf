package fcf // import "github.com/zevdg/fcf"

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// FirestoreEvent is the payload of a Firestore event.
type Event struct {
	OldValue   Value `json:"oldValue"`
	Value      Value `json:"value"`
	UpdateMask struct {
		FieldPaths []string `json:"fieldPaths"`
	} `json:"updateMask"`
}

// FirestoreValue holds Firestore fields.
type Value struct {
	CreateTime time.Time `json:"createTime"`
	// Fields is the data for this value. The type depends on the format of your
	// database. Log an interface{} value and inspect the result to see a JSON
	// representation of your database fields.
	Fields     map[string]interface{} `json:"fields"`
	Name       string                 `json:"name"`
	UpdateTime time.Time              `json:"updateTime"`
}

// type Geopoint struct {
// 	Latitude  float64
// 	Longitude float64
// }

// Decode reads the raw data from the fcf Value
// and stores it in the user value pointed to by u
func (v Value) Decode(u interface{}) error {
	return unmarshalMap(v, u)
}

func assertTypeMatch(userType reflect.Type, typeIndicator string) error {
	userKind := userType.Kind()
	m := map[string]reflect.Kind{
		"stringValue":    reflect.String,
		"booleanValue":   reflect.Bool,
		"referenceValue": reflect.String,
	}
	if userKind == reflect.Interface {
		if userType.NumMethod() == 0 {
			return nil // empty interface is allowed
		}
		return fmt.Errorf("type mismatch: Cannot unmarshal firestore values into non-empty interface: %v", userType)
	}

	if (typeIndicator == "integerValue" && reflect.Int <= userKind && userKind <= reflect.Float64) ||
		(typeIndicator == "doubleValue" && (userKind == reflect.Float32 || userKind == reflect.Float64)) ||
		(typeIndicator == "timestampValue" && userType.PkgPath() == "time" && userType.Name() == "Time") ||
		typeIndicator == "nullValue" ||
		userKind == m[typeIndicator] {
		return nil
	}
	return fmt.Errorf("type mismatch: Cannot unmarshal firestore %s into a %s field", typeIndicator, userKind)
}

func unmarshalMap(fcfMap interface{}, usrStruct interface{}) error {
	mapFields := reflect.Indirect(reflect.ValueOf(fcfMap)).FieldByName("Fields")
	usrVal := reflect.Indirect(reflect.ValueOf(usrStruct))
	for i := 0; i < usrVal.Type().NumField(); i++ {
		fieldMeta := usrVal.Type().Field(i)
		key := fieldMeta.Name
		if tag := fieldMeta.Tag.Get("fcf"); tag != "" {
			key = tag
		}
		wrappedVal := mapFields.MapIndex(reflect.ValueOf(key))
		if !wrappedVal.IsValid() {
			// field on user's struct doesn't exist in firestore data
			// skip it
			continue
		}
		wrappedVal = wrappedVal.Elem()
		fcfUnionType := wrappedVal.MapKeys()[0]
		fcfVal := wrappedVal.MapIndex(fcfUnionType).Elem()
		typeIndicator := fcfUnionType.String()

		fieldVal := usrVal.Field(i)
		if fieldVal.Kind() == reflect.Ptr && typeIndicator != "nullValue" {
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
			}
			fieldVal = fieldVal.Elem()
		}

		err := assertTypeMatch(fieldVal.Type(), typeIndicator)
		if err != nil {
			return fmt.Errorf("Error unmarshalling field %s: %v", fieldMeta.Name, err)
		}
		switch typeIndicator {
		case "referenceValue":
			unmarshalReference(fcfVal, fieldVal)
		case "timestampValue":
			unmarshalTimestamp(fcfVal, fieldVal)
		case "integerValue":
			if fieldVal.Kind() <= reflect.Int64 {
				unmarshalInt(fcfVal, fieldVal)
			} else if fieldVal.Kind() <= reflect.Uintptr {
				unmarshalUint(fcfVal, fieldVal)
			} else {
				unmarshalIntegerToFloat(fcfVal, fieldVal)
			}
		case "nullValue":
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
		default:
			// the conversion was added for float64 -> float32
			// seems safe to do for all types
			// but may need to be restricted if it causes problems
			fcfVal = fcfVal.Convert(fieldVal.Type())
			fieldVal.Set(fcfVal)
		}
	}
	return nil
}

func unmarshalReference(fcfVal reflect.Value, fieldVal reflect.Value) {
	fieldVal.SetString(strings.Split(fcfVal.String(), "/databases/(default)/documents")[1])
}

func unmarshalInt(fcfVal reflect.Value, fieldVal reflect.Value) {
	val, err := strconv.ParseInt(fcfVal.String(), 0, fieldVal.Type().Bits())
	if err != nil {
		panic(err) // shouldn't ever happen?
	}
	fieldVal.SetInt(val)
}

func unmarshalUint(fcfVal reflect.Value, fieldVal reflect.Value) {
	val, err := strconv.ParseUint(fcfVal.String(), 0, fieldVal.Type().Bits())
	if err != nil {
		panic(err) // shouldn't ever happen?
	}
	fieldVal.SetUint(val)
}

func unmarshalIntegerToFloat(fcfVal reflect.Value, fieldVal reflect.Value) {
	val, err := strconv.ParseFloat(fcfVal.String(), fieldVal.Type().Bits())
	if err != nil {
		panic(err) // shouldn't ever happen?
	}
	fieldVal.SetFloat(val)
}

func unmarshalDecimal(fcfVal reflect.Value, fieldVal reflect.Value) {
	if fieldVal.Type().Bits() == 32 {
		fieldVal.Set(fcfVal.Convert(fieldVal.Type()))
	} else {
		fieldVal.Set(fcfVal)
	}
}

func unmarshalTimestamp(fcfVal reflect.Value, fieldVal reflect.Value) {
	t, err := time.Parse(time.RFC3339Nano, fcfVal.String())
	if err != nil {
		panic(err)
	}
	fieldVal.Set(reflect.ValueOf(t))
}
