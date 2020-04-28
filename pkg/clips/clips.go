package clips

// #cgo CFLAGS: -I ../../clips_source
// #cgo LDFLAGS: -L ../../clips_source -l clips
// #include <clips.h>
//
// void goFunction(void *env, DATA_OBJECT *data);
//
// static inline void callGoFunction(void * env, DATA_OBJECT *data) {
//	 goFunction(env,data);
// }
//
// int define_function(void *environment)
// {
//     return EnvDefineFunction(
//         environment, "go-function", 'u',
//         PTIEF callGoFunction, "go-function");
// }
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

const defFunction = `
(deffunction %[1]s ($?args)
  (go-function %[1]s (expand$ ?args)))
`

// Error error returned from CLIPS
type Error struct {
	Err error
}

// Callback is the signature for functions that will be called from CLIPS
type Callback func([]interface{}) (interface{}, error)

// Environment stores a CLIPS environment
type Environment struct {
	env      unsafe.Pointer
	callback map[string]Callback
}

var environmentObj = make(map[unsafe.Pointer]*Environment)

func (e *Error) Error() string {
	return e.Err.Error()
}

// CreateEnvironment creates a new instance of a CLIPS environment
func CreateEnvironment() *Environment {
	ret := &Environment{
		env:      C.CreateEnvironment(),
		callback: make(map[string]Callback),
	}
	runtime.SetFinalizer(ret, func(env *Environment) {
		env.Close()
	})
	C.define_function(ret.env)
	environmentObj[ret.env] = ret

	return ret
}

// Close destroys the CLIPS environment
func (env *Environment) Close() {
	if env.env != nil {
		delete(environmentObj, env.env)
		C.DestroyEnvironment(env.env)
		env.env = nil
	}
}

// Load loads a set of constructs into the CLIPS data base. Constructs can be in text or binary format. Equivalent to CLIPS (load)
func (env *Environment) Load(path string) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	errint := int(C.EnvBload(env.env, cpath))
	if errint != 1 {
		errint = int(C.EnvLoad(env.env, cpath))
	}
	if errint != 1 {
		return &Error{
			Err: fmt.Errorf("Unable to load file %s", path),
		}
	}
	return nil
}

// Save saves the current state of the environment
func (env *Environment) Save(path string, binary bool) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var errint int
	if binary {
		errint = int(C.EnvBsave(env.env, cpath))
	} else {
		errint = int(C.EnvSave(env.env, cpath))
	}
	if errint != 1 {
		return &Error{
			Err: fmt.Errorf("Unable to save to file %s", path),
		}
	}
	return nil
}

// BatchStar executes the CLIPS code found in path. Equivalent to CLIPS (batch*)
func (env *Environment) BatchStar(path string) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	if C.EnvBatchStar(env.env, cpath) != 1 {
		return &Error{
			Err: fmt.Errorf("Unable to open file %s", path),
		}
	}
	return nil
}

// Build builds a single construct within the CLIPS environment
func (env *Environment) Build(construct string) error {
	cconstruct := C.CString(construct)
	defer C.free(unsafe.Pointer(cconstruct))
	if C.EnvBuild(env.env, cconstruct) != 1 {
		return &Error{
			Err: fmt.Errorf("Unable to parse construct %s", construct),
		}
	}
	return nil
}

// Eval evaluates an expression returning its value
func (env *Environment) Eval(construct string) (interface{}, error) {
	cconstruct := C.CString(construct)
	defer C.free(unsafe.Pointer(cconstruct))

	data := createDataObject(env)
	errint := int(C.EnvEval(env.env, cconstruct, data.byRef()))

	if errint != 1 {
		return nil, &Error{
			Err: fmt.Errorf("Unable to parse construct %s", construct),
		}
	}
	return data.Value(), nil
}

// Reset resets the CLIPS environment
func (env *Environment) Reset() {
	C.EnvReset(env.env)
}

// Clear clears the CLIPS environment
func (env *Environment) Clear() {
	C.EnvClear(env.env)
}

// DefineFunction defines a Go function within the CLIPS environment
func (env *Environment) DefineFunction(name string, callback Callback) error {
	env.callback[name] = callback
	return env.Build(fmt.Sprintf(defFunction, name))
}
