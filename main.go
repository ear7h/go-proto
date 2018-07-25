package main

import (
	"os"
	"strings"
	"path/filepath"
	"log"
	"io"
	"bufio"
	"errors"
	"regexp"
	"fmt"
)

var BuildProto = regexp.MustCompile(`//\s*\+build\s*proto`)
var ProtoPragmaIgnore = regexp.MustCompile(`//\s*go:proto\s*ignore`)
var ProtoPragmaClear= regexp.MustCompile(`//\s*go:proto\s*clear`)
var ProtoPragma = regexp.MustCompile(`//\s*go:proto\s`)
var Caps = regexp.MustCompile(`^[A-Z]\w*`)

const (
	UpperCaseBuiltin = "Builtin"
	LowerCaseBuiltin = "bultin"

	UpperCaseUints = "Uints"
	LowerCaseUints = "uints"
	UpperCaseUintN = "UintN"
	LowerCaseUintN = "uintN"

	UpperCaseInts = "Ints"
	LowerCaseInts = "ints"
	UpperCaseIntN = "IntN"
	LowerCaseIntN = "intN"

	UpperCaseFloats = "Floats"
	LowerCaseFloats = "floats"
)

const (
	ReplaceRegexFmt = `(%s)([^a-z])`
)

type setItem struct {}
var item = setItem{}

func ParseVal(str string) []string {
	vals := strings.Split(str, ",")
	expanded := []string{}
	set := map[string]struct{}{}
	add := func(s string) {
		fmt.Println("add: ", s)
		set[s] = item
		expanded = append(expanded, s)
		fmt.Println(set, expanded)
	}

	L:
	for _, v := range vals {
		switch v {
		case "":
			continue L
		case LowerCaseBuiltin:
			for _, v := range Builtins {
				add(v.String())
			}
		case LowerCaseInts:
			for _, v := range Ints {
				add(v.String())
			}
		case LowerCaseIntN:
			for _, v := range Ints[1:] {
				add(v.String())
			}
		case LowerCaseUints:
			for _, v := range Uints {
				add(v.String())
			}
		case LowerCaseUintN:
			for _, v := range Uints[1:] {
				add(v.String())
			}
		case LowerCaseFloats:
			for _, v := range Floats {
				add(v.String())
			}
		case UpperCaseBuiltin:
			for _, v := range Builtins {
				add(strings.Title(v.String()))
			}
		case UpperCaseInts:
			for _, v := range Ints {
				add(strings.Title(v.String()))
			}
		case UpperCaseIntN:
			for _, v := range Ints[1:] {
				add(strings.Title(v.String()))
			}
		case UpperCaseUints:
			for _, v := range Uints {
				add(strings.Title(v.String()))
			}
		case UpperCaseUintN:
			for _, v := range Uints[1:] {
				add(strings.Title(v.String()))
			}
		case UpperCaseFloats:
			for _, v := range Floats {
				add(strings.Title(v.String()))
			}
		default:
			add(v)
		}
	}

	ret := make([]string, len(set))
	i := 0
	// ensure predictable order
	for _,v := range expanded {
		if _, ok := set[v];ok {
			ret[i] = v
			delete(set, v)
			i++
		}
	}

	fmt.Println("vals: ", ret)

	return ret
}

const (
	MethodLower = "lower"
	MethodCapital = "capital"
	MethodSizeBits = "sizebits"
	MethodSizeBytes = "sizebytes"
)

type Method struct {
	Receiver, Name string

}

// methods array
// []string{ ".", "receiver", "method"}
// the functions and method are evaluated from right to left
func ParseMethod(valStr string) (Method, error) {
	arr := strings.Split(valStr, ".")
	if len(arr) != 2{
		return Method{}, errors.New("methods must be of the form reciver.method")
	}
	return Method{Receiver: arr[0], Name: arr[1]}, nil
}

func PutVars(str string, vars map[string][]string, varOrder *[]string, meths map[string]Method, methsOrder *[]string) error {
	fmt.Println("---putvar--\n\n", str)
	defer fmt.Println("\n\n---putvar--")
	for _, v := range strings.Split(str, " ") {
		fmt.Println(v)
		if !strings.Contains(v, "=") {
			continue
		}

		keyVal := strings.Split(v, "=")
		if len(keyVal) != 2 {
			return errors.New("invalid key value formations")
		}

		key := Caps.FindString(keyVal[0])
		if key != keyVal[0] {
			return errors.New("bad key")
		}

		val := keyVal[1]
		if val == "/" {
			delete(vars, key)
			delete(meths, key)
		} else if strings.Contains(val, ".") {
			// T=intN T1=T.sizeof
			var err error
			meths[key], err = ParseMethod(val)
			if err != nil {
				return err
			}
			*methsOrder = append(*methsOrder, key)
		} else {
			vars[key] = ParseVal(keyVal[1])
			*varOrder = append(*varOrder, key)
		}
	}

	return nil
}

func ReplaceMeths(str string, meths map[string]Method, methsOrder []string, state map[string]string) string {

	for _, k := range methsOrder {
		v := meths[k]
		replace := ExecMethod(state[v.Receiver], v.Name)
		str = regexp.
			MustCompile(fmt.Sprintf(ReplaceRegexFmt, k)).
			ReplaceAllString(str, replace+"$2")
	}

	return str
}

func ReplaceBlock(str string, vars map[string][]string, varOrder []string, meths map[string]Method, methsOrder []string, state map[string]string, out io.Writer) {
	fmt.Println("ReplaceBlock\n", str, "\nvars:", vars, "\norder:", varOrder)

	if len(vars) == 0 {
		str = ReplaceMeths(str, meths, methsOrder, state)

		out.Write([]byte(str))
		return
	}

	for _, k := range varOrder {
		v := vars[k]
		newOrder := varOrder[1:] // queue 80s music .. .. ....
		delete(vars, k)
		if v[0] == "." {

		}
		for _, vv := range v {
			newStr := regexp.
				MustCompile(fmt.Sprintf(ReplaceRegexFmt, k)).
				ReplaceAllString(str, vv+"$2")
			state[k] = vv
			ReplaceBlock(newStr, vars, newOrder, meths, methsOrder, state, out)
		}
	}
}

func DoReplace(s *bufio.Scanner, out io.Writer) (n int, err error) {
	vars := map[string][]string{}
	varOrder := []string{}
	meths := map[string]Method{}
	methsOrder := []string{}
	block := &strings.Builder{}
	for s.Scan() {
		n++
		//out.Write(s.Bytes())

		line := s.Text()
		if ProtoPragmaIgnore.MatchString(line) {
			if !s.Scan() {
				break
			}
			fmt.Println("ignore:\n", s.Text())
		} else if ProtoPragmaClear.MatchString(line) {
			ReplaceBlock(block.String(), vars, varOrder, meths, methsOrder, map[string]string{}, out)
			block.Reset()
			fmt.Println("clear:\n", s.Text())
			out.Write(append(s.Bytes(), '\n'))

			//clear
			vars = map[string][]string{}
			varOrder = []string{}
			meths = map[string]Method{}
		} else if ProtoPragma.MatchString(line) {
			ReplaceBlock(block.String(), vars, varOrder, meths, methsOrder, map[string]string{}, out)
			block.Reset()

			err = PutVars(line, vars, &varOrder, meths, &methsOrder)
			if err != nil {
				return n, err
			}

			fmt.Println("---vars--\n\n", vars, "\n\n---")

			out.Write(append(s.Bytes(), '\n'))
		} else {
			block.Write(append(s.Bytes(), '\n'))
		}
	}

	if err := s.Err(); err != nil {
		return n, err
	}

	ReplaceBlock(block.String(), vars, varOrder, meths, methsOrder, map[string]string{}, out)

	return n, nil
}

func CheckBuildProto(s *bufio.Scanner) error {
	if !s.Scan() {
		return errors.New("no text")
	}
	if err := s.Err(); err != nil {
		return err
	}

	if !BuildProto.MatchString(s.Text()) {
		return errors.New("no proto header")
	}

	return nil
}

func GenName(name string)string {
	if strings.HasSuffix(name, "_test_proto.go") {
		return strings.Replace(name, "_test_proto.go", "_gen_test.go", 1)
	} else if strings.HasSuffix(name, "_proto.go") {
		return strings.Replace(name, "_proto.go", "_gen.go", 1)
	} else if strings.HasSuffix(name, ".go") {
		return strings.Replace(name, ".go", "_gen.go", 1)
	} else {
		return name + ".gen.go"
	}
}

func main() {
	log.SetOutput(os.Stdout)

	var glob string
	if len(os.Args) == 1 {
		glob = "*_proto.go"
	} else if os.Args[1] == "." {
		glob = "*_proto.go"
	} else {
		glob = os.Args[1]
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	glob = filepath.Join(wd, glob)
	fmt.Println("glob: ", glob)

	files, err := filepath.Glob(glob)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("files", files)

	for _, file := range files {
		fd, err := os.OpenFile(file, os.O_RDWR, 0600)
		if err != nil {
			log.Fatal(err)
		}
		defer fd.Close()

		s := bufio.NewScanner(fd)

		err = CheckBuildProto(s)
		if err != nil {
			log.Fatalf("%v: %v", file, err)
		}

		out, err := os.OpenFile(GenName(file), os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0666)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()


		n, err := DoReplace(s, out)
		if err != nil {
			log.Fatalf("%v (%d): %v", file, n+1, err)
		}
	}
}
