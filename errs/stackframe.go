package errs

import (
	"runtime"
	"strings"
)

// StackFrame describes a single captured call frame.
type StackFrame struct {
	// File is the absolute file path where the frame was captured.
	File string
	// LineNo is the line number within the File.
	LineNo int
	// Name is the function name without the package path.
	Name string
	// Package is the full package path for the function.
	Package string
	// ProgramCounter is the raw program counter for the frame.
	ProgramCounter uintptr
}

// NewStackFrame builds a StackFrame from a program counter value.
func NewStackFrame(pc uintptr) *StackFrame {
	fn := runtime.FuncForPC(pc)
	fnFileName, fnLineNo := fn.FileLine(pc)
	fnName := fn.Name()
	fnPkgName := ""
	if lastSlash := strings.LastIndex(fnName, "/"); lastSlash >= 0 {
		fnPkgName += fnName[:lastSlash] + "/"
		fnName = fnName[lastSlash+1:]
	}
	if period := strings.Index(fnName, "."); period >= 0 {
		fnPkgName += fnName[:period]
		fnName = fnName[period+1:]
	}

	fnName = strings.ReplaceAll(fnName, "·", ".")

	return &StackFrame{File: fnFileName, LineNo: fnLineNo, Name: fnName, Package: fnPkgName, ProgramCounter: pc}
}
