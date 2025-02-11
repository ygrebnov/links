package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
)

func newServer() *httptest.Server {
	rootHandler := func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `<p>Links:</p><ul>
<li><a href="nosubsequentlinks">no subsequent links</a>,</li>
<li><a href="error">error</a>,</li>
<li><a href="notfound">not found</a>,</li>
<li><a href="http://other.host">external link</a>.</li>
</ul>`)
	}

	nosubsequentlinksHandler := func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "No links here.")
	}

	errorHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/nosubsequentlinks", nosubsequentlinksHandler)
	http.HandleFunc("/error", errorHandler)
	http.HandleFunc("/notfound", http.NotFound)

	return httptest.NewServer(http.DefaultServeMux)
}
