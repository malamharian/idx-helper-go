package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var errCloudflareBlocked = errors.New("cloudflare blocked request (403)")

var periodMap = map[string]string{
	"tw1":     "tw1",
	"tw2":     "tw2",
	"tw3":     "tw3",
	"tahunan": "audit",
}

const (
	idxBaseURL       = "https://idx.co.id"
	apiURL           = idxBaseURL + "/primary/ListedCompany/GetFinancialReport"
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

const fetchPageSize = 1000

func setRequestHeaders(req *http.Request, ua string) {
	if ua == "" {
		ua = defaultUserAgent
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", idxBaseURL+"/")
}

func fetchReports(client *http.Client, year, period, ua string, onProgress func(string)) ([]ReportResult, error) {
	apiPeriod := strings.ToLower(period)
	if mapped, ok := periodMap[apiPeriod]; ok {
		apiPeriod = mapped
	}

	logf := func(format string, args ...any) {
		if onProgress != nil {
			onProgress(fmt.Sprintf(format, args...))
		}
	}

	var all []ReportResult
	for page := 1; ; page++ {
		indexFrom := (page-1)*fetchPageSize + 1
		results, err := fetchReportsPage(client, year, apiPeriod, indexFrom, ua, onProgress)
		if err != nil {
			logf("Page %d (indexFrom=%d): error — %v", page, indexFrom, err)
			return nil, err
		}
		logf("Page %d (indexFrom=%d): got %d results", page, indexFrom, len(results))
		all = append(all, results...)
		if len(results) < fetchPageSize {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	logf("Total: %d results across %d page(s)", len(all), (len(all)+fetchPageSize-1)/fetchPageSize)
	return all, nil
}

func fetchReportsPage(client *http.Client, year, apiPeriod string, indexFrom int, ua string, onProgress func(string)) ([]ReportResult, error) {
	params := url.Values{
		"indexFrom":  {fmt.Sprintf("%d", indexFrom)},
		"pageSize":   {fmt.Sprintf("%d", fetchPageSize)},
		"year":       {year},
		"reportType": {"rdf"},
		"EmitenType": {"s"},
		"periode":    {apiPeriod},
		"kodeEmiten": {""},
		"SortColumn": {"KodeEmiten"},
		"SortOrder":  {"asc"},
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest("GET", apiURL+"?"+params.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		setRequestHeaders(req, ua)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*5) * time.Second)
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*5) * time.Second)
			}
			continue
		}

		if resp.StatusCode == 403 {
			return nil, errCloudflareBlocked
		}

		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*5) * time.Second)
			}
			continue
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, fmt.Errorf("parsing JSON: %w", err)
		}

		if indexFrom == 1 && len(apiResp.Results) == 0 && onProgress != nil {
			onProgress(fmt.Sprintf("DEBUG: HTTP 200 but empty results. Body preview: %s", truncateBodyForLog(body, 1200)))
		}
		return apiResp.Results, nil
	}

	return nil, fmt.Errorf("failed after 3 attempts: %w", lastErr)
}

func truncateBodyForLog(body []byte, max int) string {
	s := strings.TrimSpace(string(body))
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("... (%d more bytes)", len(s)-max)
}

func downloadFile(
	ctx context.Context,
	client *http.Client,
	company, fileName, filePath, downloadDir, year, period, ua string,
	onProgress func(string),
) bool {
	fullURL := idxBaseURL + filePath
	maxRetries := 5
	retryDelay := 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			if onProgress != nil {
				onProgress(fmt.Sprintf("Cancelled: %s", fileName))
			}
			return false
		}

		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			if onProgress != nil {
				onProgress(fmt.Sprintf("Error: %v for %s (attempt %d/%d)", err, fileName, attempt, maxRetries))
			}
			if attempt < maxRetries {
				time.Sleep(retryDelay)
			}
			continue
		}
		if ua == "" {
			ua = defaultUserAgent
		}
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Referer", idxBaseURL+"/")

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return false
			}
			if onProgress != nil {
				onProgress(fmt.Sprintf("Error: %v for %s (attempt %d/%d)", err, fileName, attempt, maxRetries))
			}
			if attempt < maxRetries {
				time.Sleep(retryDelay)
			}
			continue
		}

		if resp.StatusCode == 200 {
			safeFilename := strings.ReplaceAll(strings.ReplaceAll(fileName, "/", "-"), "\\", "-")
			outputDir := filepath.Join(downloadDir, company, year, period)
			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				resp.Body.Close()
				if onProgress != nil {
					onProgress(fmt.Sprintf("Error creating dir: %v", err))
				}
				return false
			}

			outputPath := filepath.Join(outputDir, safeFilename)
			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				if onProgress != nil {
					onProgress(fmt.Sprintf("Error reading body: %v for %s", err, fileName))
				}
				if attempt < maxRetries {
					time.Sleep(retryDelay)
				}
				continue
			}

			if err := os.WriteFile(outputPath, data, 0o644); err != nil {
				if onProgress != nil {
					onProgress(fmt.Sprintf("Error writing file: %v", err))
				}
				return false
			}

			if onProgress != nil {
				onProgress(fmt.Sprintf("Downloaded: %s/%s", company, safeFilename))
			}
			return true
		}

		resp.Body.Close()
		if onProgress != nil {
			onProgress(fmt.Sprintf("HTTP %d for %s (attempt %d/%d)", resp.StatusCode, fileName, attempt, maxRetries))
		}
		if attempt < maxRetries {
			time.Sleep(retryDelay)
		}
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("Failed after %d attempts: %s", maxRetries, fileName))
	}
	return false
}
