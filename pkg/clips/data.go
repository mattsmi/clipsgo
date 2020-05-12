package clips

// #cgo CFLAGS: -I ../../clips_source
// #cgo LDFLAGS: -L ../../clips_source -l clips
// #include <clips/clips.h>
//
// struct multifield *multifield_ptr(void *mallocval) {
// 	 return (struct multifield *)mallocval;
// }
//
// short get_data_type(struct dataObject *data )
// {
//	 return GetpType(data);
// }
//
// short set_data_type(struct dataObject *data, int type)
// {
//   return SetpType(data, type);
// }
//
// void *get_data_value(struct dataObject *data)
// {
//   return GetpValue(data);
// }
//
// void *set_data_value(struct dataObject *data, void *value)
// {
//   return SetpValue(data, value);
// }
//
// long get_data_begin(struct dataObject *data)
// {
//   return GetpDOBegin(data);
// }
//
// long set_data_begin(struct dataObject *data, long begin)
// {
//   return SetpDOBegin(data, begin);
// }
//
// long get_data_end(struct dataObject *data)
// {
//   return GetpDOEnd(data);
// }
//
// long set_data_end(struct dataObject *data, long end)
// {
//   return SetpDOEnd(data, end);
// }
//
// long get_data_length(struct dataObject *data)
// {
//   return GetpDOLength(data);
// }
//
// short get_multifield_type(struct multifield *mf, long index)
// {
//   return GetMFType(mf, index);
// }
//
// short set_multifield_type(struct multifield *mf, long index, short type)
// {
//   return SetMFType(mf, index, type);
// }
//
// void *get_multifield_value(struct multifield *mf, long index)
// {
//   return GetMFValue(mf, index);
// }
//
// void *set_multifield_value(struct multifield *mf, long index, void *value)
// {
//   return SetMFValue(mf, index, value);
// }
//
// long get_multifield_length(struct multifield *mf)
// {
//   return GetMFLength(mf);
// }
//
// char *to_string(void *data)
// {
//   return (char *) ValueToString(data);
// }
//
// long long to_integer(void *data)
// {
//   return ValueToLong(data);
// }
//
// double to_double(void *data)
// {
//   return ValueToDouble(data);
// }
//
// void *to_pointer(void *data)
// {
//   return ValueToPointer(data);
// }
//
// void *to_external_address(void *data)
// {
//   return ValueToExternalAddress(data);
// }
import "C"
import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

// Symbol represents a CLIPS SYMBOL value
type Symbol string

// InstanceName represents a CLIPS INSTANCE_NAME value
type InstanceName Symbol

// DataObject wraps a CLIPS data object
type DataObject struct {
	env  *Environment
	typ  Type
	data *C.struct_dataObject
}

func createDataObjectInitialized(env *Environment, data *C.struct_dataObject) *DataObject {
	ret := &DataObject{
		env:  env,
		typ:  -1,
		data: data,
	}
	return ret
}

func createDataObject(env *Environment) *DataObject {
	datamem := C.malloc(C.sizeof_struct_dataObject)
	data := (*C.struct_dataObject)(datamem)
	ret := createDataObjectInitialized(env, data)
	runtime.SetFinalizer(ret, func(data *DataObject) {
		data.Delete()
	})
	return ret
}

// Delete frees up associated memory
func (do *DataObject) Delete() {
	if do.data != nil {
		C.free(unsafe.Pointer(do.data))
		do.data = nil
	}
}

func (do *DataObject) byRef() *C.struct_dataObject {
	return do.data
}

// Value returns the Go value for this data object
func (do *DataObject) Value() interface{} {
	dtype := Type(C.get_data_type(do.data))
	dvalue := C.get_data_value(do.data)

	if dvalue == C.NULL {
		return nil
	}

	return do.goValue(dtype, dvalue)
}

func (do *DataObject) clipsTypeFor(v interface{}) Type {
	if v == nil {
		return SYMBOL
	}
	switch v.(type) {
	case bool, Symbol:
		return SYMBOL
	case int, int8, int16, int32, int64:
		return INTEGER
	case float32, float64:
		return FLOAT
	case unsafe.Pointer:
		return EXTERNAL_ADDRESS
	case InstanceName:
		return INSTANCE_NAME
	case string:
		return STRING
	case *ImpliedFact, *TemplateFact:
		return FACT_ADDRESS
	case *Instance:
		return INSTANCE_ADDRESS
	default:
		switch reflect.TypeOf(v).Kind() {
		case reflect.Array, reflect.Slice:
			return MULTIFIELD
		}
	}
	return SYMBOL
}

// SetValue copies the go value into the dataobject
func (do *DataObject) SetValue(value interface{}) {
	var dtype Type
	if do.typ < 0 {
		dtype = do.clipsTypeFor(value)
	} else {
		dtype = do.typ
	}

	C.set_data_type(do.data, dtype.CVal())
	C.set_data_value(do.data, do.clipsValue(value))
}

// goValue converts a CLIPS data value into a Go data structure
func (do *DataObject) goValue(dtype Type, dvalue unsafe.Pointer) interface{} {
	switch dtype {
	case FLOAT:
		return float64(C.to_double(dvalue))
	case INTEGER:
		return int64(C.to_integer(dvalue))
	case STRING:
		return C.GoString(C.to_string(dvalue))
	case EXTERNAL_ADDRESS:
		return C.to_external_address(dvalue)
	case SYMBOL:
		cstr := C.to_string(dvalue)
		gstr := C.GoString(cstr)
		if gstr == "nil" {
			return nil
		}
		if gstr == "TRUE" {
			return true
		}
		if gstr == "FALSE" {
			return false
		}
		return Symbol(gstr)
	case INSTANCE_NAME:
		return InstanceName(C.GoString(C.to_string(dvalue)))
	case MULTIFIELD:
		return do.multifieldToList()
	case FACT_ADDRESS:
		return do.env.newFact(C.to_pointer(dvalue))
	case INSTANCE_ADDRESS:
		return createInstance(do.env, C.to_pointer(dvalue))
	}
	return nil
}

// clipsValue convers a Go data structure into a CLIPS data value
func (do *DataObject) clipsValue(dvalue interface{}) unsafe.Pointer {
	if dvalue == nil {
		vstr := C.CString("nil")
		defer C.free(unsafe.Pointer(vstr))
		return C.EnvAddSymbol(do.env.env, vstr)
	}
	switch v := dvalue.(type) {
	case bool:
		if v {
			vstr := C.CString("TRUE")
			defer C.free(unsafe.Pointer(vstr))
			return C.EnvAddSymbol(do.env.env, vstr)
		}
		vstr := C.CString("FALSE")
		defer C.free(unsafe.Pointer(vstr))
		return C.EnvAddSymbol(do.env.env, vstr)
	case int:
		return C.EnvAddLong(do.env.env, C.longlong(v))
	case int8:
		return C.EnvAddLong(do.env.env, C.longlong(v))
	case int16:
		return C.EnvAddLong(do.env.env, C.longlong(v))
	case int32:
		return C.EnvAddLong(do.env.env, C.longlong(v))
	case int64:
		return C.EnvAddLong(do.env.env, C.longlong(v))
	case float32:
		return C.EnvAddDouble(do.env.env, C.double(v))
	case float64:
		return C.EnvAddDouble(do.env.env, C.double(v))
	case unsafe.Pointer:
		return C.EnvAddExternalAddress(do.env.env, v, C.C_POINTER_EXTERNAL_ADDRESS)
	case string:
		vstr := C.CString(v)
		defer C.free(unsafe.Pointer(vstr))
		return C.EnvAddSymbol(do.env.env, vstr)
	case Symbol:
		vstr := C.CString(string(v))
		defer C.free(unsafe.Pointer(vstr))
		return C.EnvAddSymbol(do.env.env, vstr)
	case InstanceName:
		vstr := C.CString(string(v))
		defer C.free(unsafe.Pointer(vstr))
		return C.EnvAddSymbol(do.env.env, vstr)
	case []interface{}:
		return do.listToMultifield(v)
	case *ImpliedFact:
		return v.factptr
	case *TemplateFact:
		return v.factptr
	case *Instance:
		return v.instptr
	}
	if reflect.TypeOf(dvalue).Kind() == reflect.Slice || reflect.TypeOf(dvalue).Kind() == reflect.Array {
		s := reflect.ValueOf(dvalue)
		mvalue := make([]interface{}, s.Len())
		for i := 0; i < s.Len(); i++ {
			mvalue[i] = s.Index(i).Interface()
		}
		return do.listToMultifield(mvalue)
	}
	// Fall back to FALSE in typical CLIPS style
	vstr := C.CString("FALSE")
	defer C.free(unsafe.Pointer(vstr))
	return C.EnvAddSymbol(do.env.env, vstr)
}

func (do *DataObject) multifieldToList() []interface{} {
	end := C.get_data_end(do.data)
	begin := C.get_data_begin(do.data)
	multifield := C.multifield_ptr(C.get_data_value(do.data))

	ret := make([]interface{}, 0, end-begin+1)
	for i := begin; i <= end; i++ {
		dtype := Type(C.get_multifield_type(multifield, i))
		dvalue := C.get_multifield_value(multifield, i)
		ret = append(ret, do.goValue(dtype, dvalue))
	}
	return ret
}

func (do *DataObject) listToMultifield(values []interface{}) unsafe.Pointer {
	size := C.long(len(values))
	ret := C.EnvCreateMultifield(do.env.env, size)
	multifield := C.multifield_ptr(ret)
	for i, v := range values {
		C.set_multifield_type(multifield, C.long(i+1), C.short(do.clipsTypeFor(v)))
		C.set_multifield_value(multifield, C.long(i+1), do.clipsValue(v))
	}
	C.set_data_begin(do.data, 1)
	C.set_data_end(do.data, size)
	return ret
}

// ExtractValue attempts to put the represented data value into the item provided by the user.
func (do *DataObject) ExtractValue(retval interface{}, extractClasses bool) error {
	directType := reflect.TypeOf(retval)
	if directType.Kind() != reflect.Ptr {
		return fmt.Errorf("retval must be a pointer to the value to be filled in")
	}
	val := do.Value()
	return do.env.convertArg(reflect.ValueOf(retval), reflect.ValueOf(val), extractClasses)
}

// MustExtractValue attempts to put the represented data value into the item provided by the user, and panics if it can't
func (do *DataObject) MustExtractValue(retval interface{}, extractClasses bool) {
	if err := do.ExtractValue(retval, extractClasses); err != nil {
		panic(err)
	}
}

func safeIndirect(output reflect.Value) reflect.Value {
	val := reflect.Indirect(output)
	if !val.IsValid() {
		val = reflect.New(output.Type().Elem())
		output.Set(val)
		val = val.Elem()
	}
	return val
}
func (env *Environment) convertArg(output reflect.Value, data reflect.Value, extractClasses bool) error {
	val := safeIndirect(output)

	if extractClasses {
		dif := data.Interface()
		var subinst *Instance
		var err error
		instname, ok := dif.(InstanceName)
		if ok {
			if instname == "nil" {
				// it's not an instance we can look up, it's just nil
				subinst = nil
			} else {
				subinst, err = env.FindInstance(instname, "")
				if err != nil {
					return err
				}
			}
		} else {
			subinst, ok = dif.(*Instance)
		}
		if ok {
			if subinst == nil {
				data = reflect.ValueOf(nil)
			} else {
				// extract the instance
				return subinst.Extract(val.Addr().Interface())
			}
		}
	}

	if !data.IsValid() {
		switch val.Kind() {
		case reflect.Ptr, reflect.Interface:
			val.Set(reflect.Zero(val.Type()))
			return nil
		default:
			return fmt.Errorf("Unable to convert nil value to non-pointer type %v", val.Type())
		}
	}

	checktype := val.Type()
	if val.Kind() == reflect.Ptr && data.Kind() != reflect.Ptr {
		checktype = checktype.Elem()
	}

	if data.Type().AssignableTo(checktype) {
		if val.Kind() == reflect.Ptr && data.Kind() != reflect.Ptr {
			val = safeIndirect(val)
		}
		val.Set(data)
		return nil
	}

	if data.Kind() == reflect.Int64 {
		// Make an exception when it's just loss of scale, and make it work
		val = safeIndirect(val)
		intval := data.Int()
		var checkval int64
		switch val.Type().Kind() {
		case reflect.Int64:
			// no check needed
		case reflect.Int:
			checkval = int64(int(intval))
		case reflect.Int32:
			checkval = int64(int32(intval))
		case reflect.Int16:
			checkval = int64(int16(intval))
		case reflect.Int8:
			checkval = int64(int8(intval))
		case reflect.Uint:
			checkval = int64(uint(intval))
		case reflect.Uint64:
			checkval = int64(uint64(intval))
		case reflect.Uint32:
			checkval = int64(uint32(intval))
		case reflect.Uint16:
			checkval = int64(uint16(intval))
		case reflect.Uint8:
			checkval = int64(uint8(intval))
		default:
			return fmt.Errorf(`Invalid type "%v", expected "%v"`, data.Type(), val.Type())
		}
		if checkval != intval {
			return fmt.Errorf(`Integer %d too large`, intval)
		}
		val.SetInt(checkval)
		return nil
	} else if data.Kind() == reflect.Float64 {
		val = safeIndirect(val)
		floatval := data.Float()
		switch val.Type().Kind() {
		case reflect.Float64:
			// no check needed
		case reflect.Float32:
			if float64(float32(floatval)) != floatval {
				return fmt.Errorf(`Floating point %f too precise to represent`, floatval)
			}
		default:
			return fmt.Errorf(`Invalid type "%v", expected "%v"`, data.Type(), val.Type())
		}
		val.SetFloat(floatval)
		return nil
	} else if data.Kind() == reflect.Slice {
		sliceval := val
		slicetype := val.Type()
		var mustSet bool
		if slicetype.Kind() == reflect.Ptr {
			// if we were handed a pointer, make sure it points to something and then set that thing
			sliceval = sliceval.Elem()
			slicetype = slicetype.Elem()
		}
		if slicetype.Kind() != reflect.Slice {
			return fmt.Errorf(`Invalid type "%v", expected "%v"`, data.Type(), val.Type())
		}
		if !sliceval.IsValid() || sliceval.Cap() < data.Len() {
			mustSet = true
			sliceval = reflect.MakeSlice(reflect.SliceOf(slicetype.Elem()), data.Len(), data.Len())
		}
		// see if we can translate to right kind of slice
		for i := 0; i < data.Len(); i++ {
			// what we get from CLIPS is always []interface{}, so there's no check for that here
			indexval := data.Index(i).Elem()
			if err := env.convertArg(sliceval.Index(i), indexval, extractClasses); err != nil {
				return err
			}
		}
		if data.Len() > 0 && mustSet {
			if val.Kind() == reflect.Ptr {
				val.Set(reflect.New(sliceval.Type()))
				val.Elem().Set(sliceval)
			} else {
				val.Set(sliceval)
			}
		}
		return nil
	} else if data.Type().ConvertibleTo(checktype) {
		if val.Kind() == reflect.Ptr && data.Kind() != reflect.Ptr {
			val = safeIndirect(val)
		}
		// This could actually handle ints and floats, too, except it hides wraparound and loss of precision
		converted := data.Convert(val.Type())
		val.Set(converted)
		return nil
	}
	return fmt.Errorf(`Invalid type "%v", expected "%v"`, data.Type(), val.Type())
}
