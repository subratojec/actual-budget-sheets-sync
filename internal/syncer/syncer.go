package syncer

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"actual-budget-sync/internal/actual"
	"actual-budget-sync/internal/config"
	"actual-budget-sync/internal/sheets"
)

type Syncer struct {
	cfg          *config.Config
	sheetsClient *sheets.Client
	actualClient *actual.Client
}

// NewSyncer creates a new synchronization orchestrator
func NewSyncer(cfg *config.Config, sClient *sheets.Client, aClient *actual.Client) *Syncer {
	return &Syncer{
		cfg:          cfg,
		sheetsClient: sClient,
		actualClient: aClient,
	}
}

// SyncResult holds the stats of a sync run
type SyncResult struct {
	TotalRows    int
	Processed    int
	Skipped      int
	Failed       int
	ErrorDetails []string
}

// Run executes the synchronization process
func (s *Syncer) Run(ctx context.Context) (*SyncResult, error) {
	log.Println("Starting sync process...")

	// 1. Fetch transactions and detected column layout from Google Sheets
	rows, cols, err := s.sheetsClient.FetchRowsAndColumns(
		ctx,
		s.cfg.GoogleSpreadsheetID,
		s.cfg.GoogleSheetRange,
		s.cfg.DefaultExpensePositive,
	)
	if err != nil {
		return nil, fmt.Errorf("error reading sheet: %w", err)
	}

	result := &SyncResult{
		TotalRows: len(rows),
	}

	if len(rows) == 0 {
		log.Println("No rows found in sheet range.")
		return result, nil
	}

	log.Printf("Retrieved %d rows from Google Sheet.", len(rows))

	// Pre-load Actual metadata
	if err := s.actualClient.EnsureMetadataLoaded(); err != nil {
		return nil, fmt.Errorf("error connecting to Actual Budget API: %w", err)
	}

	for _, row := range rows {
		// Check if already synced
		if strings.HasPrefix(strings.ToLower(row.Status), "synced") {
			result.Skipped++
			continue
		}

		if row.Account == "" {
			errStr := fmt.Sprintf("Row %d: missing account name", row.RowIndex)
			log.Println("Warning:", errStr)
			result.Failed++
			result.ErrorDetails = append(result.ErrorDetails, errStr)
			continue
		}

		// Resolve Account ID
		acc, err := s.actualClient.FindAccountByName(row.Account)
		if err != nil {
			errStr := fmt.Sprintf("Row %d: %v", row.RowIndex, err)
			log.Println("Error:", errStr)
			result.Failed++
			result.ErrorDetails = append(result.ErrorDetails, errStr)
			continue
		}

		// Resolve or Auto-Create Category ID (optional)
		var categoryID string
		if row.Category != "" {
			cat, err := s.actualClient.FindOrCreateCategory(row.Category)
			if err != nil {
				log.Printf("Row %d: Category %q could not be resolved or created: %v", row.RowIndex, row.Category, err)
			} else if cat != nil {
				categoryID = cat.ID
			}
		}

		// Prepare transaction
		tx := actual.Transaction{
			Date:       row.Date,
			Amount:     row.Amount,
			PayeeName:  row.Payee,
			Category:   categoryID,
			Notes:      row.Notes,
			Cleared:    row.Cleared,
			ImportedID: fmt.Sprintf("gsheet-%d-%d", row.RowIndex, time.Now().Unix()),
		}

		// Post transaction to Actual Budget
		if err := s.actualClient.CreateTransaction(acc.ID, tx); err != nil {
			errStr := fmt.Sprintf("Row %d: failed to create transaction in Actual: %v", row.RowIndex, err)
			log.Println("Error:", errStr)
			result.Failed++
			result.ErrorDetails = append(result.ErrorDetails, errStr)
			continue
		}

		// Mark row as synced in Google Sheet
		nowStr := time.Now().Format("2006-01-02 15:04:05")
		syncMsg := fmt.Sprintf("Synced: %s", nowStr)
		if err := s.sheetsClient.MarkRowSynced(ctx, s.cfg.GoogleSpreadsheetID, s.cfg.GoogleSheetTabName, row.RowIndex, cols.Status, syncMsg); err != nil {
			log.Printf("Warning: row %d synced to Actual but failed to update Sheet status: %v", row.RowIndex, err)
		}

		log.Printf("Successfully imported Row %d: %s | %s | %s | %.2f",
			row.RowIndex, row.Date, acc.Name, row.Payee, float64(row.Amount)/100.0)

		result.Processed++
	}

	log.Printf("Sync complete. Processed: %d, Skipped: %d, Failed: %d\n",
		result.Processed, result.Skipped, result.Failed)

	return result, nil
}
