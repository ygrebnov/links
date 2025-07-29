package internal

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const applicationName = "links"

var version, buildTime string

func Inspect(cfgFile, startURL string) error {
	cfg, cfgErr := newConfig(cfgFile, injectables{})
	if cfgErr != nil {
		return fmt.Errorf("cannot load configuration: %w", cfgErr)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	doneInspecting := make(chan struct{}, 1)
	donePrinting := make(chan struct{}, 1)
	toPrint := make(chan *link, 1024)

	data := &sync.Map{}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConnsPerHost:   1024,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	i, err := newInspector(
		cfg.Inspector,
		&http.Client{Timeout: cfg.Inspector.RequestTimeout, Transport: tr},
		data,
		toPrint,
		injectables{},
	)
	if err != nil {
		return fmt.Errorf("cannot initialize inspector: %w", err)
	}

	newPrinter(&cfg.Printer, injectables{}, data).run(ctx, toPrint, doneInspecting, donePrinting)

	i.inspect(ctx, startURL, doneInspecting)

	<-donePrinting

	return nil
}

func ShowConfig(cfgFile, o string) error {
	c, err := newConfigurator(cfgFile, injectables{})
	if err != nil {
		return fmt.Errorf("cannot initialize configurator: %w", err)
	}

	out := outputFormatYAML
	if outputFormat(o) == outputFormatJSON {
		out = outputFormatJSON
	}

	return c.show(out)
}

func SetConfig(cfgFile, key, value string) error {
	c, err := newConfigurator(cfgFile, injectables{})
	if err != nil {
		return fmt.Errorf("cannot initialize configurator: %w", err)
	}

	return c.set(key, value)
}

func ShowVersion() {
	cfgDeps := injectables{}
	_, _ = cfgDeps.getPrintFn()("links,", "version:", version+",", "built:", buildTime)
}
