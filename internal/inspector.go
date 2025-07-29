package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/ygrebnov/workers"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type inspector interface {
	inspect(ctx context.Context, startPath string, done chan<- struct{})
}

type defaultInspector struct {
	cfg           *inspectorConfig
	baseURL       *url.URL
	excludedCodes map[int]struct{}

	htmlProvider workers.Workers[*link]
	htmlParser   workers.Workers[[]string]
	httpClient   httpClient

	visitedURLs *sync.Map

	toPrint chan<- *link

	wg sync.WaitGroup

	deps injectables
}

func newInspector(
	cfg *inspectorConfig,
	httpClient httpClient,
	visitedURLs *sync.Map,
	toPrint chan<- *link,
	deps injectables,
) (inspector, error) {
	baseURL, err := url.ParseRequestURI(cfg.Host)
	if err != nil {
		return nil, err
	}

	excludedCodes := make(map[int]struct{}, len(cfg.SkipStatusCodes))
	for _, code := range cfg.SkipStatusCodes {
		excludedCodes[code] = struct{}{}
	}

	return &defaultInspector{
		cfg:           cfg,
		baseURL:       baseURL,
		excludedCodes: excludedCodes,
		httpClient:    httpClient,
		visitedURLs:   visitedURLs,
		toPrint:       toPrint,
		wg:            sync.WaitGroup{},
		deps:          deps,
	}, nil
}

func (i *defaultInspector) inspect(ctx context.Context, startPath string, done chan<- struct{}) {
	ctx, cancel := context.WithCancel(ctx)

	i.htmlProvider = workers.New[*link](ctx, &workers.Config{MaxWorkers: uint(runtime.NumCPU()), StartImmediately: true})
	i.htmlParser = workers.New[[]string](ctx, &workers.Config{MaxWorkers: uint(runtime.NumCPU()), StartImmediately: true})

	go i.parseHTML(ctx)
	go i.provideHTML(ctx)

	i.wg.Add(1)
	_ = i.htmlProvider.AddTask(i.newGetHTMLTask(startPath))

	i.wg.Wait()
	cancel()
	done <- struct{}{}
}

// parseHTML controls HTML parsing flow.
func (i *defaultInspector) parseHTML(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case e := <-i.htmlParser.GetErrors():
			_, _ = i.deps.getPrintFn()(fmt.Errorf("error parsing page content: %w", e))
			i.wg.Done()

		case paths := <-i.htmlParser.GetResults():
			for _, path := range paths {
				i.wg.Add(1)
				_ = i.htmlProvider.AddTask(i.newGetHTMLTask(path))
			}

			i.wg.Done()
		}
	}
}

// provideHTML controls HTML provision flow.
func (i *defaultInspector) provideHTML(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case e := <-i.htmlProvider.GetErrors():
			_, _ = i.deps.getPrintFn()(fmt.Errorf("error doing http request: %w", e))
			i.wg.Done()

		case l := <-i.htmlProvider.GetResults():
			if l == nil {
				i.wg.Done()
				break
			}

			_, excludedCode := i.excludedCodes[l.code]

			if excludedCode {
				i.wg.Done()
				break
			}

			i.toPrint <- l

			if l.code < http.StatusBadRequest {
				i.wg.Add(1)
				_ = i.htmlParser.AddTask(i.newGetLinksTask(l.body))
			}

			i.wg.Done()
		}
	}
}

func (i *defaultInspector) newGetHTMLTask(path string) func(ctx context.Context) *link {
	return func(ctx context.Context) *link {
		u, err := i.baseURL.Parse(path)
		switch {
		case err != nil:
			return i.store(&link{URL: path, code: statusError})

		case u.Host != i.baseURL.Host && i.cfg.LogExternalLinks:
			return i.store(&link{URL: u.String(), code: statusExternalLink})

		case u.Host != i.baseURL.Host:
			return nil // skip external link.
		}

		existingLink, exists := i.visitedURLs.Load(u.String())
		if exists {
			existingLink.(*link).Occurrences++
			return nil
		}

		attempts := byte(0)
		for attempts < i.cfg.RetryAttempts {
			req, err1 := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
			if err1 != nil {
				return i.store(&link{URL: u.String(), code: statusError})
			}
			req.Header.Add("User-Agent", applicationName+"/"+version)

			resp, err2 := i.httpClient.Do(req)
			switch {
			case err2 != nil && errors.Is(err2, syscall.ECONNRESET):
				select {
				case <-ctx.Done():
					// TODO: add a test case for this.
					return i.store(&link{URL: u.String(), code: statusError})

				case <-time.After(i.cfg.RetryDelay):
					attempts++
				}

			case err2 != nil:
				return i.store(&link{URL: u.String(), code: statusError})

			default:
				return i.store(&link{URL: u.String(), code: resp.StatusCode, body: resp.Body})
			}
		}

		return i.store(&link{URL: u.String(), code: statusError})
	}
}

func (i *defaultInspector) store(l *link) *link {
	if _, loaded := i.visitedURLs.LoadOrStore(l.URL, l); loaded {
		return nil
	}
	return l
}

func (i *defaultInspector) newGetLinksTask(data io.ReadCloser) func(ctx context.Context) ([]string, error) {
	// TODO: pass url to return in error.
	return func(ctx context.Context) ([]string, error) {
		defer data.Close()

		var (
			doc *html.Node
			err error
		)

		attempts := byte(0)
		for attempts < i.cfg.RetryAttempts {
			doc, err = i.deps.getHTMLParse()(data)
			switch {
			case err != nil && errors.Is(err, syscall.ECONNRESET):
				select {
				case <-ctx.Done():
					// TODO: add a test case for this.
					return nil, err

				case <-time.After(i.cfg.RetryDelay):
					attempts++
				}

			case err != nil:
				return nil, err

			default:
				res := make([]string, 0)
				for n := range doc.Descendants() {
					if n.Type == html.ElementNode && n.DataAtom == atom.A {
						for _, a := range n.Attr {
							if a.Key == "href" {
								res = append(res, a.Val)
								break
							}
						}
					}
				}

				return res, nil
			}
		}

		return nil, err
	}
}
