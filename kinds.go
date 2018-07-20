package main

import "reflect"

var Uints = []reflect.Kind{
	reflect.Uint,
	reflect.Uint8,
	reflect.Uint16,
	reflect.Uint32,
	reflect.Uint64,
}

var Ints = []reflect.Kind{
	reflect.Int,
	reflect.Int8,
	reflect.Int16,
	reflect.Int32,
	reflect.Int64,
}

var Floats = []reflect.Kind {
	reflect.Float32,
	reflect.Float64,
}

var Builtins = append(append(append(Uints, Ints...), Floats...), reflect.Bool)
