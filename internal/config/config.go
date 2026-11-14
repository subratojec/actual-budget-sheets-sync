package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configurations
type Config struct {
	ActualAPIURL            string
	ActualAPIKey            string
	ActualSyncID            string
	ActualEncryptionPassword string

	GoogleServiceAccountKeyPath string
	GoogleSpreadsheetID         string
	GoogleSheetTabName          string
	GoogleSheetRange            string

	// DefaultExpensePositive: if true, a positive number in the Amount column is treated as an expense (negative minor units)
	DefaultExpensePositive bool
}

// LoadConfig loads configuration from environment variables or .env file
func LoadConfig(envFile string) (*Config, error) {
	if envFile != "" {
		_ = godotenv.Load(envFile)
	} else {
		_ = godotenv.Load(".env")
	}

	cfg := &Config{
		ActualAPIURL:            getEnv("ACTUAL_HTTP_API_URL", "http://localhost:5007"),
		ActualAPIKey:            os.Getenv("ACTUAL_HTTP_API_KEY"),
		ActualSyncID:            os.Getenv("ACTUAL_SYNC_ID"),
		ActualEncryptionPassword: os.Getenv("ACTUAL_ENCRYPTION_PASSWORD"),

		GoogleServiceAccountKeyPath: getEnv("GOOGLE_SERVICE_ACCOUNT_KEY_PATH", "./service-account.json"),
		GoogleSpreadsheetID:         os.Getenv("GOOGLE_SPREADSHEET_ID"),
		GoogleSheetTabName:          getEnv("GOOGLE_SHEET_TAB_NAME", "Sheet1"),
		GoogleSheetRange:            getEnv("GOOGLE_SHEET_RANGE", "Sheet1!A1:Z"),

		DefaultExpensePositive: strings.ToLower(getEnv("DEFAULT_EXPENSE_POSITIVE", "true")) == "true",
	}

	// Validate required variables
	var missing []string
	if cfg.ActualAPIKey == "" {
		missing = append(missing, "ACTUAL_HTTP_API_KEY")
	}
	if cfg.ActualSyncID == "" {
		missing = append(missing, "ACTUAL_SYNC_ID")
	}
	if cfg.GoogleSpreadsheetID == "" {
		missing = append(missing, "GOOGLE_SPREADSHEET_ID")
	}
	if cfg.GoogleServiceAccountKeyPath == "" {
		missing = append(missing, "GOOGLE_SERVICE_ACCOUNT_KEY_PATH")
	}

	if len(missing) > 0 {
		return cfg, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
