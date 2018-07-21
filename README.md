# go-proto

## example

The following file is an example prototype:
__key elements:__
* `//+build proto`
    * this is required, will keep the file from being built in a binary
    * also allows for running `go-proto` with a `*.go` glob and other files will not be touched
* `//go:proto ignore`
    * this will tell the generator to ignore the next line
* `//go:proto T=Builtin`
    * This is a type variable definition. Code will be generated such that `T` is ![sensibly](#variable-replace) replaced with all builtin type names capitalized (excluding complex)
        * the variable name must start with a capital and can only contain alphanumerics and underscore"
        * `T` can be used in tokens/idetifiers (ie function names) as long as they are properly camel cased
            * `func TAdd(T) T` 
        * other groups: `kinds`, `uints`, `Uints`, `ints`, `Ints`
        * variations of ints/uints also have `*intN` for fixed size integers
    * Also supported is a comma delimited list of arbitrary names
        * `//go:proto T=*os.File,*bytes.Reader,*strings.Reader`
    * While untested, I belive multiple variables are supported
        * syntax: `//go:proto T1=uints,ints T2=uints,ints`
        * This would create a sort of matrix and code would be generated
* `//go:proto T:/`
    * the generator is stateful, and this pragma will remove the previously set value from the variable `T` 

run `go-proto` or `go-proto .` or `go-proto types_pointers_proto.go`

types_pointers_proto.go:
```go
//+build proto

package main

//go:proto ignore
type T Int

//go:proto T=Builtin
type TPointer struct {
	addr uint
	a Allocator
}

func (p TPointer) uint() Uint {
	return Uint{addr: p.addr, a: p.a}
}

func (p TPointer) Addr() uint {
	return p.addr
}

func (p TPointer) SizeOf() uint {
	return Uint64Size
}

func (p TPointer) Get() (uint, error) {
	return p.uint().Get()
}

func (p TPointer) Set(val T) error {
	return p.uint().Set(val.addr)
}
//go:proto T:/
````

## variable replace
variable names are replaced via the following regex:
```regexp
(T)([^a-z])
```
where "T" is replaced by the variable name