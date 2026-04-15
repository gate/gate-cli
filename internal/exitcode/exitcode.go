package exitcode

import "fmt"

// Error carries an explicit process exit code.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(code int, err error) error {
	return &Error{Code: code, Err: err}
}
