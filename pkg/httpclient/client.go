package httpclient

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"math/rand"
	"time"

	"golang.org/x/net/proxy"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewClient creates a new http.Client with the specified proxy settings.
func NewClient(proxyURL, proxyList string, useTor bool) (*http.Client, error) {
	if useTor {
		proxyURL = "socks5://127.0.0.1:9050"
	}

	if proxyList != "" {
		proxies, err := readProxyList(proxyList)
		if err != nil {
			return nil, err
		}
		if len(proxies) > 0 {
			proxyURL = proxies[rand.Intn(len(proxies))]
		}
	}

	if proxyURL != "" {
		proxyURLParsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("error parsing proxy URL: %w", err)
		}

		var transport http.RoundTripper
		if proxyURLParsed.Scheme == "socks5" {
			var auth *proxy.Auth
			if proxyURLParsed.User != nil {
				password, _ := proxyURLParsed.User.Password()
				auth = &proxy.Auth{
					User:     proxyURLParsed.User.Username(),
					Password: password,
				}
			}
			dialer, err := proxy.SOCKS5("tcp", proxyURLParsed.Host, auth, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("error creating SOCKS5 dialer: %w", err)
			}
			transport = &http.Transport{
				Dial: dialer.Dial,
			}
		} else {
			transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURLParsed),
			}
		}

		return &http.Client{
			Transport: transport,
		}, nil
	}

	return &http.Client{}, nil
}

func readProxyList(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening proxy list file: %w", err)
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxies = append(proxies, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading proxy list file: %w", err)
	}

	return proxies, nil
}
