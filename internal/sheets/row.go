package sheets

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// TransactionRow represents a row in the Google Sheet parsed for Actual Budget
type TransactionRow struct {
	RowIndex  int // 1-indexed row number in the spreadsheet
	DateRaw   string
	Date      string // "YYYY-MM-DD"
	Account   string
	Payee     string
	AmountRaw string
	Amount    int64 // in minor units / cents (e.g. -5000)
	Category  string
	Notes     string
	Type      string // "Expense", "Income", etc.
	Cleared   bool
	Status    string
}

// ColumnIndices stores the 0-indexed column position of each field
type ColumnIndices struct {
	Date     int
	Account  int
	Payee    int
	Amount   int
	Category int
	Notes    int
	Type     int
	Cleared  int
	Status   int
}

// DefaultColumnIndices returns fallback 0-indexed column positions
func DefaultColumnIndices() ColumnIndices {
	return ColumnIndices{
		Date:     0,
		Account:  1,
		Payee:    2,
		Amount:   3,
		Category: 4,
		Notes:    5,
		Type:     6,
		Cleared:  7,
		Status:   8,
	}
}

// DetectColumnIndices inspects the header row to determine column indexes
func DetectColumnIndices(header []any) ColumnIndices {
	indices := ColumnIndices{
		Date:     -1,
		Account:  -1,
		Payee:    -1,
		Amount:   -1,
		Category: -1,
		Notes:    -1,
		Type:     -1,
		Cleared:  -1,
		Status:   -1,
	}

	for i, col := range header {
		name := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", col)))
		switch {
		case strings.Contains(name, "date") || strings.Contains(name, "time"):
			if indices.Date == -1 {
				indices.Date = i
			}
		case strings.Contains(name, "account") || strings.Contains(name, "bank") || strings.Contains(name, "card"):
			if indices.Account == -1 {
				indices.Account = i
			}
		case strings.Contains(name, "payee") || strings.Contains(name, "merchant") || strings.Contains(name, "description") || strings.Contains(name, "to"):
			if indices.Payee == -1 {
				indices.Payee = i
			}
		case strings.Contains(name, "amount") || strings.Contains(name, "cost") || strings.Contains(name, "price") || strings.Contains(name, "total"):
			if indices.Amount == -1 {
				indices.Amount = i
			}
		case strings.Contains(name, "category") || strings.Contains(name, "cat"):
			if indices.Category == -1 {
				indices.Category = i
			}
		case strings.Contains(name, "note") || strings.Contains(name, "memo"):
			if indices.Notes == -1 {
				indices.Notes = i
			}
		case strings.Contains(name, "type"):
			if indices.Type == -1 {
				indices.Type = i
			}
		case strings.Contains(name, "clear"):
			if indices.Cleared == -1 {
				indices.Cleared = i
			}
		case strings.Contains(name, "status") || strings.Contains(name, "sync"):
			if indices.Status == -1 {
				indices.Status = i
			}
		}
	}

	// Fallback to default positions if core headers were not found
	fallback := DefaultColumnIndices()
	if indices.Date == -1 {
		indices.Date = fallback.Date
	}
	if indices.Account == -1 {
		indices.Account = fallback.Account
	}
	if indices.Payee == -1 {
		indices.Payee = fallback.Payee
	}
	if indices.Amount == -1 {
		indices.Amount = fallback.Amount
	}
	if indices.Category == -1 {
		indices.Category = fallback.Category
	}
	if indices.Notes == -1 {
		indices.Notes = fallback.Notes
	}
	if indices.Type == -1 {
		indices.Type = fallback.Type
	}
	if indices.Cleared == -1 {
		indices.Cleared = fallback.Cleared
	}
	if indices.Status == -1 {
		indices.Status = fallback.Status
	}

	return indices
}

func getCellString(row []any, idx int) string {
	if idx >= 0 && idx < len(row) && row[idx] != nil {
		return strings.TrimSpace(fmt.Sprintf("%v", row[idx]))
	}
	return ""
}

// ParseRow extracts and formats transaction details from a sheet row
func ParseRow(row []any, rowIndex int, cols ColumnIndices, defaultExpensePositive bool) (*TransactionRow, error) {
	dateRaw := getCellString(row, cols.Date)
	account := getCellString(row, cols.Account)
	payee := getCellString(row, cols.Payee)
	amountRaw := getCellString(row, cols.Amount)
	category := getCellString(row, cols.Category)
	notes := getCellString(row, cols.Notes)
	txType := getCellString(row, cols.Type)
	clearedRaw := getCellString(row, cols.Cleared)
	status := getCellString(row, cols.Status)

	// If the row is empty or already synced, skip
	if dateRaw == "" && account == "" && amountRaw == "" {
		return nil, nil // Empty row
	}

	// Parse date into YYYY-MM-DD
	parsedDate, err := ParseFlexibleDate(dateRaw)
	if err != nil {
		return nil, fmt.Errorf("row %d: invalid date %q: %w", rowIndex, dateRaw, err)
	}

	// Parse amount
	amountMinor, err := ParseAmount(amountRaw, txType, defaultExpensePositive)
	if err != nil {
		return nil, fmt.Errorf("row %d: invalid amount %q: %w", rowIndex, amountRaw, err)
	}

	cleared := true
	if clearedRaw != "" {
		lower := strings.ToLower(clearedRaw)
		if lower == "false" || lower == "no" || lower == "0" || lower == "pending" {
			cleared = false
		}
	}

	return &TransactionRow{
		RowIndex:  rowIndex,
		DateRaw:   dateRaw,
		Date:      parsedDate,
		Account:   account,
		Payee:     payee,
		AmountRaw: amountRaw,
		Amount:    amountMinor,
		Category:  category,
		Notes:     notes,
		Type:      txType,
		Cleared:   cleared,
		Status:    status,
	}, nil
}

// ParseFlexibleDate parses multiple common date representations into YYYY-MM-DD
func ParseFlexibleDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now().Format("2006-01-02"), nil
	}

	// If ISO date like 2026-11-14 or 2026-11-14T...
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		return s[:10], nil
	}

	layouts := []string{
		"2006-01-02",
		"02-01-2006",
		"01/02/2006",
		"02/01/2006",
		"2006/01/02",
		"Jan 2, 2006",
		"2 Jan 2006",
		"January 2, 2006",
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC1123,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("unrecognized date format: %s", s)
}

// ParseAmount converts numeric strings (e.g. "$15.50", "-50", "200") to integer cents
func ParseAmount(raw, txType string, defaultExpensePositive bool) (int64, error) {
	// Clean currency symbols, commas, spaces
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.ReplaceAll(cleaned, "$", "")
	cleaned = strings.ReplaceAll(cleaned, "₹", "")
	cleaned = strings.ReplaceAll(cleaned, "€", "")
	cleaned = strings.ReplaceAll(cleaned, "£", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")

	if cleaned == "" {
		return 0, fmt.Errorf("empty amount")
	}

	val, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number format: %w", err)
	}

	// Convert to integer cents (minor units)
	cents := int64(math.Round(val * 100))

	// Determine direction:
	// In Actual Budget:
	// - Negative amount = Outflow / Expense
	// - Positive amount = Inflow / Income
	typeLower := strings.ToLower(strings.TrimSpace(txType))
	if typeLower == "income" || typeLower == "inflow" || typeLower == "deposit" {
		if cents < 0 {
			cents = -cents
		}
		return cents, nil
	} else if typeLower == "expense" || typeLower == "outflow" || typeLower == "payment" {
		if cents > 0 {
			cents = -cents
		}
		return cents, nil
	}

	// If no type specified:
	if defaultExpensePositive && cents > 0 {
		cents = -cents
	}

	return cents, nil
}

// ColIndexToLetter converts 0-indexed column number to spreadsheet column letter (0->A, 25->Z, 26->AA)
func ColIndexToLetter(colIndex int) string {
	result := ""
	for colIndex >= 0 {
		result = string(rune('A'+(colIndex%26))) + result
		colIndex = (colIndex / 26) - 1
	}
	return result
}
