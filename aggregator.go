package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

const excelMaxRows = 1_048_576

var excludedSheets = map[string]bool{
	"Context":    true,
	"InlineXBRL": true,
}

type sheetEntry struct {
	sourceName string
	rows       [][]interface{}
}

type fileResult struct {
	sourceName string
	sheets     map[string][][]interface{}
	err        string
}

func getXlsxFiles(baseDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".xlsx") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func readSheetsFromFile(filePath string) fileResult {
	sourceName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	sheets := make(map[string][][]interface{})

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fileResult{sourceName: sourceName, sheets: sheets, err: fmt.Sprintf("%s\t%v", filePath, err)}
	}
	defer f.Close()

	for _, sheetName := range f.GetSheetList() {
		if excludedSheets[sheetName] {
			continue
		}
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}
		var converted [][]interface{}
		for _, row := range rows {
			irow := make([]interface{}, len(row))
			for j, cell := range row {
				irow[j] = cell
			}
			converted = append(converted, irow)
		}
		sheets[sheetName] = converted
	}

	return fileResult{sourceName: sourceName, sheets: sheets}
}

func makeSheetName(base string, part int) string {
	if part == 1 {
		if len(base) > 31 {
			return base[:31]
		}
		return base
	}
	suffix := fmt.Sprintf("_%d", part)
	maxBase := 31 - len(suffix)
	if len(base) > maxBase {
		base = base[:maxBase]
	}
	return base + suffix
}

func aggregate(
	ctx context.Context,
	baseDir, outputPath string,
	workers int,
	onProgress func(string),
) (bool, []string) {
	if onProgress != nil {
		onProgress(fmt.Sprintf("Scanning %s for .xlsx files...", baseDir))
	}

	xlsxFiles, err := getXlsxFiles(baseDir)
	if err != nil {
		if onProgress != nil {
			onProgress(fmt.Sprintf("Error scanning directory: %v", err))
		}
		return false, nil
	}
	if len(xlsxFiles) == 0 {
		if onProgress != nil {
			onProgress("No .xlsx files found.")
		}
		return false, nil
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("Found %d files. Reading...", len(xlsxFiles)))
	}

	type indexedResult struct {
		result fileResult
	}

	resultsCh := make(chan indexedResult, len(xlsxFiles))

	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for _, fp := range xlsxFiles {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}
			result := readSheetsFromFile(path)
			resultsCh <- indexedResult{result: result}
		}(fp)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	sheetData := make(map[string][]sheetEntry)
	sheetOrder := make([]string, 0)
	sheetSeen := make(map[string]bool)
	var errors []string
	done := 0
	total := len(xlsxFiles)

	for ir := range resultsCh {
		if ctx.Err() != nil {
			if onProgress != nil {
				onProgress("Cancelled.")
			}
			return false, errors
		}

		done++
		r := ir.result
		if r.err != "" {
			errors = append(errors, r.err)
			if onProgress != nil {
				onProgress(fmt.Sprintf("[%d/%d] SKIP: %s", done, total, r.sourceName))
			}
			continue
		}

		if onProgress != nil {
			onProgress(fmt.Sprintf("[%d/%d] Read: %s", done, total, r.sourceName))
		}

		for sheetName, rows := range r.sheets {
			if !sheetSeen[sheetName] {
				sheetSeen[sheetName] = true
				sheetOrder = append(sheetOrder, sheetName)
			}
			sheetData[sheetName] = append(sheetData[sheetName], sheetEntry{
				sourceName: r.sourceName,
				rows:       rows,
			})
		}
	}

	if ctx.Err() != nil {
		return false, errors
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("Writing %d sheets to %s...", len(sheetData), outputPath))
	}

	outFile := excelize.NewFile()
	defer outFile.Close()

	outFile.DeleteSheet("Sheet1")

	totalRows := 0
	for _, sheetName := range sheetOrder {
		entries := sheetData[sheetName]
		part := 1
		wsName := makeSheetName(sheetName, part)
		outFile.NewSheet(wsName)
		partRows := 0

		for _, entry := range entries {
			for _, row := range entry.rows {
				if partRows >= excelMaxRows {
					part++
					wsName = makeSheetName(sheetName, part)
					outFile.NewSheet(wsName)
					partRows = 0
				}

				outRow := make([]interface{}, 0, 1+len(row))
				outRow = append(outRow, entry.sourceName)
				if len(row) == 0 {
					outRow = append(outRow, nil)
				} else {
					outRow = append(outRow, row...)
				}

				rowNum := partRows + 1
				cell, _ := excelize.CoordinatesToCellName(1, rowNum)
				outFile.SetSheetRow(wsName, cell, &outRow)
				totalRows++
				partRows++
			}
		}
	}

	if err := outFile.SaveAs(outputPath); err != nil {
		if onProgress != nil {
			onProgress(fmt.Sprintf("Error saving file: %v", err))
		}
		return false, errors
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("Done! Saved %s (%d rows, %d sheets)", outputPath, totalRows, len(sheetData)))
	}

	return true, errors
}
