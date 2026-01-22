# errs-go
Extendable typed error handling for Go, with tags and optional stack frames.

## Install
```bash
go get github.com/bronlabs/errs-go/errs
```

## Quick start
```go
package main

import (
	"fmt"

	"github.com/bronlabs/errs-go/errs"
)

var ValidationErr = errs.NewType("validation failed")

func main() {
	err := ValidationErr.
		WithTag("field", "email").
		WithMessage("missing @")
	
	fmt.Printf("%s", err)
}
```

## Wrapping and joining
```go
var (
    IOErr = errs.New("io error")
    UnsupportedErr = errs.New("unsupported operation")
)

func load() error {
	return IOErr.
		WithTag("path", "/tmp/input.txt").
		WithStackFrame()
}

func parse() error {
	return errs.Wrap(load()).
		WithMessage("parse error")
}

func main() {
	err := errs.Join(parse(), UnsupportedErr)
	if v, ok := errs.HasTag(err, "path"); ok {
		fmt.Println("path:", v)
	}
}
```

## Utility helpers
The `Must`, `Must1`, and `Must2` helpers are intended for setup and tests where
failing fast is acceptable, such as parsing configuration, loading fixtures, or
wiring dependencies. They let you keep call sites compact while still surfacing
unexpected errors immediately.

```go
value := errs.Must1(os.ReadFile("config.json"))
```

## Compatibility
`errs.Is` and `errs.As` are aliases for `errors.Is` and `errors.As`, so you can
use the standard library patterns across error chains produced by this library.
