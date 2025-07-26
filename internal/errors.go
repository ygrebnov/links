package internal

import "github.com/ygrebnov/errorc"

var (
	ErrInvalidConfigurationSettings    = errorc.New("invalid configuration settings")
	ErrInvalidPrinterOutputFormatValue = errorc.New("invalid printer.outputFormat value")
	ErrEmptyHostValue                  = errorc.New("empty host value")
	ErrInvalidHostValue                = errorc.New("invalid host value")
)
