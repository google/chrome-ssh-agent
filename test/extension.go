package test

import (
	"net/url"
)

func makeExtensionUrl(extensionId string, relPath string, queryString string) *url.URL {
	return &url.URL{
		Scheme:   "chrome-extension",
		Host:     extensionId,
		Path:     relPath,
		RawQuery: queryString,
	}
}
