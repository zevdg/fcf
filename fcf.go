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

type GeoPoint struct {
	Latitude  float64 `fcf:"latitude"`
	Longitude float64 `fcf:"longitude"`
}

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
		(fcfType == "geoPointValue" && userKind == reflect.Struct) ||
		fcfType == "nullValue" || fcfType == "" {
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

func (f staticField) String() string {
	return fmt.Sprintf("staticField{ %s - %s - %s }", f.name, f.fcfType, info(f.val))
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

func (f staticField) Type() reflect.Type {
	return f.val.Type()
}

func (f staticField) Set(newVal reflect.Value) {
	f.val.Set(newVal)
}

type dynamicField struct {
	key     reflect.Value
	fcfType string
	fcf     reflect.Value
	parent  reflect.Value
}

func (f dynamicField) String() string {
	return fmt.Sprintf("dynamicField{ %s - %s - %s }", f.key, f.fcfType, info(f.parent))
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

func (f dynamicField) Type() reflect.Type {
	return f.parent.Type().Elem()
}

func (f dynamicField) getOrInit() reflect.Value {
	v := f.parent.MapIndex(f.key)
	if v.IsValid() {
		return v
	}
	f.Set(reflect.Zero(f.Type()))
	return f.parent.MapIndex(f.key)
}

func (f dynamicField) Set(newVal reflect.Value) {
	f.parent.SetMapIndex(f.key, newVal)
}

type field interface {
	Name() string
	FcfType() string
	Fcf() reflect.Value
	Type() reflect.Type
	String() string
	setter
}

type setter interface {
	Set(reflect.Value)
}

func unwrap(wrappedVal reflect.Value) (unwrappedVal reflect.Value, fcfType string) {
	wrappedVal = wrappedVal.Elem() // sheds interface{} outer layer
	if wrappedVal.Kind() != reflect.Map {
		// raw value special case (e.g. GeoPoint fields)
		return wrappedVal, ""
	}
	fcfUnionType := wrappedVal.MapKeys()[0]
	return wrappedVal.MapIndex(fcfUnionType).Elem(), fcfUnionType.String()
}

func getFields(fcfMap reflect.Value, uVal setter) (fields []field, err error) {
	var usrVal reflect.Value
	switch v := uVal.(type) {
	case reflect.Value:
		usrVal = v
	case staticField:
		usrVal = v.val
	case dynamicField:
		usrVal = v.getOrInit()
	}

	if !((usrVal.Kind() == reflect.Interface && usrVal.Type().NumMethod() == 0) ||
		usrVal.Kind() == reflect.Struct ||
		usrVal.Kind() == reflect.Map) {
		typeStr := usrVal.Kind().String()
		if usrVal.IsValid() {
			typeStr = usrVal.Type().String()
		}
		return nil, fmt.Errorf("Can only get fields from Struct, Map, or empty interface types, not %v", typeStr)
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
		return fields, nil
	}

	var mapType reflect.Type
	if usrVal.Kind() == reflect.Interface {
		var x map[string]interface{}
		mapType = reflect.TypeOf(x)
	} else {
		mapType = usrVal.Type()
	}
	if usrVal.IsNil() {
		usrVal = reflect.MakeMapWithSize(mapType, len(fcfMap.MapKeys()))
		uVal.Set(usrVal)
	}
	for _, key := range fcfMap.MapKeys() {
		fcfVal, fcfType := unwrap(fcfMap.MapIndex(key))
		fields = append(fields, dynamicField{
			key:     key,
			fcfType: fcfType,
			fcf:     fcfVal,
			parent:  usrVal,
		})
	}
	return fields, nil
}

func unmarshal(fcfMap reflect.Value, usrVal setter) error {
	fields, err := getFields(fcfMap, usrVal)
	if err != nil {
		return err
	}
	for _, field := range fields {
		fcfVal := field.Fcf()
		fieldType := field.Type()

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
			fcfVal = fcfVal.MapIndex(reflect.ValueOf("fields")).Elem()
			fallthrough
		case "geoPointValue":
			err = unmarshal(fcfVal, field)
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

// Conversions

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

// Helpers

func d(prefix string, x reflect.Value) {
	fmt.Printf("%s: %s\n", prefix, info(x))
}
func info(x reflect.Value) string {
	return fmt.Sprintf("%v | %v | %v\n", x.Kind(), x.Type(), x)
}
