package toolargs

import "fmt"

func errInvalidArguments(msg string) error {
	return fmt.Errorf("invalid arguments: %s", msg)
}

func errInvalidArgumentsf(format string, args ...interface{}) error {
	return fmt.Errorf("invalid arguments: "+format, args...)
}
