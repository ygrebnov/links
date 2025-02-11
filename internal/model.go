package internal

import (
	"io"
	"net/http"
	"strings"
)

type sortableURL = string
type sortableURLs []sortableURL

func (s sortableURLs) Len() int {
	return len(s)
}

func (s sortableURLs) Less(i, j int) bool {
	si := strings.Split(s[i], "/")
	sj := strings.Split(s[j], "/")

	if len(si) != len(sj) {
		return len(si) < len(sj)
	}

	for a := range len(si) {
		if si[a] != sj[a] {
			return si[a] < sj[a]
		}
	}

	return true
}

func (s sortableURLs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type link struct {
	body        io.ReadCloser
	URL         string
	Status      string
	code        int
	Occurrences byte
}

const (
	statusOK           = 200
	statusExternalLink = 991
	statusError        = 992
)

type outputFormat string

func (o outputFormat) isFile() bool {
	return o == outputFormatHTML || o == outputFormatCSV
}

const (
	outputFormatStdOut outputFormat = "stdout"
	outputFormatHTML   outputFormat = "html"
	outputFormatCSV    outputFormat = "csv"
	outputFormatYAML   outputFormat = "yaml"
	outputFormatJSON   outputFormat = "json"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type htmlTemplate interface {
	Execute(io.Writer, interface{}) error
}
