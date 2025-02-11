package internal

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type res struct {
	mu sync.Mutex
	d  []string
}

func (r *res) add(s string) {
	r.mu.Lock()
	r.d = append(r.d, s)
	r.mu.Unlock()
}

type mockTemplate struct {
	executeFn func(file io.Writer, data interface{}) error
}

func (t *mockTemplate) Execute(file io.Writer, data interface{}) error {
	return t.executeFn(file, data)
}

func TestPrinter(t *testing.T) {
	tests := []struct {
		name       string
		before     func(t *testing.T) injectables
		tempDir    string
		cfg        *printerConfig
		data       []*link
		expected   []string
		checkOrder bool
		checkFile  bool
		fileName   string
	}{
		{
			name: "nominal",
			cfg:  nil,
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected: []string{"200 - link1", "404 - link2", "ERR - link3", "EXT - link4"},
		},

		{
			name: "sorted",
			cfg:  &printerConfig{SortOutput: true},
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link1/level2", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected:   []string{"200 - link1", "404 - link2", "ERR - link3", "EXT - link4", "200 - link1/level2"},
			checkOrder: true,
		},

		{
			name: "display occurrences",
			cfg:  &printerConfig{DisplayOccurrences: true},
			data: []*link{
				{URL: "link2", Occurrences: 1, code: http.StatusNotFound},
				{URL: "link4", Occurrences: 0, code: statusExternalLink},
				{URL: "link1", Occurrences: 24, code: http.StatusOK},
				{URL: "link3", Occurrences: 0, code: statusError},
			},
			expected: []string{"200 - 25 - link1", "404 - 2 - link2", "ERR - 1 - link3", "EXT - 1 - link4"},
		},

		{
			name: "skip ok",
			cfg:  &printerConfig{SkipOK: true},
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected: []string{"404 - link2", "ERR - link3", "EXT - link4"},
		},

		{
			name: "skip ok, sort output",
			cfg:  &printerConfig{SkipOK: true, SortOutput: true},
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected:   []string{"404 - link2", "ERR - link3", "EXT - link4"},
			checkOrder: true,
		},

		{
			name:    "html output",
			tempDir: t.TempDir(),
			cfg:     &printerConfig{OutputFormat: outputFormatHTML, DoNotOpenFileReport: true},
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected:  []string{"200 - link1", "404 - link2", "ERR - link3", "EXT - link4"},
			checkFile: true,
			fileName:  "links.html",
		},

		{
			name:    "csv output",
			tempDir: t.TempDir(),
			cfg:     &printerConfig{OutputFormat: outputFormatCSV, DoNotOpenFileReport: true},
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected:  []string{"200 - link1", "404 - link2", "ERR - link3", "EXT - link4"},
			checkFile: true,
			fileName:  "links.csv",
		},

		{
			name: "error parsing template",
			before: func(t *testing.T) injectables {
				t.Chdir(t.TempDir())

				return injectables{}
			},
			cfg: &printerConfig{OutputFormat: outputFormatHTML, DoNotOpenFileReport: true},
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected: []string{"200 - link1", "404 - link2", "ERR - link3", "EXT - link4"},
		},

		{
			name:    "error creating report file",
			tempDir: "-:",
			cfg:     &printerConfig{OutputFormat: outputFormatHTML, DoNotOpenFileReport: true},
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected: []string{"200 - link1", "404 - link2", "ERR - link3", "EXT - link4"},
		},

		{
			name:    "error executing template",
			tempDir: t.TempDir(),
			before: func(*testing.T) injectables {
				return injectables{
					templateParseFiles: func(_ ...string) (htmlTemplate, error) {
						return &mockTemplate{
							executeFn: func(_ io.Writer, _ interface{}) error {
								return errors.New("error executing template")
							},
						}, nil
					},
				}
			},
			cfg: &printerConfig{OutputFormat: outputFormatHTML, DoNotOpenFileReport: true},
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected: []string{"200 - link1", "404 - link2", "ERR - link3", "EXT - link4"},
		},

		{
			name:    "panic on executing template",
			tempDir: t.TempDir(),
			before: func(*testing.T) injectables {
				return injectables{
					templateParseFiles: func(_ ...string) (htmlTemplate, error) {
						return &mockTemplate{
							executeFn: func(_ io.Writer, _ interface{}) error {
								panic("panic on executing template")
							},
						}, nil
					},
				}
			},
			cfg: &printerConfig{OutputFormat: outputFormatHTML, DoNotOpenFileReport: true},
			data: []*link{
				{URL: "link2", code: http.StatusNotFound},
				{URL: "link4", code: statusExternalLink},
				{URL: "link1", code: http.StatusOK},
				{URL: "link3", code: statusError},
			},
			expected: []string{"200 - link1", "404 - link2", "ERR - link3", "EXT - link4"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := injectables{}
			if test.before != nil {
				deps = test.before(t)
			}

			if test.tempDir != "" {
				deps.tempDir = func() string {
					return test.tempDir
				}
			}

			t.Chdir("..")

			r := &res{}
			deps.printFn = func(a ...any) (n int, err error) {
				r.add(strings.Trim(fmt.Sprint(a), "[]"))
				return 0, nil
			}
			data := &sync.Map{}
			p := newPrinter(test.cfg, deps, data)

			doneInspecting := make(chan struct{}, 1)
			donePrinting := make(chan struct{}, 1)
			toPrint := make(chan *link, 1024)

			p.run(toPrint, doneInspecting, donePrinting)

			for _, l := range test.data {
				data.Store(l.URL, l)
				toPrint <- l
			}
			time.Sleep(10 * time.Millisecond)

			doneInspecting <- struct{}{}

			<-donePrinting

			switch {
			case test.checkFile:
				reportFile := filepath.Join(test.tempDir, test.fileName)
				_, err := os.Stat(reportFile)
				require.NoError(t, err)

				f, err := os.Open(reportFile)
				require.NoError(t, err)

				b, err := io.ReadAll(f)
				require.NoError(t, err)

				require.NotEmpty(t, b)
				require.NoError(t, f.Close())

			case test.checkOrder:
				require.Len(t, r.d, len(test.expected))

				for i := range test.expected {
					require.Equal(t, test.expected[i], r.d[i])
				}

			default:
				require.ElementsMatch(t, test.expected, r.d)
			}
		})
	}
}
