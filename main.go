package abi

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	fuzz "github.com/google/gofuzz"
)

func unpackPack(abi abi.ABI, method string, inputType []interface{}, input []byte) bool {
	outptr := reflect.New(reflect.TypeOf(inputType))
	if err := abi.Unpack(outptr.Interface(), method, input); err == nil {
		output, err := abi.Pack(method, input)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(input, output) {
			panic(fmt.Sprintf("unpackPack is not equal, \ninput : %x\noutput: %x", input, output))
		}
		return true
	}
	return false
}

func packUnpack(abi abi.ABI, method string, input []interface{}) bool {
	if packed, err := abi.Pack(method, input); err == nil {
		outptr := reflect.New(reflect.TypeOf(input))
		err := abi.Unpack(outptr.Interface(), method, packed)
		if err != nil {
			panic(err)
		}
		out := outptr.Elem().Interface()
		if !reflect.DeepEqual(input, out) {
			panic(fmt.Sprintf("unpackPack is not equal, \ninput : %x\noutput: %x", input, out))
		}
		return true
	}
	return false
}

type args struct {
	name string
	typ  string
}

func createABI(name string, stateMutability, payable *string, inputs []args) (abi.ABI, error) {
	sig := fmt.Sprintf(`[{ "type" : "function", "name" : "%v" `, name)
	if stateMutability != nil {
		sig += fmt.Sprintf(`, "stateMutability": "%v" `, *stateMutability)
	}
	if payable != nil {
		sig += fmt.Sprintf(`, "payable": %v `, *payable)
	}
	if len(inputs) > 0 {
		sig += fmt.Sprintf(`, "inputs" : [ {`)
		for i, inp := range inputs {
			sig += fmt.Sprintf(`"name" : "%v", "type" : "%v" `, inp.name, inp.typ)
			if i+1 < len(inputs) {
				sig += ","
			}
		}
		sig += "} ]"
		sig += fmt.Sprintf(`, "outputs" : [ {`)
		for i, inp := range inputs {
			sig += fmt.Sprintf(`"name" : "%v", "type" : "%v" `, inp.name, inp.typ)
			if i+1 < len(inputs) {
				sig += ","
			}
		}
		sig += "} ]"
	}
	sig += `}]`

	abi, err := abi.JSON(strings.NewReader(sig))
	if err != nil {
		//panic(fmt.Sprintf("err: %v, abi: %v", err.Error(), sig))
	}
	return abi, err
}

func fillStruct(structs []interface{}, data []byte) {
	if structs != nil && len(data) != 0 {
		fuzz.NewFromGoFuzz(data).Fuzz(&structs)
	}
}

func createStructs(args []args) []interface{} {
	structs := make([]interface{}, len(args))
	for i, arg := range args {
		t, err := abi.NewType(arg.typ, "", nil)
		if err != nil {
			panic(err)
		}
		structs[i] = reflect.New(t.GetType()).Elem()
	}
	return structs
}

func runFuzzer(input []byte) int {
	good := false

	names := []string{"", "_name", "name", "NAME", "name_", "__", "_name_", "n"}
	stateMut := []string{"", "pure", "view", "payable"}
	stateMutabilites := []*string{nil, &stateMut[0], &stateMut[1], &stateMut[2], &stateMut[3]}
	pays := []string{"true", "false"}
	payables := []*string{nil, &pays[0], &pays[1]}
	varNames := []string{"a", "b", "c", "d", "e", "f", "g"}
	varNames = append(varNames, names...)
	varTypes := []string{"bool", "address", "bytes", "string",
		"uint", "int", "uint8", "int8", "uint8", "int8", "uint16", "int16",
		"uint24", "int24", "uint32", "int32", "uint40", "int40", "uint48", "int48", "uint56", "int56",
		"uint64", "int64", "uint72", "int72", "uint80", "int80", "uint88", "int88", "uint96", "int96",
		"uint104", "int104", "uint112", "int112", "uint120", "int120", "uint128", "int128", "uint136", "int136",
		"uint144", "int144", "uint152", "int152", "uint160", "int160", "uint168", "int168", "uint176", "int176",
		"uint184", "int184", "uint192", "int192", "uint200", "int200", "uint208", "int208", "uint216", "int216",
		"uint224", "int224", "uint232", "int232", "uint240", "int240", "uint248", "int248", "uint256", "int256",
		"bytes1", "bytes2", "bytes3", "bytes4", "bytes5", "bytes6", "bytes7", "bytes8", "bytes9", "bytes10", "bytes11",
		"bytes12", "bytes13", "bytes14", "bytes15", "bytes16", "bytes17", "bytes18", "bytes19", "bytes20", "bytes21",
		"bytes22", "bytes23", "bytes24", "bytes25", "bytes26", "bytes27", "bytes28", "bytes29", "bytes30", "bytes31",
		"byte32", "byte"}
	rnd := rand.New(rand.NewSource(123456))
	if len(input) > 0 {
		rnd = rand.New(rand.NewSource(int64(input[0])))
	}
	for _, name := range names {
		for _, stateMut := range stateMutabilites {
			for _, payable := range payables {
				var arg []args
				for i := rnd.Int31n(5); i > 0; i-- {
					argName := varNames[rnd.Int31n(int32(len(varNames)))]
					argTyp := varTypes[rnd.Int31n(int32(len(varTypes)))]
					if rnd.Int31n(10) == 0 {
						argTyp += "[]"
					} else if rnd.Int31n(10) == 0 {
						arrayArgs := rnd.Int31n(30)
						argTyp += fmt.Sprintf("[%d]", arrayArgs)
					}
					arg = append(arg, args{
						name: argName,
						typ:  argTyp,
					})
				}
				abi, err := createABI(name, stateMut, payable, arg)
				if err != nil {
					continue
				}
				structs := createStructs(arg)
				b := unpackPack(abi, name, structs, input)
				fillStruct(structs, input)
				c := packUnpack(abi, name, structs)
				good = good || b || c
			}
		}
	}
	if good {
		return 1
	}
	return 0
}

func Fuzz(input []byte) int {
	return runFuzzer(input)
}
