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

)

const (
	ReplaceRegexFmt = `(%s)([^a-z])`
)

type setItem struct {}
var item = setItem{}

func ParseVal(str string) []string {
	vals := strings.Split(str, ",")
	set := map[string]struct{}{}

	L:
	for _, v := range vals {
		switch v {
		case "":
			continue L
		case LowerCaseBuiltin:
			for _, v := range Builtins {
				set[v.String()] = item
			}
		case LowerCaseInts:
			for _, v := range Ints {
				set[v.String()] = item
			}
		case LowerCaseIntN:
			for _, v := range Ints[1:] {
				set[v.String()] = item
			}
		case LowerCaseUints:
			for _, v := range Uints {
				set[v.String()] = item
			}
		case LowerCaseUintN:
			for _, v := range Uints[1:] {
				set[v.String()] = item
			}
		case UpperCaseBuiltin:
			for _, v := range Builtins {
				set[strings.Title(v.String())] = item
			}
		case UpperCaseInts:
			for _, v := range Ints {
				set[strings.Title(v.String())] = item
			}
		case UpperCaseIntN:
			for _, v := range Ints[1:] {
				set[strings.Title(v.String())] = item
			}
		case UpperCaseUints:
			for _, v := range Uints {
				set[strings.Title(v.String())] = item
			}
		case UpperCaseUintN:
			for _, v := range Uints[1:] {
				set[strings.Title(v.String())] = item
			}
		default:
			set[v] = item
		}
	}

	ret := make([]string, len(set))
	i := 0
	for k := range set {
		ret[i] = k
		i++
	}

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

func PutVars(str string, vars map[string][]string, meths map[string]Method) error {
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
		} else {
			vars[key] = ParseVal(keyVal[1])
		}
	}

	return nil
}

func ReplaceMeths(str string, meths map[string]Method, state map[string]string) string {

	for k, v := range meths {
		replace := ExecMethod(state[v.Receiver], v.Name)
		str = regexp.
			MustCompile(fmt.Sprintf(ReplaceRegexFmt, k)).
			ReplaceAllString(str, replace+"$2")
	}

	return str
}

func ReplaceBlock(str string, vars map[string][]string, meths map[string]Method, state map[string]string, out io.Writer) {
	fmt.Println("ReplaceBlock\n", str, "\n", vars)

	if len(vars) == 0 {
		str = ReplaceMeths(str, meths, state)

		out.Write([]byte(str))
		return
	}

	for k, v := range vars {
		delete(vars, k)
		if v[0] == "." {

		}
		for _, vv := range v {
			newStr := regexp.
				MustCompile(fmt.Sprintf(ReplaceRegexFmt, k)).
				ReplaceAllString(str, vv+"$2")
			state[k] = vv
			ReplaceBlock(newStr, vars, meths, state, out)
		}
	}
}

func DoReplace(s *bufio.Scanner, out io.Writer) (n int, err error) {
	vars := map[string][]string{}
	meths := map[string]Method{}
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
			ReplaceBlock(block.String(), vars, meths, map[string]string{}, out)
			block.Reset()
			fmt.Println("clear:\n", s.Text())
			out.Write(append(s.Bytes(), '\n'))
		} else if ProtoPragma.MatchString(line) {
			ReplaceBlock(block.String(), vars, meths, map[string]string{}, out)
			block.Reset()

			err = PutVars(line, vars, meths)
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

	ReplaceBlock(block.String(), vars, meths, map[string]string{}, out)

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
	if strings.HasSuffix(name, ".go") {
		return strings.Replace(name, ".go", "_gen.go", 1)
	} else if strings.Contains(name, ".") {
		return name + ".gen.go"
	} else {
		return name + "_gen.go"
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
