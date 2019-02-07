package fcf // import "github.com/zevdg/fcf"

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Event is the payload of a Firestore event.
type Event struct {
	OldValue   Value `json:"oldValue"`
	Value      Value `json:"value"`
	UpdateMask struct {
		FieldPaths []string `json:"fieldPaths"`
	} `json:"updateMask"`
}

// Value holds Firestore fields.
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
	return unmarshal(reflect.Indirect(reflect.ValueOf(v)).FieldByName("Fields"), reflect.Indirect(reflect.ValueOf(u)))
}

func assertTypeMatch(userType reflect.Type, fcfType string) error {
	userKind := userType.Kind()
	if userKind == reflect.Interface {
		if userType.NumMethod() == 0 {
			return nil // empty interface is allowed
		}
		return fmt.Errorf("type mismatch: Cannot unmarshal firestore values into non-empty interface: %v", userType)
	}

	if (fcfType == "integerValue" && reflect.Int <= userKind && userKind <= reflect.Float64) ||
		(fcfType == "doubleValue" && (userKind == reflect.Float32 || userKind == reflect.Float64)) ||
		(fcfType == "timestampValue" && userType.PkgPath() == "time" && userType.Name() == "Time") ||
		(fcfType == "mapValue" && (userKind == reflect.Struct || userKind == reflect.Map)) ||
		((fcfType == "stringValue" || fcfType == "referenceValue") && userKind == reflect.String) ||
		(fcfType == "booleanValue" && userKind == reflect.Bool) ||
		fcfType == "nullValue" {
		return nil
	}
	return fmt.Errorf("type mismatch: Cannot unmarshal firestore %s into a %s field", fcfType, userKind)
}

type staticField struct {
	name    string
	fcfType string
	fcf     reflect.Value
	val     reflect.Value
}

func (f staticField) Name() string {
	return f.name
}

func (f staticField) FcfType() string {
	return f.fcfType
}

func (f staticField) Fcf() reflect.Value {
	return f.fcf
}

func (f staticField) Val() reflect.Value {
	return f.val
}

func (f staticField) Set(newVal reflect.Value) {
	f.val.Set(newVal)
}

type dynamicField struct {
	key     reflect.Value
	fcfType string
	fcf     reflect.Value
	parent  reflect.Value
	val     reflect.Value
}

func (f dynamicField) Name() string {
	return f.key.String()
}
func (f dynamicField) FcfType() string {
	return f.fcfType
}

func (f dynamicField) Fcf() reflect.Value {
	return f.fcf
}

func (f dynamicField) Val() reflect.Value {
	return f.val
}

func (f dynamicField) Set(newVal reflect.Value) {
	f.parent.SetMapIndex(f.key, newVal)
}

type field interface {
	Name() string
	FcfType() string
	Fcf() reflect.Value
	Val() reflect.Value
	Set(reflect.Value)
}

func unwrap(wrappedVal reflect.Value) (unwrappedVal reflect.Value, fcfType string) {
	wrappedVal = wrappedVal.Elem() // sheds interface{} outer layer to reveal map[string]interface{}
	fcfUnionType := wrappedVal.MapKeys()[0]
	return wrappedVal.MapIndex(fcfUnionType).Elem(), fcfUnionType.String()
}

func getFields(fcfMap reflect.Value, usrVal reflect.Value) (fields []field) {
	if usrVal.Kind() == reflect.Interface && usrVal.Type().NumMethod() == 0 {
		var nilMap map[string]interface{}
		usrVal.Set(reflect.MakeMapWithSize(reflect.TypeOf(nilMap), len(fcfMap.MapKeys())))
		usrVal = usrVal.Elem()
	}
	if usrVal.Kind() == reflect.Map {
		if usrVal.IsNil() {
			usrVal.Set(reflect.MakeMapWithSize(usrVal.Type(), len(fcfMap.MapKeys())))
		}
		fieldType := usrVal.Type().Elem()
		for _, key := range fcfMap.MapKeys() {
			fcfVal, fcfType := unwrap(fcfMap.MapIndex(key))
			fields = append(fields, dynamicField{
				key:     key,
				fcfType: fcfType,
				fcf:     fcfVal,
				parent:  usrVal,
				val:     reflect.Zero(fieldType),
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
			fcfVal, fcfType := unwrap(wrappedVal)

			fieldVal := usrVal.Field(i)
			if fieldVal.Kind() == reflect.Ptr && fcfType != "nullValue" {
				if fieldVal.IsNil() {
					fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
				}
				fieldVal = fieldVal.Elem()
			}
			fields = append(fields, staticField{
				name:    fieldMeta.Name,
				fcfType: fcfType,
				fcf:     fcfVal,
				val:     fieldVal,
			})
		}
		return fields
	}
	panic(fmt.Sprintf("Can only get fields Struct, Map, or empty interface types, not %v", usrVal.Type()))
}

func unmarshal(fcfMap reflect.Value, usrVal reflect.Value) error {
	for _, field := range getFields(fcfMap, usrVal) {
		fcfVal := field.Fcf()
		fieldType := field.Val().Type()
		err := assertTypeMatch(fieldType, field.FcfType())
		if err != nil {
			return fmt.Errorf("Error unmarshalling field %s: %v", field.Name(), err)
		}
		switch field.FcfType() {
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
			err = unmarshal(fcfVal.MapIndex(reflect.ValueOf("fields")).Elem(), field.Val())
			if err != nil {
				return fmt.Errorf("Error on field %v: %v", field.Name(), err)
			}
			// unmarshal sets values in-place
			// so no need to convert and set
			continue
		}

		fcfVal = fcfVal.Convert(fieldType)
		field.Set(fcfVal)
	}

	return nil
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
