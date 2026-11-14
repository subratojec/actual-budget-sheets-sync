package actual

// APIResponse wraps the top-level data property returned by actual-http-api
type APIResponse[T any] struct {
	Data T `json:"data"`
}

// Account represents an Actual Budget account
type Account struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Offbudget bool   `json:"offbudget"`
	Closed    bool   `json:"closed"`
}

// Category represents an Actual Budget category
type Category struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsIncome bool   `json:"is_income"`
	Hidden   bool   `json:"hidden"`
	GroupID  string `json:"group_id"`
}

// Transaction represents a transaction to add or retrieve in Actual Budget
type Transaction struct {
	ID         string `json:"id,omitempty"`
	Account    string `json:"account,omitempty"`
	Date       string `json:"date"`                 // "YYYY-MM-DD"
	Amount     int64  `json:"amount"`               // integer in minor units / cents (e.g. -5000 for -$50.00)
	Payee      string `json:"payee,omitempty"`      // Payee UUID (optional)
	PayeeName  string `json:"payee_name,omitempty"` // Payee name
	Category   string `json:"category,omitempty"`   // Category UUID (optional)
	Notes      string `json:"notes,omitempty"`
	Cleared    bool   `json:"cleared"`
	ImportedID string `json:"imported_id,omitempty"` // For deduplication
}

// CreateTransactionPayload is the body structure expected by POST /accounts/{id}/transactions
type CreateTransactionPayload struct {
	Transaction Transaction `json:"transaction"`
}

// CreateCategoryInput holds the fields to create a category
type CreateCategoryInput struct {
	Name    string `json:"name"`
	GroupID string `json:"group_id"`
}

// CreateCategoryPayload is the body structure expected by POST /categories
type CreateCategoryPayload struct {
	Category CreateCategoryInput `json:"category"`
}
