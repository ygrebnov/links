package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"

	"github.com/ygrebnov/links/templates"
)

const fallbackToConsoleMsg = "error generating file, printing results to console"

type printer interface {
	run(ctx context.Context, toPrint <-chan *link, finalize <-chan struct{}, done chan<- struct{})
}

type defaultPrinter struct {
	cfg  *printerConfig
	deps injectables
	data *sync.Map
	wg   sync.WaitGroup
}

func newPrinter(cfg *printerConfig, deps injectables, data *sync.Map) printer {
	if cfg == nil {
		cfg = &printerConfig{}
	}

	return &defaultPrinter{cfg: cfg, deps: deps, data: data}
}

func (p *defaultPrinter) run(
	ctx context.Context,
	toPrint <-chan *link,
	finalize <-chan struct{},
	done chan<- struct{},
) {
	go func() {
		for {
			select {
			case l := <-toPrint:
				p.wg.Add(1)
				go p.printOne(l)

			case <-finalize:
				p.wg.Wait() // wait for all p.printOne to finish.

				p.printAll(ctx)

				done <- struct{}{}
			}
		}
	}()
}

var statuses = map[int]string{
	statusError:        "ERR",
	statusExternalLink: "EXT",
}

func (p *defaultPrinter) getStatus(code int) string {
	if s, ok := statuses[code]; ok {
		return s
	}

	return strconv.Itoa(code)
}

func (p *defaultPrinter) printOne(l *link) {
	defer p.wg.Done()

	if p.cfg.SortOutput ||
		p.cfg.DisplayOccurrences ||
		p.cfg.OutputFormat.isFile() ||
		(l.code == statusOK && p.cfg.SkipOK) {
		return
	}

	_, _ = p.deps.getPrintFn()(p.getStatus(l.code), "-", l.URL)
}

func (p *defaultPrinter) printAll(ctx context.Context) {
	if !p.cfg.SortOutput &&
		!p.cfg.DisplayOccurrences &&
		!p.cfg.OutputFormat.isFile() {
		return
	}

	keys := make(sortableURLs, 0)

	p.data.Range(func(k, _ interface{}) bool {
		keys = append(keys, k.(sortableURL))

		return true
	})

	if p.cfg.SortOutput {
		sort.Sort(keys)
	}

	results := p.printResults(keys)

	defer func() {
		if ePanic := recover(); ePanic != nil {
			fmt.Println(fallbackToConsoleMsg, ePanic)

			p.cfg.OutputFormat = outputFormatStdOut
			p.printResults(keys)
		}
	}()

	if err := p.generateFile(ctx, results); err != nil {
		fmt.Println(fallbackToConsoleMsg, err)

		p.cfg.OutputFormat = outputFormatStdOut
		p.printResults(keys)
	}
}

func (p *defaultPrinter) printResults(keys sortableURLs) []*link {
	results := make([]*link, 0, len(keys))

	for _, k := range keys {
		l, _ := p.data.Load(k)
		lTyped := l.(*link)

		switch {
		case p.cfg.SkipOK && lTyped.code == statusOK:
			continue

		case p.cfg.OutputFormat.isFile():
			lTyped.Occurrences++
			lTyped.Status = p.getStatus(lTyped.code)
			results = append(results, lTyped)

		case p.cfg.DisplayOccurrences:
			_, _ = p.deps.getPrintFn()(p.getStatus(lTyped.code), "-", lTyped.Occurrences+1, "-", k)

		default:
			_, _ = p.deps.getPrintFn()(p.getStatus(lTyped.code), "-", k)
		}
	}

	return results
}

func (p *defaultPrinter) generateFile(ctx context.Context, results []*link) error {
	if !p.cfg.OutputFormat.isFile() {
		return nil
	}

	var (
		path string
		err  error
	)

	switch p.cfg.OutputFormat {
	case outputFormatHTML:
		path, err = p.generateHTMLFile(results)
	case outputFormatCSV:
		path, err = p.generateCSVFile(results)
	}

	if err != nil {
		return err
	}

	fmt.Println("generated report:", path)

	if p.cfg.DoNotOpenFileReport {
		return nil
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", path) // TODO: use context.
	case "windows", "darwin":
		cmd = exec.CommandContext(ctx, "open", path) // TODO: use context.
	default:
		return nil
	}

	return cmd.Run()
}

// createFile creates a report file with the given name and returns its path and file handle.
// The file is created in the temporary directory.
func (p *defaultPrinter) createFile(name string) (path string, file *os.File, err error) {
	path = filepath.Join(p.deps.getTempDir()(), name)

	file, err = os.Create(path)

	return
}

// generateHTMLFile generates an HTML file with the results.
func (p *defaultPrinter) generateHTMLFile(results []*link) (string, error) {
	t, err := p.deps.getTemplateParseFiles()(templates.GetLinksTemplate(), "links.html")
	if err != nil {
		return "", err
	}

	path, file, err := p.createFile("links.html")
	if err != nil {
		return "", err
	}

	defer func() {
		_ = file.Close()
	}()

	err = t.Execute(file, results)
	if err != nil {
		return "", err
	}

	return path, nil
}

// generateCSVFile generates a CSV file with the results.
func (p *defaultPrinter) generateCSVFile(results []*link) (string, error) {
	path, file, err := p.createFile("links.csv")
	if err != nil {
		return "", err
	}

	defer func() {
		_ = file.Close()
	}()

	w := csv.NewWriter(file)
	err = w.Write([]string{"Status", "Occurrences", "URL"})
	if err != nil {
		return "", err
	}

	for _, l := range results {
		err = w.Write(
			[]string{
				l.Status,
				strconv.Itoa(int(l.Occurrences + 1)),
				l.URL,
			},
		)
		if err != nil {
			return "", err
		}
	}

	w.Flush()

	return path, nil
}
