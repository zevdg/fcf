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
	return unmarshal(reflect.Indirect(reflect.ValueOf(v)).FieldByName("Fields"), root{reflect.Indirect(reflect.ValueOf(u))})
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
		(fcfType == "arrayValue" && userKind == reflect.Slice) ||
		((fcfType == "stringValue" || fcfType == "referenceValue") && userKind == reflect.String) ||
		(fcfType == "booleanValue" && userKind == reflect.Bool) ||
		(fcfType == "geoPointValue" && userKind == reflect.Struct) ||
		fcfType == "nullValue" || fcfType == "" {
		return nil
	}
	return fmt.Errorf("type mismatch: Cannot unmarshal firestore %s into a %s field", fcfType, userKind)
}

type structField struct {
	name    string
	fcfType string
	fcf     reflect.Value
	val     reflect.Value
}

func (f structField) String() string {
	return fmt.Sprintf("structField{ %s - %s - %s }", f.name, f.fcfType, info(f.val))
}

func (f structField) Name() string {
	return f.name
}

func (f structField) FcfType() string {
	return f.fcfType
}

func (f structField) Fcf() reflect.Value {
	return f.fcf
}

func (f structField) Type() reflect.Type {
	return f.val.Type()
}

func (f structField) getOrInit() reflect.Value {
	return f.val
}

func (f structField) Set(newVal reflect.Value) {
	f.val.Set(newVal)
}

type mapField struct {
	name    string
	key     reflect.Value
	fcfType string
	fcf     reflect.Value
	parent  reflect.Value
}

func (f mapField) String() string {
	return fmt.Sprintf("mapField{ %s - %s - %s }", f.key, f.fcfType, info(f.parent))
}

func (f mapField) Name() string {
	return f.name
}
func (f mapField) FcfType() string {
	return f.fcfType
}

func (f mapField) Fcf() reflect.Value {
	return f.fcf
}

func (f mapField) Type() reflect.Type {
	return f.parent.Type().Elem()
}

func (f mapField) getOrInit() reflect.Value {
	v := f.parent.MapIndex(f.key)
	if v.IsValid() {
		return v
	}
	f.Set(reflect.Zero(f.Type()))
	return f.parent.MapIndex(f.key)
}

func (f mapField) Set(newVal reflect.Value) {
	f.parent.SetMapIndex(f.key, newVal)
}

type sliceField struct {
	name    string
	i       int
	fcfType string
	fcf     reflect.Value
	parent  reflect.Value
}

func (f sliceField) String() string {
	return fmt.Sprintf("mapField{ %s - %s - %s }", f.Name(), f.fcfType, info(f.parent))
}

func (f sliceField) Name() string {
	return f.name
}
func (f sliceField) FcfType() string {
	return f.fcfType
}

func (f sliceField) Fcf() reflect.Value {
	return f.fcf
}

func (f sliceField) Type() reflect.Type {
	return f.parent.Type().Elem()
}

func (f sliceField) getOrInit() reflect.Value {
	if f.parent.Len() > f.i {
		return f.parent.Index(f.i)
	}
	v := reflect.Zero(f.Type())
	f.parent = reflect.Append(f.parent, v)
	return v
}

func (f sliceField) Set(newVal reflect.Value) {
	f.parent.Index(f.i).Set(newVal)
}

type root struct {
	val reflect.Value
}

func (r root) Name() string {
	return ""
}

func (r root) getOrInit() reflect.Value {
	return r.val
}

func (r root) Set(newVal reflect.Value) {
	r.val.Set(newVal)
}

type field interface {
	FcfType() string
	Fcf() reflect.Value
	Type() reflect.Type
	String() string
	fieldBag
}

type fieldBag interface {
	Name() string
	getOrInit() reflect.Value
	Set(reflect.Value)
}

func unwrapFcfVal(wrappedVal reflect.Value) (unwrappedVal reflect.Value, fcfType string) {
	wrappedVal = wrappedVal.Elem() // sheds interface{} outer layer
	if wrappedVal.Kind() != reflect.Map {
		// raw value special case (e.g. GeoPoint fields)
		return wrappedVal, ""
	}
	fcfUnionType := wrappedVal.MapKeys()[0]
	return wrappedVal.MapIndex(fcfUnionType).Elem(), fcfUnionType.String()
}

func getSliceFields(fcfVal reflect.Value, uVal fieldBag) (fields []field, err error) {
	usrVal, parentName := uVal.getOrInit(), uVal.Name()

	if !(usrVal.Kind() == reflect.Slice ||
		(usrVal.Kind() == reflect.Interface && usrVal.Type().NumMethod() == 0)) {
		typeStr := usrVal.Kind().String()
		if usrVal.IsValid() {
			typeStr = usrVal.Type().String()
		}
		return nil, fmt.Errorf("Can only unmarshal array types into slice or empty interface fields, not %v", typeStr)
	}

	var sliceType reflect.Type
	if usrVal.Kind() == reflect.Interface {
		var x []interface{}
		sliceType = reflect.TypeOf(x)
	} else {
		sliceType = usrVal.Type()
	}
	if usrVal.IsNil() {
		usrVal = reflect.MakeSlice(sliceType, fcfVal.Len(), fcfVal.Len())
		uVal.Set(usrVal)
	}
	for i := 0; i < fcfVal.Len(); i++ {

		fcfFieldVal, fcfType := unwrapFcfVal(fcfVal.Index(i))
		fields = append(fields, sliceField{
			name:    fmt.Sprintf("%s[%d]", parentName, i),
			i:       i,
			fcfType: fcfType,
			fcf:     fcfFieldVal,
			parent:  usrVal,
		})
	}
	return fields, nil
}

func getStructFields(fcfVal reflect.Value, usrVal reflect.Value, parentName string) (fields []field) {
	for i := 0; i < usrVal.Type().NumField(); i++ {
		fieldMeta := usrVal.Type().Field(i)
		key := fieldMeta.Name
		if tag := fieldMeta.Tag.Get("fcf"); tag != "" {
			key = tag
		}
		wrappedVal := fcfVal.MapIndex(reflect.ValueOf(key))
		if !wrappedVal.IsValid() {
			// field on user's struct doesn't exist in firestore data
			// skip it
			continue
		}
		fcfFieldVal, fcfType := unwrapFcfVal(wrappedVal)
		fieldVal := usrVal.Field(i)
		if fieldVal.Kind() == reflect.Ptr && fcfType != "nullValue" {
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
			}
			fieldVal = fieldVal.Elem()
		}
		name := fieldMeta.Name
		if parentName != "" {
			name = parentName + "." + name
		}
		fields = append(fields, structField{
			name:    name,
			fcfType: fcfType,
			fcf:     fcfFieldVal,
			val:     fieldVal,
		})
	}
	return fields
}

func getMapFields(fcfVal reflect.Value, uVal fieldBag) (fields []field, err error) {
	usrVal, parentName := uVal.getOrInit(), uVal.Name()

	if !((usrVal.Kind() == reflect.Interface && usrVal.Type().NumMethod() == 0) ||
		usrVal.Kind() == reflect.Struct ||
		usrVal.Kind() == reflect.Map) {
		typeStr := usrVal.Kind().String()
		if usrVal.IsValid() {
			typeStr = usrVal.Type().String()
		}
		return nil, fmt.Errorf("Can only unmarshal object/map types into Struct, Map, or empty interface fields, not %v", typeStr)
	}

	if usrVal.Kind() == reflect.Struct {
		// get fields from usrVal
		return getStructFields(fcfVal, usrVal, parentName), nil
	}

	// usrVal is Map, Slice, or empty interface
	// get fields from fcfVal
	var mapType reflect.Type
	if usrVal.Kind() == reflect.Interface {
		var x map[string]interface{}
		mapType = reflect.TypeOf(x)
	} else {
		mapType = usrVal.Type()
	}
	if usrVal.IsNil() {
		usrVal = reflect.MakeMapWithSize(mapType, len(fcfVal.MapKeys()))
		uVal.Set(usrVal)
	}
	for _, key := range fcfVal.MapKeys() {
		fcfFieldVal, fcfType := unwrapFcfVal(fcfVal.MapIndex(key))
		fields = append(fields, mapField{
			name:    fmt.Sprintf("%s[%q]", parentName, key),
			key:     key,
			fcfType: fcfType,
			fcf:     fcfFieldVal,
			parent:  usrVal,
		})
	}
	return fields, nil
}

func getFields(fcfVal reflect.Value, uVal fieldBag) (fields []field, err error) {
	if fcfVal.Kind() == reflect.Slice {
		return getSliceFields(fcfVal, uVal)
	}
	return getMapFields(fcfVal, uVal)
}

func unmarshal(fcfMap reflect.Value, usrVal fieldBag) error {
	fields, err := getFields(fcfMap, usrVal)
	if err != nil {
		return err
	}
	for _, field := range fields {
		fcfVal := field.Fcf()
		err := assertTypeMatch(field.Type(), field.FcfType())
		if err != nil {
			return fmt.Errorf("Error unmarshalling field %s: %v", field.Name(), err)
		}

		switch field.FcfType() {
		case "mapValue":
			fcfVal = fcfVal.MapIndex(reflect.ValueOf("fields")).Elem()
		case "arrayValue":
			fcfVal = fcfVal.MapIndex(reflect.ValueOf("values")).Elem()
		case "geoPointValue":
			// do nothing
		default:
			if err := setBasicType(field); err != nil {
				return fmt.Errorf("Error setting field %s: %v", field.Name(), err)
			}
			continue
		}
		if err := unmarshal(fcfVal, field); err != nil {
			return err
		}
	}
	return nil
}

// Conversions
func setBasicType(field field) error {
	fcfVal := field.Fcf()
	fieldType := field.Type()
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
	}

	fcfVal = fcfVal.Convert(fieldType)
	field.Set(fcfVal)
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

// Helpers

func d(prefix string, x reflect.Value) {
	fmt.Printf("%s: %s\n", prefix, info(x))
}
func info(x reflect.Value) string {
	return fmt.Sprintf("%v | %v | %v\n", x.Kind(), x.Type(), x)
}
