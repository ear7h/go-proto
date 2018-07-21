package main

import (
	"strings"
	"strconv"
)

func ExecMethod(receiverVal, method string) string {
	switch method {
	case MethodCapital:
		return strings.Title(receiverVal)
	case MethodLower:
		return strings.ToLower(receiverVal)
	case MethodSizeBits:
		return SizeBits(receiverVal)
	case MethodSizeBytes:
		return SizeBytes(receiverVal)
	default:
		return "<< method not found >>"
	}
}

func SizeBits(receiverVal string) string {
	receiverVal = strings.ToLower(receiverVal)
	return strconv.Itoa(sizeBits(receiverVal))
}

func SizeBytes(receiverVal string) string {
	receiverVal = strings.ToLower(receiverVal)
	return strconv.Itoa(sizeBits(receiverVal)/8)
}

func sizeBits(t string) int {
	switch t {
	case "uint8", "int8", "byte", "bool":
		return 8
	case "uint16", "int16":
		return 16
	case "uint32", "int32", "float32":
		return 32
	case "uint64", "int64", "float64":
		return 64
	default:
		return -8
	}
}