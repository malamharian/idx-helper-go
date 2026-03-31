package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const defaultConcurrency = 5

type App struct {
	ctx       context.Context
	client    *http.Client
	jar       http.CookieJar
	ua        string
	mu        sync.Mutex
	cancels   map[string]context.CancelFunc
	aggCancel context.CancelFunc
	sem       chan struct{}
}

func NewApp() *App {
	jar, _ := cookiejar.New(nil)
	return &App{
		client: &http.Client{
			Timeout: 60 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		jar:     jar,
		cancels: make(map[string]context.CancelFunc),
		sem:     make(chan struct{}, defaultConcurrency),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) emitLog(msg string) {
	wailsRuntime.EventsEmit(a.ctx, "log", msg)
}

func (a *App) emitDownloadProgress(p DownloadProgress) {
	wailsRuntime.EventsEmit(a.ctx, "download:progress", p)
}

func (a *App) emitAggregateProgress(p AggregateProgress) {
	wailsRuntime.EventsEmit(a.ctx, "aggregate:progress", p)
}

// WarmupSession launches a headless browser to solve the Cloudflare challenge
// and injects the resulting cookies into the HTTP client.
func (a *App) WarmupSession() error {
	w, err := warmupCookies(a.emitLog)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.jar = w.Jar
	a.ua = w.UserAgent
	a.client.Jar = w.Jar
	a.mu.Unlock()

	return nil
}

func (a *App) FetchReports(year, period string) ([]CompanyResult, error) {
	results, err := fetchReports(a.client, year, period, a.ua, a.emitLog)

	if err != nil && errors.Is(err, errCloudflareBlocked) {
		a.emitLog("Cloudflare blocked request — warming up session...")
		if warmErr := a.WarmupSession(); warmErr != nil {
			return nil, fmt.Errorf("warmup failed: %w", warmErr)
		}
		a.emitLog("Session warmed up, retrying fetch...")
		results, err = fetchReports(a.client, year, period, a.ua, a.emitLog)
	}

	if err != nil {
		return nil, err
	}

	companies := make([]CompanyResult, 0, len(results))
	for _, r := range results {
		companies = append(companies, CompanyResult{
			Code:        r.KodeEmiten,
			Attachments: r.Attachments,
		})
	}
	return companies, nil
}

func (a *App) StartDownload(code string, attachments []Attachment, dir, year, period string) {
	a.mu.Lock()
	if cancel, ok := a.cancels[code]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancels[code] = cancel
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			delete(a.cancels, code)
			a.mu.Unlock()
		}()

		a.emitDownloadProgress(DownloadProgress{
			Code: code, Status: "Queued...", Progress: 0, Running: true,
		})

		a.sem <- struct{}{}
		defer func() { <-a.sem }()

		if ctx.Err() != nil {
			a.emitDownloadProgress(DownloadProgress{
				Code: code, Status: "Cancelled", Progress: 0, Running: false,
			})
			return
		}

		total := len(attachments)
		ok := 0

		for i, att := range attachments {
			if ctx.Err() != nil {
				a.emitDownloadProgress(DownloadProgress{
					Code:    code,
					Status:  fmt.Sprintf("Cancelled (%d/%d)", ok, total),
					Running: false,
				})
				return
			}

			a.emitDownloadProgress(DownloadProgress{
				Code:    code,
				Status:  fmt.Sprintf("Downloading %d/%d...", i+1, total),
				Running: true,
			})

			success := downloadFile(ctx, a.client, code, att.FileName, att.FilePath, dir, year, period, a.ua, func(msg string) {
				a.emitLog(msg)
			})

			if success {
				ok++
				a.emitLog(fmt.Sprintf("[%s] %s", code, att.FileName))
			} else if ctx.Err() != nil {
				a.emitDownloadProgress(DownloadProgress{
					Code:    code,
					Status:  fmt.Sprintf("Cancelled (%d/%d)", ok, total),
					Running: false,
				})
				return
			} else {
				a.emitLog(fmt.Sprintf("[%s] FAILED: %s", code, att.FileName))
			}

			a.emitDownloadProgress(DownloadProgress{
				Code:     code,
				Progress: float64(i+1) / float64(total),
				Running:  true,
			})
		}

		label := "Done"
		if ok < total {
			label = "Partial"
		}
		a.emitDownloadProgress(DownloadProgress{
			Code:     code,
			Status:   fmt.Sprintf("%s (%d/%d)", label, ok, total),
			Progress: 1.0,
			Running:  false,
		})
	}()
}

func (a *App) CancelDownload(code string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if cancel, ok := a.cancels[code]; ok {
		cancel()
	}
}

func (a *App) CancelAllDownloads() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, cancel := range a.cancels {
		cancel()
	}
}

func (a *App) SetConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sem = make(chan struct{}, n)
}

func (a *App) StartAggregate(inputDir, outputPath string) {
	a.mu.Lock()
	if a.aggCancel != nil {
		a.aggCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.aggCancel = cancel
	a.mu.Unlock()

	go func() {
		a.emitAggregateProgress(AggregateProgress{Status: "running", Message: "Aggregating..."})

		success, errors := aggregate(ctx, inputDir, outputPath, 8, func(msg string) {
			a.emitLog(msg)
		})

		if ctx.Err() != nil {
			a.emitAggregateProgress(AggregateProgress{Status: "cancelled", Message: "Cancelled"})
		} else if success {
			msg := "Done!"
			if len(errors) > 0 {
				msg = fmt.Sprintf("Done! %d error(s)", len(errors))
			}
			a.emitAggregateProgress(AggregateProgress{Status: "done", Message: msg})
		} else {
			a.emitAggregateProgress(AggregateProgress{Status: "error", Message: "Failed"})
		}
	}()
}

func (a *App) CancelAggregate() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.aggCancel != nil {
		a.aggCancel()
	}
}

func (a *App) SelectDirectory() (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Choose Directory",
	})
}

func (a *App) SelectSaveFile() (string, error) {
	return wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Save Aggregated File",
		DefaultFilename: "aggregated_financial_statements.xlsx",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Excel Files", Pattern: "*.xlsx"},
		},
	})
}
