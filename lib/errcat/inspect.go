package errcat

/*
	Return the value of `err.(*errcat.Error).Category` if that typecast works,
	or the sentinel value `errcat.unknown` if the typecast fails,
	or nil if the error is nil.

	This is useful for switching on the category of an error, even when
	functions declare that they return the broader `error` interface,
	like so:

		result, err := somepkg.SomeFunc()
		switch errcat.Category(err) {
		case nil:
			// good!  pass!
		case somepkg.ErrAlreadyDone:
			// good!  pass!
		case somepkg.ErrDataCorruption:
			// ... handle ...
		default:
			panic("bug: unknown error category")
		}
*/
func Category(err error) interface{} {
	if err == nil {
		return nil
	}
	e, ok := err.(*Error)
	if !ok {
		return (unknown)(nil)
	}
	return e.Category
}

// sentinel type
type unknown interface{}
