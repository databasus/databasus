package telegram

import (
	"net/http"
	"net/url"
	"slices"
)

func parseProxyURL(rawProxyURL string) (*url.URL, error) {
	parsedProxyURL, err := url.Parse(rawProxyURL)
	if err != nil || !slices.Contains([]string{"http", "https", "socks5", "socks5h"}, parsedProxyURL.Scheme) ||
		parsedProxyURL.Host == "" {
		return nil, errInvalidProxyURL
	}

	return parsedProxyURL, nil
}

func newProxyTransport(proxyURL *url.URL) (*http.Transport, error) {
	defaultTransport, isTransport := http.DefaultTransport.(*http.Transport)
	if !isTransport {
		return nil, errUnexpectedDefaultTransport
	}

	transport := defaultTransport.Clone()
	transport.Proxy = http.ProxyURL(proxyURL)

	return transport, nil
}
