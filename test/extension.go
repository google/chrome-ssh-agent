package test

import (
	"net/url"
)

const (
	extensionId = "eechpbnaifiimgajnomdipfaamobdfha"
)

func makeExtensionUrl(relPath string, queryString string) *url.URL {
	return &url.URL{
		Scheme:   "chrome-extension",
		Host:     extensionId,
		Path:     relPath,
		RawQuery: queryString,
	}
}
