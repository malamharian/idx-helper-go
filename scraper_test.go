package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"testing"
	"time"
)

func newPlainClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConnsPerHost: 5,
		},
	}
}

func clientWithJar(jar http.CookieJar) *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		Jar:     jar,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConnsPerHost: 5,
		},
	}
}

// TestPlainFetchGets403 confirms that a plain HTTP client gets blocked by Cloudflare.
func TestPlainFetchGets403(t *testing.T) {
	client := newPlainClient()
	_, err := fetchReports(client, "2024", "tahunan", "", func(msg string) { t.Log(msg) })
	if err == nil {
		t.Fatal("Expected error, got nil — Cloudflare may not be active right now")
	}
	t.Logf("Plain client error (expected): %v", err)
}

// TestWarmupAndFetch tests the rod browser warmup + standard http.Client fetch.
func TestWarmupAndFetch(t *testing.T) {
	t.Log("=== Step 1: Warming up cookies via browser ===")
	w, err := warmupCookies(func(msg string) {
		t.Logf("  warmup: %s", msg)
	})
	if err != nil {
		t.Fatalf("Warmup failed: %v", err)
	}

	t.Log("=== Step 2: Fetching reports ===")
	client := clientWithJar(w.Jar)
	results, err := fetchReports(client, "2024", "tahunan", w.UserAgent, func(msg string) { t.Log(msg) })
	if err != nil {
		t.Fatalf("fetchReports failed after warmup: %v", err)
	}

	t.Logf("SUCCESS: got %d results", len(results))
	if len(results) > 0 {
		r := results[0]
		t.Logf("First result: %s (%d attachments)", r.KodeEmiten, len(r.Attachments))
	}
}

// TestWarmupAndDownload does the full flow but only downloads one small file.
func TestWarmupAndDownload(t *testing.T) {
	t.Log("=== Step 1: Warming up cookies ===")
	w, err := warmupCookies(func(msg string) {
		t.Logf("  warmup: %s", msg)
	})
	if err != nil {
		t.Fatalf("Warmup failed: %v", err)
	}

	client := clientWithJar(w.Jar)

	t.Log("=== Step 2: Fetching reports ===")
	results, err := fetchReports(client, "2024", "tahunan", w.UserAgent, func(msg string) { t.Log(msg) })
	if err != nil {
		t.Fatalf("fetchReports failed: %v", err)
	}
	if len(results) == 0 {
		t.Skip("No results returned")
	}

	var targetAtt Attachment
	var targetCode string
	for _, r := range results {
		for _, att := range r.Attachments {
			if att.FileType == ".xlsx" {
				targetAtt = att
				targetCode = r.KodeEmiten
				break
			}
		}
		if targetCode != "" {
			break
		}
	}
	if targetCode == "" {
		t.Skip("No xlsx attachment found")
	}

	t.Logf("=== Step 3: Downloading 1 file: %s / %s ===", targetCode, targetAtt.FileName)

	tmpDir, err := os.MkdirTemp("", "idx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	success := downloadFile(ctx, client, targetCode, targetAtt.FileName, targetAtt.FilePath, tmpDir, "2024", "audit", w.UserAgent, func(msg string) {
		t.Logf("  %s", msg)
	})

	if success {
		t.Log("Download: SUCCESS")
	} else {
		t.Error("Download: FAILED")
	}
}

func truncStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("... (%d more bytes)", len(s)-maxLen)
}
