package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

// newTestServer creates a new mock HTTP server that serves the test data.
func newTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/index.php", func(w http.ResponseWriter, r *http.Request) {
		topic := r.URL.Query().Get("topic")
		action := r.URL.Query().Get("action")
		user := r.URL.Query().Get("u")

		if topic == "6" {
			html, err := ioutil.ReadFile("thread_6.html")
			if err != nil {
				http.Error(w, "error reading test data", http.StatusInternalServerError)
				return
			}
			fmt.Fprint(w, string(html))
			return
		}

		if action == "profile" && user == "3" {
			html, err := ioutil.ReadFile("user_3.html")
			if err != nil {
				http.Error(w, "error reading test data", http.StatusInternalServerError)
				return
			}
			fmt.Fprint(w, string(html))
			return
		}

		http.NotFound(w, r)
	})

	return httptest.NewServer(mux)
}
