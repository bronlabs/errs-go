package errs_test

import (
	"fmt"
	"testing"

	"github.com/bronlabs/errs-go/errs"
	"github.com/stretchr/testify/require"
)

func TestSanity(t *testing.T) {
	t.Parallel()

	e1 := errs.New("an error occurred")
	require.Error(t, e1)
	require.Equal(t, "an error occurred", e1.Error())

	e2 := e1.WithMessage("additional context is %d", 42)
	require.Error(t, e2)
	require.Equal(t, "ERROR: an error occurred: additional context is 42\n", fmt.Sprintf("%s", e2))

	tag := "code"
	e3 := e2.WithTag(tag, "E123")
	require.Error(t, e3)
	require.Equal(t, "ERROR: an error occurred: additional context is 42\n--- Tags: {\"code\":\"E123\"}\n", fmt.Sprintf("%s", e3))

	tags := e3.Tags()
	require.Contains(t, tags, tag)
	value, found := e3.Tags()[tag]
	require.True(t, found)
	require.Equal(t, "E123", value)

	v3, exists3 := errs.HasTag(e3, tag)
	require.True(t, exists3)
	require.Equal(t, "E123", v3)

	require.True(t, errs.Is(e3, e1))
	require.True(t, errs.Is(e3, e2))

	e4 := e3.WithStackFrame()
	v4, exists4 := errs.HasTag(e4, tag)
	require.True(t, exists4)
	require.Equal(t, "E123", v4)

	e5 := errs.New("a different error")

	e6 := errs.Join(e4, e5)

	require.True(t, errs.Is(e6, e1))
	require.True(t, errs.Is(e6, e2))
	require.True(t, errs.Is(e6, e3))
	require.True(t, errs.Is(e6, e4))
	require.True(t, errs.Is(e6, e5))
	require.True(t, errs.Is(e6, e6))
}

var errFoo = errs.New("FOO")

func foo() error {
	return errFoo.WithMessage("foo error")
}

func bar() error {
	return errs.Wrap(foo()).WithMessage("bar error")
}

func TestUnwrap(t *testing.T) {
	t.Parallel()
	err := bar()
	require.ErrorIs(t, err, errFoo)

	wrappedErr := errs.Unwrap(err)
	require.Len(t, wrappedErr, 1)
	require.ErrorIs(t, wrappedErr[0], errFoo)
}
