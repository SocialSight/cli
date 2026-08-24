package oauth

import "github.com/pkg/browser"

func defaultOpenBrowser(url string) error {
	return browser.OpenURL(url)
}
