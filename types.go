package resly

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// validator is used to optionally validate types that implement
// this interface before executing associated method (either query
// or mutation).
type validator interface {
	Validate() error
}

// TypeDef represents a Go type and list of accessors translated into
// GraphQL schema.
type TypeDef struct {
	// Name is a unique object name.
	Name string

	// Type is a corresponding Go type for a GraphQL type.
	Type reflect.Type

	// Funcs is a list of methods for this type.
	Funcs map[string]FuncDef
}

// ClosureDef represents anonymous closure function definition.
type ClosureDef func(funcdef FuncDef, in []reflect.Value) (out []reflect.Value)

// FuncDef represents a named method of the type definition that is
// translated to the attribute accessor of the GraphQL type.
type FuncDef struct {
	// Name is a name of the method, it should be unique within a
	// single type definition.
	Name string

	// Func is an associated function used to call the given method.
	Func reflect.Value
	Type reflect.Type

	// In is an input argument type of the function.
	In reflect.Type

	// Out is the type that function returns as the first return parameter.
	Out reflect.Type
}

func (fd FuncDef) call(in []reflect.Value) (interface{}, error) {
	out := fd.Func.Call(in)
	res, err := out[0].Interface(), out[1].Interface()
	if err != nil {
		return res, err.(error)
	}
	return res, nil
}

func (fd FuncDef) Call(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	in := []reflect.Value{reflect.ValueOf(ctx)}
	if fd.In != nil {
		var (
			inValue     = reflect.New(fd.In)
			inInterface = inValue.Interface()
		)
		if err := jsonUnpack(args, inInterface); err != nil {
			return nil, err
		}
		if v, ok := inInterface.(validator); ok {
			if err := v.Validate(); err != nil {
				return nil, err
			}
		}
		in = append(in, inValue.Elem())
	}

	return fd.call(in)
}

func (fd FuncDef) CallBound(ctx context.Context, source interface{}) (interface{}, error) {
	return fd.call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(source)})
}

var errorType = reflect.TypeOf(make([]error, 1)).Elem()

// EncloseFunc overrides function of the function definition with a new
// function "clouse". Closure accepts as the first argument original function.
func EncloseFunc(funcdef FuncDef, closures ...ClosureDef) FuncDef {
	for _, closure := range closures {
		closure := closure
		originalFuncDef := funcdef
		funcdef.Func = reflect.MakeFunc(funcdef.Type, func(in []reflect.Value) []reflect.Value {
			out := closure(originalFuncDef, in)
			if len(out) == 1 {
				return out
			}

			errValue := out[1]
			if !out[1].IsValid() {
				errValue = reflect.Zero(errorType)
			}
			out = []reflect.Value{
				out[0],
				errValue,
			}
			return out
		})
	}
	return funcdef
}

// DefineFunc returns a new function definition that should comply
// one of the following two signatures:
//
//	func Variant1(ctx context.Context) (interface{}, error) {
//		// ...
//	}
//
//	func Variant2(ctx context.Context, input TypeInput) (interface{}, error) {
//		// ...
//	}
//
// When specified function v does not comply given variants, method
// returns an error.
func DefineFunc(name string, v interface{}) (funcdef FuncDef, err error) {
	var (
		funcValue = reflect.ValueOf(v)
		funcType  = reflect.TypeOf(v)
	)

	// Ensure that the method is compatible with method definition that
	// we support, otherwise panic.
	if funcType.Kind() != reflect.Func {
		return funcdef, errors.Errorf("func %q must be a func, is %T", name, v)
	}

	var (
		in reflect.Type

		numIn  = funcType.NumIn()
		numOut = funcType.NumOut()

		errorInterface   = reflect.TypeOf((*error)(nil)).Elem()
		contextInterface = reflect.TypeOf((*context.Context)(nil)).Elem()
	)

	// Ensure that first argument implements context.Context interface,
	// all declared functions should handle context in order to gracefully
	// shutdown the server.
	//
	// The second argument can be omitted. When specified, it should be
	// a structure.
	if numIn < 1 {
		return funcdef, errors.Errorf(
			"Function %q must take at least 1 argument `context.Context`, takes %d.",
			name, numOut,
		)
	}
	if !funcType.In(0).Implements(contextInterface) {
		return funcdef, errors.Errorf(
			"The first argument of the function %q must implement "+
				"`context.Context` interface", name,
		)
	}
	if numIn == 2 {
		if in = funcType.In(1); in.Kind() != reflect.Struct {
			return funcdef, errors.Errorf(
				"The second argument of the function %q must be a struct", name)
		}
	}

	// Ensure that the second returned argument is an error, which will
	// be propagated to the client through GraphQL interface.
	if numOut != 2 {
		return funcdef, errors.Errorf(
			"Function %q must return exatly 2 argument, returns %d. "+
				"Both 'Mutation' and 'Query' require return parameter, "+
				"when this behavior must be preserved, consider using "+
				"`scalar Void` as return type.",
			name, numOut,
		)
	}
	if !funcType.Out(numOut - 1).Implements(errorInterface) {
		return funcdef, errors.Errorf("Second return arg of %q must implement error", name)
	}

	return FuncDef{
		Name: name,
		Func: funcValue,
		Type: funcType,
		In:   in,
		Out:  funcType.Out(0),
	}, nil
}

// NewFunc creates a new function definition, on error it panics.
//
// See documentation of DefineFunc for more details.
func NewFunc(name string, v interface{}) FuncDef {
	funcdef, err := DefineFunc(name, v)
	if err != nil {
		panic(err)
	}
	return funcdef
}

// Funcs defines a list of methods for the GraphQL type. Method is
// any synthetic method.
type Funcs map[string]interface{}

// DefineType returns a new type definition for the given structure.
//
// Method accepts optional methods for the type, all methods should
// accept context.Context instance as the first argument and parent
// type as the second argument.
//
// For example:
//
//	type Author struct {
//		Name    string
//		BookIDs []string
//	}
//
//	type Book struct {
//		ID    string
//		Title string
//	}
//
//	typedef, err := DefineType(Author{}, Funcs{
//		"books": func(ctx context.Context, a Author) ([]Book, error) {
//			// Fetch books here.
//		},
//	})
//
// Please, consider using value instead of pointer to the parent type.
func DefineType(v interface{}, funcs Funcs) (typedef TypeDef, err error) {
	gotype := reflect.TypeOf(v)

	if gotype.Kind() != reflect.Struct {
		return typedef, errors.Errorf("type must be a struct, is %T", v)
	}

	funcdefs := make(map[string]FuncDef)
	for name, v := range funcs {
		funcdefs[name], err = DefineFunc(name, v)
		if err != nil {
			return typedef, err
		}

		// Ensure that input argument of the method is the same as the
		// parent type.
		if funcdefs[name].In == nil {
			return typedef, errors.Errorf("func %q is missing parent argument", name)
		}
		if funcdefs[name].In != gotype {
			return typedef, errors.Errorf("parent type of %q does not match", name)
		}
	}

	return TypeDef{
		Name:  strings.ToLower(gotype.Name()),
		Type:  gotype,
		Funcs: funcdefs,
	}, nil
}

// NewType creates a new type definition, on error it panics.
//
// See documentation of DefineType for more details.
func NewType(v interface{}, methods Funcs) TypeDef {
	typedef, err := DefineType(v, methods)
	if err != nil {
		panic(err)
	}
	return typedef
}
