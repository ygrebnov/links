package internal

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

type mockHTTPClient struct {
	data              map[string]*http.Response
	do                func(*mockHTTPClient, *http.Request) (*http.Response, error)
	connResetOccurred bool
}

func (c *mockHTTPClient) defaultDo(req *http.Request) (*http.Response, error) {
	switch {
	case req.URL.Path == "/error":
		return nil, errors.New("error")

	case req.URL.Path == "/connresetattempt" && !c.connResetOccurred:
		c.connResetOccurred = true
		return nil, syscall.ECONNRESET

	case req.URL.Path == "/connreset":
		return nil, syscall.ECONNRESET
	}

	r, ok := c.data[req.URL.String()]
	if !ok {
		return &http.Response{StatusCode: http.StatusNotFound}, nil
	}
	return r, nil
}

func (c *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.do(c, req)
}

var defaultConfig = &inspectorConfig{
	Host:             "http://host",
	LogExternalLinks: true,
	RetryDelay:       10 * time.Millisecond,
	RetryAttempts:    3,
}

var defaultHTML = `<p>Links:</p><ul>
<li><a href="link1">Link1</a>
<li><a href="/some/link2">Link2</a>
<li><a href="error">Error</a>
<li><a href="http://host/link3">Link3</a>
<li><a href="http://other.host">Other host</a>
</ul>`

func TestInspector(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *inspectorConfig
		before      func(t *testing.T) injectables
		httpClient  httpClient
		expected    map[string]int
		expectedErr error
	}{
		{
			name: "nominal",
			cfg:  defaultConfig,
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(defaultHTML)),
					},
					"http://host/link3": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`no links here`)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start":      http.StatusOK,
				"http://host/link1":      http.StatusNotFound,
				"http://host/some/link2": http.StatusNotFound,
				"http://other.host":      statusExternalLink,
				"http://host/error":      statusError,
				"http://host/link3":      http.StatusOK,
			},
		},

		{
			name: "repeating",
			cfg:  defaultConfig,
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body: io.NopCloser(
							strings.NewReader(
								`<p>Links:</p><ul>
<li><a href="link1">Link1</a>
<li><a href="link1">Link1</a>
<li><a href="link1">Link1</a>
<li><a href="link2">Link2</a>
<li><a href="link2">Link2</a>
<li><a href="http://other.host">Other host</a>
<li><a href="http://other.host">Other host</a>
</ul>`,
							),
						),
					},
					"http://host/link1": {
						StatusCode: http.StatusOK,
						Body: io.NopCloser(
							strings.NewReader(
								`<p>Links:</p><ul>
<li><a href="link1">Link1</a>
<li><a href="link1">Link1</a>
<li><a href="link1">Link1</a>
<li><a href="link2">Link2</a>
<li><a href="link2">Link2</a>
<li><a href="http://other.host">Other host</a>
</ul>`,
							),
						),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start": http.StatusOK,
				"http://host/link1": http.StatusOK,
				"http://host/link2": http.StatusNotFound,
				"http://other.host": statusExternalLink,
			},
		},

		{
			name: "excluded codes",
			cfg: &inspectorConfig{
				Host:             "http://host",
				LogExternalLinks: true,
				RetryDelay:       10 * time.Millisecond,
				RetryAttempts:    3,
				SkipStatusCodes:  []int{404},
			},
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(defaultHTML)),
					},
					"http://host/link3": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`no links here`)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start": http.StatusOK,
				"http://other.host": statusExternalLink,
				"http://host/error": statusError,
				"http://host/link3": http.StatusOK,
			},
		},

		{
			name: "skip external links",
			cfg: &inspectorConfig{
				Host:          "http://host",
				RetryDelay:    10 * time.Millisecond,
				RetryAttempts: 3,
			},
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(defaultHTML)),
					},
					"http://host/link3": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`no links here`)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start":      http.StatusOK,
				"http://host/link1":      http.StatusNotFound,
				"http://host/some/link2": http.StatusNotFound,
				"http://host/error":      statusError,
				"http://host/link3":      http.StatusOK,
			},
		},

		{
			name: "invalid host",
			cfg:  defaultConfig,
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body: io.NopCloser(
							strings.NewReader(
								`<p>Links:</p><ul>
<li><a href="link1">Link1</a>
<li><a href="--://invalid">Link2</a>
<li><a href="error">Error</a>
<li><a href="http://host/link3">Link3</a>
<li><a href="http://other.host">Other host</a>
</ul>`,
							),
						),
					},
					"http://host/link3": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`no links here`)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start": http.StatusOK,
				"http://host/link1": http.StatusNotFound,
				"--://invalid":      statusError,
				"http://other.host": statusExternalLink,
				"http://host/error": statusError,
				"http://host/link3": http.StatusOK,
			},
		},

		{
			name: "conn reset attempt",
			cfg:  defaultConfig,
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body: io.NopCloser(
							strings.NewReader(
								`<p>Links:</p><ul>
<li><a href="link1">Link1</a>
<li><a href="/some/link2">Link2</a>
<li><a href="error">Error</a>
<li><a href="http://host/link3">Link3</a>
<li><a href="http://other.host">Other host</a>
<li><a href="connresetattempt">Link1</a>
</ul>`,
							),
						),
					},
					"http://host/link3": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`no links here`)),
					},
					"http://host/connresetattempt": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`no links here`)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start":            http.StatusOK,
				"http://host/link1":            http.StatusNotFound,
				"http://host/some/link2":       http.StatusNotFound,
				"http://other.host":            statusExternalLink,
				"http://host/error":            statusError,
				"http://host/link3":            http.StatusOK,
				"http://host/connresetattempt": http.StatusOK,
			},
		},

		{
			name: "conn reset",
			cfg:  defaultConfig,
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body: io.NopCloser(
							strings.NewReader(
								`<p>Links:</p><ul>
<li><a href="link1">Link1</a>
<li><a href="/some/link2">Link2</a>
<li><a href="error">Error</a>
<li><a href="http://host/link3">Link3</a>
<li><a href="http://other.host">Other host</a>
<li><a href="connreset">Link1</a>
</ul>`,
							),
						),
					},
					"http://host/link3": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`no links here`)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start":      http.StatusOK,
				"http://host/link1":      http.StatusNotFound,
				"http://host/some/link2": http.StatusNotFound,
				"http://other.host":      statusExternalLink,
				"http://host/error":      statusError,
				"http://host/link3":      http.StatusOK,
				"http://host/connreset":  statusError,
			},
		},

		{
			name: "request timeout",
			cfg: &inspectorConfig{
				Host:             "http://host",
				LogExternalLinks: true,
				RetryDelay:       10 * time.Millisecond,
				RetryAttempts:    3,
				RequestTimeout:   500 * time.Millisecond,
			},
			httpClient: &mockHTTPClient{
				do: func(_ *mockHTTPClient, request *http.Request) (*http.Response, error) {
					s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
						time.Sleep(1 * time.Second)
					}))
					defer s.Close()

					c := &http.Client{Timeout: 500 * time.Millisecond}
					u, _ := url.Parse(s.URL)
					request.URL = u

					return c.Do(request)
				},
			},
			expected: map[string]int{
				"http://host/start": statusError,
			},
		},

		{
			name: "do request panic",
			cfg:  defaultConfig,
			httpClient: &mockHTTPClient{
				do: func(_ *mockHTTPClient, _ *http.Request) (*http.Response, error) {
					panic("do request panic")
				},
			},
			expectedErr: errors.New("error doing http request: task execution panicked: do request panic"),
		},

		{
			name: "parse html panic",
			cfg:  defaultConfig,
			before: func(*testing.T) injectables {
				return injectables{
					htmlParse: func(_ io.Reader) (*html.Node, error) {
						panic("parse html panic")
					},
				}
			},
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(defaultHTML)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start": http.StatusOK,
			},
			expectedErr: errors.New("error parsing page content: task execution panicked: parse html panic"),
		},

		{
			name: "parse html conn reset",
			cfg:  defaultConfig,
			before: func(*testing.T) injectables {
				return injectables{
					htmlParse: func(_ io.Reader) (*html.Node, error) {
						return nil, syscall.ECONNRESET
					},
				}
			},
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(defaultHTML)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start": http.StatusOK,
			},
			expectedErr: errors.New("error parsing page content: connection reset by peer"),
		},

		{
			name: "parse html error",
			cfg:  defaultConfig,
			before: func(*testing.T) injectables {
				return injectables{
					htmlParse: func(_ io.Reader) (*html.Node, error) {
						return nil, errors.New("parse html error")
					},
				}
			},
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(defaultHTML)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start": http.StatusOK,
			},
			expectedErr: errors.New("error parsing page content: parse html error"),
		},

		{
			name: "parse html conn reset once",
			cfg:  defaultConfig,
			before: func(*testing.T) injectables {
				var failedHTMLParseAttempt atomic.Bool
				return injectables{
					htmlParse: func(r io.Reader) (*html.Node, error) {
						if failedHTMLParseAttempt.Load() {
							return html.Parse(r)
						}

						failedHTMLParseAttempt.Store(true)
						return nil, syscall.ECONNRESET
					},
				}
			},
			httpClient: &mockHTTPClient{
				data: map[string]*http.Response{
					"http://host/start": {
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(defaultHTML)),
					},
				},
				do: (*mockHTTPClient).defaultDo,
			},
			expected: map[string]int{
				"http://host/start":      http.StatusOK,
				"http://host/link1":      http.StatusNotFound,
				"http://host/some/link2": http.StatusNotFound,
				"http://other.host":      statusExternalLink,
				"http://host/error":      statusError,
				"http://host/link3":      http.StatusNotFound,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := injectables{}
			if test.before != nil {
				deps = test.before(t)
			}

			var expected sync.Map
			for k, v := range test.expected {
				expected.Store(k, v)
			}

			doneInspecting := make(chan struct{}, 1)
			toPrint := make(chan *link, 1024)

			errChannel := make(chan error, 1)
			if test.expectedErr != nil {
				deps.printFn = func(a ...any) (n int, err error) {
					errChannel <- a[0].(error)
					return 0, nil
				}
			}

			i, err := newInspector(
				test.cfg,
				test.httpClient,
				&sync.Map{},
				toPrint,
				deps,
			)
			require.NoError(t, err)

			done := make(chan struct{}, 1)
			wg := sync.WaitGroup{}

			go func() {
				for {
					select {
					case <-doneInspecting:
						done <- struct{}{}

					case l := <-toPrint:
						wg.Add(1)
						expectedCode, ok := expected.LoadAndDelete(l.URL)
						if !ok {
							t.Fail()
							fmt.Println("unexpected link:", l.URL, l.code)
							wg.Done()
							done <- struct{}{}
							return
						}

						if expectedCode != l.code {
							t.Fail()
							fmt.Println("incorrect code for link:", l.URL, "got:", l.code, "want:", expectedCode)
							wg.Done()
							done <- struct{}{}
							return
						}

						wg.Done()

					case e := <-errChannel:
						wg.Add(1)
						if e.Error() != test.expectedErr.Error() {
							t.Fail()
							fmt.Println("incorrect error:\n\tgot:\n", e.Error(), "\n\twant:\n", test.expectedErr)
							wg.Done()
							done <- struct{}{}
							return
						}

						wg.Done()
					}
				}
			}()

			i.inspect("start", doneInspecting)

			<-done
			wg.Wait()

			expected.Range(func(key, value any) bool {
				fmt.Printf("missing: %s, %d\n", key, value)
				t.FailNow()
				return false
			})
		})
	}
}
