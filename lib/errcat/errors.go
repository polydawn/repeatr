/*
	errcat is a simple universal error type that helps you produce
	errors that are both easy to categorize and handle, and also easy
	to maintain the original messages of.

	errcat does this by separating the two major parts of an error:
	the category and the message.

	The category is a value which you can switch on.
	*It is expected that the category field may be reassigned* as
	the error propagates up the stack.

	The message is the human-readable description of the error that occured.
	It *may* be further prepended with additional context info
	as it propagates out... or, not.
	The message may be redundant with the category: it is expected that
	the message will be printed to a user, while the category will
	not necessarily reach the user (it may be consumed by another layer
	of code, which may choose to re-categorize the error on its way up).

	Additional "details" may be attached in the Error.Details field;
	sometimes this can be used to provide key-value pairs which are
	useful in logging for other remote systems which must handle errors.
	However, usage of this should be minimized unless good reason is known;
	all handling logic should branch primarily on the category field,
	because that's what it's there for.

	errcat is specifically designed to be *serializable*, and just as
	importantly, *unserializable* again.
	This is helpful for making API-driven applications with
	consistent and reliably round-trip-able errors.
	errcat errors in json should appear as a very simple object:

		{"category":"your_tag", "msg":"full text goes here"}

	Typical usage patterns involve a const block in each package which
	enumerates the set of error category values that this package may return.
	When calling functions using the errcat convention, the callers may
	switch upon the returned Error's Category field:

		result, err := somepkg.SomeFunc()
		switch {
		case err == nil:
			// good!  pass!
		case err.Category == somepkg.ErrAlreadyDone:
			// good!  pass!
		case err.Category == somepkg.ErrDataCorruption:
			// ... handle ...
		default:
			panic("bug: unknown error category")
		}

	errcat.Error may also show up in the very top levels of a CLI application.
	In this situation, typical usage may involve the category enumeration to
	be *integer* types instead of the more typical strings; this makes it
	easy to use them as exit codes.

	Functions internal to packages may chose to panic up their errors.
	It is idiomatic to recover such internal panics and return the error
	as normal at the top of the package even when using panics as a
	non-local return system internally.
*/
package errcat

import "fmt"

var _ error = Error{}

type Error struct {
	Category interface{} // you should probably use an enum-like string
	Msg      string      // human-readable message to print
	Details  interface{} // nil, or map[string]string are the simple, advised choices
}

func (e Error) Error() string {
	return e.Msg
}

func Errorf(category interface{}, format string, args ...interface{}) error {
	return &Error{category, fmt.Sprintf(format, args...), nil}
}

func Errorw(category interface{}, cause error) error {
	if cause == nil {
		return nil
	}
	return &Error{category, cause.Error(), cause}
}

func Recategorize(err error, category interface{}) error {
	switch e2 := err.(type) {
	case *Error:
		return &Error{category, e2.Msg, e2.Details}
	default:
		return &Error{category, e2.Error(), nil}
	}
}
