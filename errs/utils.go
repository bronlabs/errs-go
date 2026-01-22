package errs

// Must panics if err is non-nil.
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Must1 returns v or panics if err is non-nil.
func Must1[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Must2 returns (v1, v2) or panics if err is non-nil.
func Must2[T1, T2 any](v1In T1, v2In T2, err error) (v1Out T1, v2Out T2) {
	if err != nil {
		panic(err)
	}
	return v1In, v2In
}
