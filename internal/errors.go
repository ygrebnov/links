package internal

import "errors"

var (
	ErrInvalidConfigurationSettings    = errors.New("invalid configuration settings")
	ErrInvalidPrinterOutputFormatValue = errors.New("invalid printer.outputFormat value")
	ErrEmptyHostValue                  = errors.New("empty host value")
	ErrInvalidHostValue                = errors.New("invalid host value")
)

// compoundError holds multiple errors.
type compoundError struct {
	Errors []error
}

func (e *compoundError) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}

	return errors.Join(e.Errors...).Error()
}

func (e *compoundError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// Is checks if compoundError contains single specific error.
func (e *compoundError) Is(target error) bool {
	return len(e.Errors) == 1 && errors.Is(e.Errors[0], target)
}

// newCompoundError creates a new compoundError instance.
func newCompoundError(errs ...error) error {
	e := &compoundError{}
	for _, err := range errs {
		e.Add(err)
	}

	if len(e.Errors) == 0 {
		return nil
	}

	return e
}
