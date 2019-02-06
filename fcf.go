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
	return Unmarshal(v, u)
}

func assertTypeMatch(userType reflect.Type, typeIndicator string) error {
	userKind := userType.Kind()
	if userKind == reflect.Interface {
		if userType.NumMethod() == 0 {
			return nil // empty interface is allowed
		}
		return fmt.Errorf("type mismatch: Cannot unmarshal firestore values into non-empty interface: %v", userType)
	}

	if (typeIndicator == "integerValue" && reflect.Int <= userKind && userKind <= reflect.Float64) ||
		(typeIndicator == "doubleValue" && (userKind == reflect.Float32 || userKind == reflect.Float64)) ||
		(typeIndicator == "timestampValue" && userType.PkgPath() == "time" && userType.Name() == "Time") ||
		(typeIndicator == "mapValue" && (userKind == reflect.Struct /* || userKind == reflect.Map */)) ||
		((typeIndicator == "stringValue" || typeIndicator == "referenceValue") && userKind == reflect.String) ||
		(typeIndicator == "booleanValue" && userKind == reflect.Bool) ||
		typeIndicator == "nullValue" {
		return nil
	}
	return fmt.Errorf("type mismatch: Cannot unmarshal firestore %s into a %s field", typeIndicator, userKind)
}

func Unmarshal(fcfMap interface{}, usrStruct interface{}) error {
	_, err := unmarshalToStruct(reflect.Indirect(reflect.ValueOf(fcfMap)).FieldByName("Fields"), reflect.Indirect(reflect.ValueOf(usrStruct)))
	return err
}

func unmarshalToStruct(fcfMap reflect.Value, usrVal reflect.Value) (reflect.Value, error) {
	for i := 0; i < usrVal.Type().NumField(); i++ {
		fieldMeta := usrVal.Type().Field(i)
		key := fieldMeta.Name
		if tag := fieldMeta.Tag.Get("fcf"); tag != "" {
			key = tag
		}
		wrappedVal := fcfMap.MapIndex(reflect.ValueOf(key))
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
			return usrVal, fmt.Errorf("Error unmarshalling field %s: %v", fieldMeta.Name, err)
		}
		switch typeIndicator {
		case "referenceValue":
			fcfVal = convReference(fcfVal)

		case "timestampValue":
			fcfVal = convTimestamp(fcfVal)

		case "integerValue":
			bits := fieldVal.Type().Bits()
			if fieldVal.Kind() <= reflect.Int64 {
				fcfVal = convInt(fcfVal, bits)
			} else if fieldVal.Kind() <= reflect.Uintptr {
				fcfVal = convUint(fcfVal, bits)
			} else {
				fcfVal = convIntegerToFloat(fcfVal, bits)

			}
		case "nullValue":
			fcfVal = reflect.Zero(fieldVal.Type())

		case "mapValue":
			fcfVal, err = unmarshalToStruct(fcfVal.MapIndex(reflect.ValueOf("fields")).Elem(), fieldVal)
			if err != nil {
				return usrVal, err
			}
			continue
		}
		fcfVal = fcfVal.Convert(fieldVal.Type())
		fieldVal.Set(fcfVal)
	}
	return usrVal, nil
}

func convReference(fcfVal reflect.Value) reflect.Value {
	return reflect.ValueOf(strings.Split(fcfVal.String(), "/databases/(default)/documents")[1])
}

func convTimestamp(fcfVal reflect.Value) reflect.Value {
	t, err := time.Parse(time.RFC3339Nano, fcfVal.String())
	if err != nil {
		panic(err)
	}
	return reflect.ValueOf(t)
}

func convInt(fcfVal reflect.Value, bits int) reflect.Value {
	val, err := strconv.ParseInt(fcfVal.String(), 0, bits)
	if err != nil {
		panic(err) // shouldn't ever happen?
	}
	return reflect.ValueOf(val)
}

func convUint(fcfVal reflect.Value, bits int) reflect.Value {
	val, err := strconv.ParseUint(fcfVal.String(), 0, bits)
	if err != nil {
		panic(err) // shouldn't ever happen?
	}
	return reflect.ValueOf(val)
}

func convIntegerToFloat(fcfVal reflect.Value, bits int) reflect.Value {
	val, err := strconv.ParseFloat(fcfVal.String(), bits)
	if err != nil {
		panic(err) // shouldn't ever happen?
	}
	return reflect.ValueOf(val)
}
