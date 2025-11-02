package website

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/schollz/progressbar/v3"
)

func TestDownloadAndPackageWebsite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Test Website</title>
					<link rel="stylesheet" href="style.css">
				</head>
				<body>
					<h1>Hello, Website!</h1>
					<a href="/page2.html">Page 2</a>
					<img src="image.png">
				</body>
				</html>
			`))
		case "/style.css":
			w.Header().Set("Content-Type", "text/css")
			w.Write([]byte(`body { color: red; }`))
		case "/image.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte("fake image data"))
		case "/page2.html":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Page 2</title>
				</head>
				<body>
					<h1>Page 2</h1>
					<a href="/page3.html">Page 3</a>
				</body>
				</html>
			`))
		case "/page3.html":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Page 3</title>
				</head>
				<body>
					<h1>Page 3</h1>
				</body>
				</html>
			`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	bar := progressbar.New(1)
	dn, err := DownloadAndPackageWebsite(server.URL, 2, bar)
	if err != nil {
		t.Fatalf("DownloadAndPackageWebsite failed: %v", err)
	}

	expectedFiles := []string{"", "style.css", "image.png", "page2.html", "page3.html"}
	for _, file := range expectedFiles {
		exists, err := dn.Exists(file)
		if err != nil {
			t.Fatalf("Exists failed for %s: %v", file, err)
		}
		if !exists {
			t.Errorf("Expected to find file %s in DataNode, but it was not found", file)
		}
	}
}
