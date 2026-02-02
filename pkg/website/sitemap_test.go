package website

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFetchAndParseSitemap_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap.xml":
			fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
   <url>
      <loc>http://www.example.com/</loc>
   </url>
   <url>
      <loc>http://www.example.com/page1</loc>
   </url>
</urlset>`)
		case "/sitemap_index.xml":
			fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
   <sitemap>
      <loc>http://`+r.Host+`/sitemap1.xml</loc>
   </sitemap>
   <sitemap>
      <loc>http://`+r.Host+`/sitemap2.xml</loc>
   </sitemap>
</sitemapindex>`)
		case "/sitemap1.xml":
			fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
   <url>
      <loc>http://www.example.com/page2</loc>
   </url>
</urlset>`)
		case "/sitemap2.xml":
			fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
   <url>
      <loc>http://www.example.com/page3</loc>
   </url>
</urlset>`)
		case "/sitemap.xml.gz":
			w.Header().Set("Content-Type", "application/gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			gz.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
   <url>
      <loc>http://www.example.com/gzipped</loc>
   </url>
</urlset>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	testCases := []struct {
		name     string
		url      string
		expected []string
	}{
		{"Standard Sitemap", server.URL + "/sitemap.xml", []string{"http://www.example.com/", "http://www.example.com/page1"}},
		{"Sitemap Index", server.URL + "/sitemap_index.xml", []string{"http://www.example.com/page2", "http://www.example.com/page3"}},
		{"Gzipped Sitemap", server.URL + "/sitemap.xml.gz", []string{"http://www.example.com/gzipped"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			urls, err := FetchAndParseSitemap(tc.url, server.Client())
			if err != nil {
				t.Fatalf("FetchAndParseSitemap() error = %v", err)
			}
			if !reflect.DeepEqual(urls, tc.expected) {
				t.Errorf("FetchAndParseSitemap() = %v, want %v", urls, tc.expected)
			}
		})
	}
}

func TestDiscoverSitemap_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sitemap.xml" {
			w.WriteHeader(http.StatusOK)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sitemapURL, err := DiscoverSitemap(server.URL, server.Client())
	if err != nil {
		t.Fatalf("DiscoverSitemap() error = %v", err)
	}
	expected := server.URL + "/sitemap.xml"
	if sitemapURL != expected {
		t.Errorf("DiscoverSitemap() = %v, want %v", sitemapURL, expected)
	}
}

func TestDiscoverSitemap_Bad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	_, err := DiscoverSitemap(server.URL, server.Client())
	if err == nil {
		t.Error("DiscoverSitemap() error = nil, want error")
	}
}
