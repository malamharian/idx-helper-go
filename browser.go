package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

const (
	warmupURL     = "https://idx.co.id"
	warmupTimeout = 2 * time.Minute
	challengeWait = 90 * time.Second
)

// warmupCookies launches a headless Chrome via rod+stealth, navigates to the
// IDX website, waits for Cloudflare's JS challenge to resolve, then extracts
// the resulting cookies into an http.CookieJar.
// WarmupResult holds the cookie jar and browser User-Agent captured during warmup.
type WarmupResult struct {
	Jar       http.CookieJar
	UserAgent string
}

func warmupCookies(onLog func(string)) (*WarmupResult, error) {
	if onLog != nil {
		onLog("Launching browser for Cloudflare challenge...")
	}

	path, found := launcher.LookPath()
	if !found {
		return nil, fmt.Errorf("Chrome/Chromium not found; please install Chrome")
	}
	if onLog != nil {
		onLog(fmt.Sprintf("Found browser: %s", path))
	}

	u := launcher.New().
		Bin(path).
		Headless(true).
		Set("disable-gpu").
		Set("no-sandbox").
		MustLaunch()

	browser := rod.New().
		ControlURL(u).
		Timeout(warmupTimeout).
		MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)

	if onLog != nil {
		onLog("Navigating to idx.co.id...")
	}

	err := page.Navigate(warmupURL)
	if err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if onLog != nil {
		onLog("Waiting for Cloudflare challenge to resolve...")
	}

	deadline := time.Now().Add(challengeWait)
	resolved := false
	lastTitle := ""
	for time.Now().Before(deadline) {
		title, err := page.Eval(`() => document.title`)
		if err == nil {
			t := title.Value.Str()
			lastTitle = t
			if t != "" && t != "Just a moment..." {
				if onLog != nil {
					onLog(fmt.Sprintf("Challenge resolved (page: %s)", t))
				}
				resolved = true
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !resolved {
		return nil, fmt.Errorf("Cloudflare challenge did not resolve within %s (last title: %q)", challengeWait, lastTitle)
	}

	time.Sleep(1 * time.Second)

	// Capture the browser's actual User-Agent so HTTP requests match the fingerprint
	browserUA := ""
	if uaResult, err := page.Eval(`() => navigator.userAgent`); err == nil {
		browserUA = uaResult.Value.Str()
	}

	cookies, err := page.Cookies([]string{warmupURL})
	if err != nil {
		return nil, fmt.Errorf("extracting cookies: %w", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}

	parsedURL, _ := url.Parse(warmupURL)
	httpCookies := make([]*http.Cookie, 0, len(cookies))
	for _, c := range cookies {
		httpCookies = append(httpCookies, protoCookieToHTTP(c))
	}
	jar.SetCookies(parsedURL, httpCookies)

	if onLog != nil {
		onLog(fmt.Sprintf("Extracted %d cookies (UA: %s)", len(httpCookies), truncateCookieVal(browserUA)))
		for _, c := range httpCookies {
			onLog(fmt.Sprintf("  cookie: %s = %s...", c.Name, truncateCookieVal(c.Value)))
		}
	}

	return &WarmupResult{Jar: jar, UserAgent: browserUA}, nil
}

func protoCookieToHTTP(c *proto.NetworkCookie) *http.Cookie {
	hc := &http.Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HttpOnly: c.HTTPOnly,
	}
	if c.Expires.Time().Year() > 1970 {
		hc.Expires = c.Expires.Time()
	}
	switch c.SameSite {
	case proto.NetworkCookieSameSiteLax:
		hc.SameSite = http.SameSiteLaxMode
	case proto.NetworkCookieSameSiteStrict:
		hc.SameSite = http.SameSiteStrictMode
	case proto.NetworkCookieSameSiteNone:
		hc.SameSite = http.SameSiteNoneMode
	}
	return hc
}

func truncateCookieVal(v string) string {
	if len(v) > 20 {
		return v[:20]
	}
	return v
}
