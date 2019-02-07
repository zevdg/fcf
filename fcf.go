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

type Field struct {
	name          string
	key           string
	typeIndicator string
	fcf           reflect.Value
	val           reflect.Value
}

func unwrap(wrappedVal reflect.Value) (unwrappedVal reflect.Value, typeIndicator string) {
	wrappedVal = wrappedVal.Elem() //is this necessary?
	fcfUnionType := wrappedVal.MapKeys()[0]
	return wrappedVal.MapIndex(fcfUnionType).Elem(), fcfUnionType.String()
}

func getFields(fcfMap reflect.Value, usrVal reflect.Value) (fields []Field) {
	if usrVal.Kind() == reflect.Map {
		fieldType := usrVal.Type().Elem()
		for _, key := range fcfMap.MapKeys() {
			fcfVal, typeIndicator := unwrap(fcfMap.MapIndex(key))
			fields = append(fields, Field{
				name:          key.String(),
				key:           key.String(),
				typeIndicator: typeIndicator,
				fcf:           fcfVal,
				val:           reflect.Zero(fieldType),
			})
		}
		return fields
	}
	if usrVal.Kind() == reflect.Struct {
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
			fcfVal, typeIndicator := unwrap(wrappedVal)

			fieldVal := usrVal.Field(i)
			if fieldVal.Kind() == reflect.Ptr && typeIndicator != "nullValue" {
				if fieldVal.IsNil() {
					fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
				}
				fieldVal = fieldVal.Elem()
			}
			fields = append(fields, Field{
				name:          fieldMeta.Name,
				key:           key,
				typeIndicator: typeIndicator,
				fcf:           fcfVal,
				val:           fieldVal,
			})
		}
		return fields
	}
	panic("can't get fields usrVal.Kind() " + usrVal.Kind().String())
}

func unmarshalToStruct(fcfMap reflect.Value, usrVal reflect.Value) (reflect.Value, error) {
	for _, field := range getFields(fcfMap, usrVal) {
		fcfVal := field.fcf
		fieldType := field.val.Type()
		err := assertTypeMatch(fieldType, field.typeIndicator)
		if err != nil {
			return usrVal, fmt.Errorf("Error unmarshalling field %s: %v", field.name, err)
		}
		switch field.typeIndicator {
		case "referenceValue":
			fcfVal = convReference(fcfVal)

		case "timestampValue":
			fcfVal = convTimestamp(fcfVal)

		case "integerValue":
			if fieldType.Kind() == reflect.Interface {
				fieldType = reflect.TypeOf(0)
			}
			bits := fieldType.Bits()
			if fieldType.Kind() <= reflect.Int64 || fieldType.Kind() == reflect.Interface {
				fcfVal = convInt(fcfVal, bits)
			} else if fieldType.Kind() <= reflect.Uintptr {
				fcfVal = convUint(fcfVal, bits)
			} else {
				fcfVal = convIntegerToFloat(fcfVal, bits)
			}

		case "nullValue":
			fcfVal = reflect.Zero(fieldType)

		case "mapValue":
			if fieldType.Kind() == reflect.Struct {
				fcfVal, err = unmarshalToStruct(fcfVal.MapIndex(reflect.ValueOf("fields")).Elem(), field.val)
				if err != nil {
					return usrVal, err
				}
				continue
			} else {

			}
		}

		fcfVal = fcfVal.Convert(fieldType)
		field.val.Set(fcfVal)
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
