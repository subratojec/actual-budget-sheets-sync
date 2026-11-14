package sheets

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/option"
	sheetsv4 "google.golang.org/api/sheets/v4"
)

type Client struct {
	service *sheetsv4.Service
}

// NewClient initializes a Google Sheets client using a Service Account JSON credentials file
func NewClient(ctx context.Context, credentialsFilePath string) (*Client, error) {
	srv, err := sheetsv4.NewService(ctx,
		option.WithCredentialsFile(credentialsFilePath),
		option.WithScopes(sheetsv4.SpreadsheetsScope),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize sheets service: %w", err)
	}

	return &Client{service: srv}, nil
}

// FetchRowsAndColumns retrieves all values from the specified range and detects columns
func (c *Client) FetchRowsAndColumns(ctx context.Context, spreadsheetID, sheetRange string, defaultExpensePositive bool) ([]TransactionRow, ColumnIndices, error) {
	resp, err := c.service.Spreadsheets.Values.Get(spreadsheetID, sheetRange).Context(ctx).Do()
	if err != nil {
		return nil, ColumnIndices{}, fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}

	if len(resp.Values) == 0 {
		return nil, ColumnIndices{}, nil
	}

	// Assume first row could be headers
	headerRow := resp.Values[0]
	cols := DetectColumnIndices(headerRow)

	var transactions []TransactionRow

	// Determine starting row number in the spreadsheet
	// If range is like "Sheet1!A1:Z", row 0 is 1. If "Sheet1!A2:Z", row 0 is 2.
	startRow := 1
	if strings.Contains(sheetRange, "!") {
		parts := strings.Split(sheetRange, "!")
		if len(parts) > 1 {
			// e.g. "A2:Z" or "A1:Z"
			r := parts[1]
			for i, ch := range r {
				if ch >= '0' && ch <= '9' {
					fmt.Sscanf(r[i:], "%d", &startRow)
					break
				}
			}
		}
	}

	// If row 0 was treated as header, start iterating from row 1
	startIdx := 1
	// But if row 0 didn't contain headers (e.g. was already data), start from 0
	firstDate := getCellString(headerRow, cols.Date)
	if _, err := ParseFlexibleDate(firstDate); err == nil && !strings.Contains(strings.ToLower(firstDate), "date") {
		// Header was actually a transaction!
		startIdx = 0
	}

	for i := startIdx; i < len(resp.Values); i++ {
		rowNumber := startRow + i
		txRow, err := ParseRow(resp.Values[i], rowNumber, cols, defaultExpensePositive)
		if err != nil {
			// Log and skip invalid row
			fmt.Printf("Warning: failed parsing row %d: %v\n", rowNumber, err)
			continue
		}
		if txRow == nil {
			continue // empty row
		}
		transactions = append(transactions, *txRow)
	}

	return transactions, cols, nil
}

// MarkRowSynced updates the Status column cell of the specified row to "Synced: <timestamp>"
func (c *Client) MarkRowSynced(ctx context.Context, spreadsheetID, tabName string, rowIndex int, statusColIndex int, message string) error {
	colLetter := ColIndexToLetter(statusColIndex)
	cellRange := fmt.Sprintf("%s!%s%d", tabName, colLetter, rowIndex)

	valRange := &sheetsv4.ValueRange{
		Values: [][]any{{message}},
	}

	_, err := c.service.Spreadsheets.Values.Update(spreadsheetID, cellRange, valRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to update cell %s: %w", cellRange, err)
	}

	return nil
}
