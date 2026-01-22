package errs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
)

const (
	errorHeader       = "ERROR: "
	defaultIndent     = "  "
	tagsPrefix        = "--- Tags: "
	stackFramePrefix  = "--- Frame: "
	sentinelSeparator = ": "
)

var (
	// Is reports whether any error in error's chain matches the target.
	// It is an alias for errors.Is.
	Is = errors.Is

	// As finds the first error in error's chain that matches the target, and if so,
	// sets the target to that error value. It is an alias for errors.As.
	As = errors.As
)

// Unwrap returns the wrapped error(s) from err, supporting both single and
// multi-error unwrap forms. It returns nil when err does not wrap anything.
func Unwrap(err error) []error {
	//nolint:errorlint // internal error handling
	switch x := err.(type) {
	case interface{ Unwrap() error }:
		return []error{x.Unwrap()}
	case interface{ Unwrap() []error }:
		return x.Unwrap()
	}

	return nil
}

// Error represents a typed error that can carry tags and a stack frame.
type Error interface {
	error
	fmt.Formatter

	// WithTag adds or overwrites a tag on the error.
	WithTag(string, any) Error
	// WithMessage appends or sets a formatted message on the error.
	WithMessage(format string, args ...any) Error
	// WithStackFrame captures and attaches a stack frame to the error.
	WithStackFrame() Error
	// Tags returns the tags attached to the error or nil when none exist.
	Tags() map[string]any
	// StackFrame returns the attached stack frame or nil when none exists.
	StackFrame() *StackFrame
}

// New creates a new sentinel error with a formatted message.
func New(format string, args ...any) Error {
	return &sentinelError{
		message: fmt.Sprintf(format, args...),
	}
}

// Join combines multiple errors into a single error with a shared stack frame.
// It returns nil when no errors are provided.
func Join(errs ...error) Error {
	if len(errs) == 0 {
		return nil
	}
	pc, _, _, _ := runtime.Caller(1)
	var children []error
	for _, e := range errs {
		//nolint:errorlint // internal error library
		if sentinelErr, ok := e.(*sentinelError); ok {
			children = append(children, &implError{
				message:    sentinelErr.message,
				wrapped:    []error{sentinelErr},
				stackFrame: nil,
				tags:       nil,
			})
		} else {
			children = append(children, e)
		}
	}

	return &implError{
		message:    "",
		wrapped:    children,
		stackFrame: NewStackFrame(pc),
		tags:       nil,
	}
}

func wrap(err error, i int) Error {
	pc, _, _, _ := runtime.Caller(1 + i)

	//nolint:errorlint // internal error library
	if sentinelErr, ok := err.(*sentinelError); ok {
		return &implError{
			message:    sentinelErr.message,
			wrapped:    []error{sentinelErr},
			stackFrame: NewStackFrame(pc),
			tags:       nil,
		}
	} else {
		return &implError{
			message:    "",
			wrapped:    []error{err},
			stackFrame: NewStackFrame(pc),
			tags:       nil,
		}
	}
}

// Wrap wraps err, capturing a stack frame.
func Wrap(err error) Error {
	return wrap(err, 0)
}

// HasTag searches the error chain for a tag and returns the first value found.
func HasTag(err error, tag string) (any, bool) {
	//nolint:errorlint // internal error library
	if taggedErr, ok := err.(hasTags); ok {
		if v, ok := taggedErr.Tags()[tag]; ok {
			return v, ok
		}
	}

	// Recurse into wrapped errors (similar to errors.Is/errors.As behaviour)
	//nolint:errorlint // internal error library
	if wrapped, ok := err.(wrapsMultipleErrors); ok {
		for _, child := range wrapped.Unwrap() {
			if v, ok := HasTag(child, tag); ok {
				return v, ok
			}
		}
	}
	//nolint:errorlint // internal error library
	if wrapped, ok := err.(wrapsError); ok {
		if v, ok := HasTag(wrapped.Unwrap(), tag); ok {
			return v, ok
		}
	}

	return nil, false
}

// HasTagAll returns all values for a given tag across the entire error chain.
// This is useful when multiple wrapped errors may have the same tag with different values.
func HasTagAll(err error, tag string) []any {
	var results []any
	hasTagAllRecursive(err, tag, &results)
	return results
}

func hasTagAllRecursive(err error, tag string, results *[]any) {
	if err == nil {
		return
	}

	//nolint:errorlint // internal error library
	if taggedErr, ok := err.(hasTags); ok {
		if v, ok := taggedErr.Tags()[tag]; ok {
			*results = append(*results, v)
		}
	}

	// Recurse into wrapped errors
	//nolint:errorlint // internal error library
	if wrapped, ok := err.(wrapsMultipleErrors); ok {
		for _, child := range wrapped.Unwrap() {
			hasTagAllRecursive(child, tag, results)
		}
	}
	//nolint:errorlint // internal error library
	if wrapped, ok := err.(wrapsError); ok {
		hasTagAllRecursive(wrapped.Unwrap(), tag, results)
	}
}

type sentinelError struct {
	message string
}

func (e *sentinelError) WithTag(s string, a any) Error {
	pc, _, _, _ := runtime.Caller(1)
	return &implError{
		message:    e.message,
		wrapped:    []error{e},
		stackFrame: NewStackFrame(pc),
		tags:       map[string]any{s: a},
	}
}

func (e *sentinelError) WithMessage(format string, args ...any) Error {
	pc, _, _, _ := runtime.Caller(1)
	return &implError{
		message:    e.message + sentinelSeparator + fmt.Sprintf(format, args...),
		wrapped:    []error{e},
		stackFrame: NewStackFrame(pc),
		tags:       nil,
	}
}

func (e *sentinelError) WithStackFrame() Error {
	pc, _, _, _ := runtime.Caller(1)
	return &implError{
		message:    e.message,
		wrapped:    []error{e},
		stackFrame: NewStackFrame(pc),
		tags:       nil,
	}
}

func (e *sentinelError) Error() string {
	return e.message
}

func (e *sentinelError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v', 's', 'q':
		_, _ = fmt.Fprintf(s, "%s%s\n", errorHeader, e.Error())
	}
}

func (*sentinelError) Tags() map[string]any {
	return nil
}

func (*sentinelError) StackFrame() *StackFrame {
	return nil
}

type implError struct {
	message    string
	wrapped    []error
	stackFrame *StackFrame
	tags       map[string]any
}

func (e *implError) WithTag(s string, a any) Error {
	if e.tags == nil {
		e.tags = map[string]any{s: a}
	} else {
		e.tags[s] = a
	}
	return e
}

func (e *implError) WithMessage(format string, args ...any) Error {
	if e.message == "" {
		e.message = fmt.Sprintf(format, args...)
	} else {
		e.message = e.message + sentinelSeparator + fmt.Sprintf(format, args...)
	}
	return e
}

func (e *implError) WithStackFrame() Error {
	pc, _, _, _ := runtime.Caller(1)
	e.stackFrame = NewStackFrame(pc)
	return e
}

func (e *implError) Error() string {
	return e.message
}

func (e *implError) Unwrap() []error {
	return e.wrapped
}

func (e *implError) StackFrame() *StackFrame {
	return e.stackFrame
}

func (e *implError) Tags() map[string]any {
	return e.tags
}

func (e *implError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			buf := new(bytes.Buffer)
			formatErrorChainDetailed(buf, errorHeader, "", e)
			_, _ = s.Write(buf.Bytes())
			break
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, errorHeader+e.Error()+"\n")
		if len(e.Tags()) > 0 {
			tags, err := json.Marshal(e.Tags())
			if err == nil {
				_, _ = io.WriteString(s, tagsPrefix)
				_, _ = io.Writer.Write(s, tags) //nolint:gocritic // false positive
				_, _ = io.WriteString(s, "\n")
			}
		}
	case 'q':
		_, _ = fmt.Fprintf(s, "%s%s\n", errorHeader, e.Error())
	}
}

type hasTags interface {
	error
	Tags() map[string]any
}

type hasStackFrame interface {
	error
	StackFrame() *StackFrame
}

type wrapsError interface {
	error
	Unwrap() error
}

type wrapsMultipleErrors interface {
	error
	Unwrap() []error
}

func formatErrorChainDetailed(buffer *bytes.Buffer, header, indent string, err error) {
	buffer.WriteString(indent)
	buffer.WriteString(header)
	buffer.WriteString(err.Error())
	buffer.WriteString("\n")

	//nolint:errorlint // internal error library
	if stackFrameErr, ok := err.(hasStackFrame); ok && stackFrameErr.StackFrame() != nil {
		stackFrame := stackFrameErr.StackFrame()
		buffer.WriteString(indent)
		buffer.WriteString(stackFramePrefix)
		buffer.WriteString(stackFrame.File + ":" + strconv.Itoa(stackFrame.LineNo))
		buffer.WriteString("\n")
	}

	//nolint:errorlint // internal error library
	if tagsErr, ok := err.(hasTags); ok && len(tagsErr.Tags()) > 0 {
		tags := tagsErr.Tags()
		tagsStr, err := json.Marshal(tags)
		if err == nil {
			buffer.WriteString(indent)
			buffer.WriteString(tagsPrefix)
			buffer.Write(tagsStr)
			buffer.WriteString("\n")
		}
	}

	var children []error
	//nolint:errorlint // internal error library
	if joined, ok := err.(wrapsMultipleErrors); ok {
		children = append(children, joined.Unwrap()...)
	}
	//nolint:errorlint // internal error library
	if wrapped, ok := err.(wrapsError); ok {
		children = append(children, wrapped.Unwrap())
	}
	filteredChildren := nonSentinelErrorsFilter(children)
	if len(filteredChildren) > 0 {
		buffer.WriteString(indent)
		buffer.WriteString("Caused by:\n")
		for i, child := range filteredChildren {
			formatErrorChainDetailed(buffer, "["+strconv.Itoa(i)+"] ", indent+defaultIndent, child)
		}
	}
}

func nonSentinelErrorsFilter(errs []error) []error {
	var filtered []error
	for _, e := range errs {
		//nolint:errorlint // internal error library
		if _, ok := e.(*sentinelError); !ok {
			filtered = append(filtered, e)
		}
	}

	return filtered
}
