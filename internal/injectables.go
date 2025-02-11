package internal

import (
	"fmt"
	"html/template"
	"io"
	"os"

	"golang.org/x/net/html"
)

// injectables holds injectable dependencies.
type injectables struct {
	userConfigDir      func() (string, error)
	stat               func(name string) (os.FileInfo, error)
	tempDir            func() string
	templateParseFiles func(filenames ...string) (htmlTemplate, error)
	htmlParse          func(io.Reader) (*html.Node, error)
	printFn            func(a ...any) (n int, err error)
}

// getUserConfigDir returns the userConfigDir dependency or the default implementation.
func (i *injectables) getUserConfigDir() func() (string, error) {
	if i.userConfigDir != nil {
		return i.userConfigDir
	}

	return os.UserConfigDir
}

// getStat returns the stat dependency or the default implementation.
func (i *injectables) getStat() func(name string) (os.FileInfo, error) {
	if i.stat != nil {
		return i.stat
	}

	return os.Stat
}

// getTempDir returns the tempDir dependency or the default implementation.
func (i *injectables) getTempDir() func() string {
	if i.tempDir != nil {
		return i.tempDir
	}

	return os.TempDir
}

// getTemplateParseFiles returns the templateParseFiles dependency or the default implementation.
func (i *injectables) getTemplateParseFiles() func(filenames ...string) (htmlTemplate, error) {
	if i.templateParseFiles != nil {
		return i.templateParseFiles
	}

	return func(filenames ...string) (htmlTemplate, error) {
		return template.ParseFiles(filenames...)
	}
}

// getHTMLParse returns the htmlParse dependency or the default implementation.
func (i *injectables) getHTMLParse() func(io.Reader) (*html.Node, error) {
	if i.htmlParse != nil {
		return i.htmlParse
	}

	return html.Parse
}

// getPrintFn returns the printFn dependency or the default implementation.
func (i *injectables) getPrintFn() func(a ...any) (n int, err error) {
	if i.printFn != nil {
		return i.printFn
	}

	return fmt.Println
}
