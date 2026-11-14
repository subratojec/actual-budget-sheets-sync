package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"actual-budget-sync/internal/actual"
	"actual-budget-sync/internal/config"
	"actual-budget-sync/internal/sheets"
	"actual-budget-sync/internal/syncer"
)

func main() {
	envPath := flag.String("env", ".env", "Path to .env file")
	testConn := flag.Bool("test-connection", false, "Test connections to Actual Budget and Google Sheets")
	flag.Parse()

	log.Println("Actual Budget Sheet Sync initializing...")

	cfg, err := config.LoadConfig(*envPath)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	ctx := context.Background()

	// Initialize Google Sheets client
	sClient, err := sheets.NewClient(ctx, cfg.GoogleServiceAccountKeyPath)
	if err != nil {
		log.Fatalf("Failed to initialize Google Sheets client: %v", err)
	}

	// Initialize Actual Budget client
	aClient := actual.NewClient(cfg.ActualAPIURL, cfg.ActualAPIKey, cfg.ActualSyncID, cfg.ActualEncryptionPassword)

	// If test-connection mode
	if *testConn {
		log.Println("Testing connection to Actual Budget...")
		accounts, err := aClient.GetAccounts()
		if err != nil {
			log.Fatalf("Actual Budget connection failed: %v", err)
		}
		log.Printf("Actual Budget connected! Found %d accounts.", len(accounts))
		for _, acc := range accounts {
			log.Printf(" - Account: %s (ID: %s, Closed: %t)", acc.Name, acc.ID, acc.Closed)
		}

		categories, err := aClient.GetCategories()
		if err != nil {
			log.Printf("Warning: failed to fetch categories: %v", err)
		} else {
			log.Printf("Found %d categories in Actual Budget.", len(categories))
		}

		log.Println("Testing connection to Google Sheets...")
		rows, cols, err := sClient.FetchRowsAndColumns(ctx, cfg.GoogleSpreadsheetID, cfg.GoogleSheetRange, cfg.DefaultExpensePositive)
		if err != nil {
			log.Fatalf("Google Sheets connection failed: %v", err)
		}
		log.Printf("Google Sheets connected! Found %d rows.", len(rows))
		log.Printf("Detected columns: Date=%s, Account=%s, Payee=%s, Amount=%s, Category=%s, Notes=%s, Status=%s",
			sheets.ColIndexToLetter(cols.Date),
			sheets.ColIndexToLetter(cols.Account),
			sheets.ColIndexToLetter(cols.Payee),
			sheets.ColIndexToLetter(cols.Amount),
			sheets.ColIndexToLetter(cols.Category),
			sheets.ColIndexToLetter(cols.Notes),
			sheets.ColIndexToLetter(cols.Status),
		)
		log.Println("All connections successful!")
		os.Exit(0)
	}

	// Run regular sync
	s := syncer.NewSyncer(cfg, sClient, aClient)
	result, err := s.Run(ctx)
	if err != nil {
		log.Fatalf("Sync execution failed: %v", err)
	}

	if result.Failed > 0 {
		fmt.Fprintf(os.Stderr, "Finished with %d errors.\n", result.Failed)
		os.Exit(1)
	}

	log.Println("Sync job finished successfully.")
}
