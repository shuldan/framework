package errors

import (
	"errors"
)

func Is(err, target error) bool {
	if err == nil && target == nil {
		return false
	}
	return errors.Is(err, target)
}

func As[T error](err error, target *T) bool {
	if err == nil {
		return false
	}
	return errors.As(err, target)
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}

func Join(errs ...error) error {
	return errors.Join(errs...)
}

func GetErrorCode(err error) Code {
	var e *Error
	if As(err, &e) {
		return e.Code
	}
	return ""
}
