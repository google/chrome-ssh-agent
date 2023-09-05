package e2e

import (
	"net/url"
)

func makeExtensionURL(extensionID string, relPath string, queryString string) *url.URL {
	return &url.URL{
		Scheme:   "chrome-extension",
		Host:     extensionID,
		Path:     relPath,
		RawQuery: queryString,
	}
}
