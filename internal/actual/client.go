package actual

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	baseURL            string
	apiKey             string
	syncID             string
	encryptionPassword string
	httpClient         *http.Client

	mu         sync.RWMutex
	accounts   []Account
	categories []Category
}

// NewClient creates a new Actual HTTP API client
func NewClient(baseURL, apiKey, syncID, encryptionPassword string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:            baseURL,
		apiKey:             apiKey,
		syncID:             syncID,
		encryptionPassword: encryptionPassword,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// request sends an authenticated HTTP request
func (c *Client) request(method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	if c.encryptionPassword != "" {
		req.Header.Set("x-budget-encryption-password", c.encryptionPassword)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("actual API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("failed to decode response: %w, raw: %s", err, string(respBody))
		}
	}

	return nil
}

// GetAccounts fetches all accounts for the budget
func (c *Client) GetAccounts() ([]Account, error) {
	path := fmt.Sprintf("/v1/budgets/%s/accounts", c.syncID)
	var resp APIResponse[[]Account]
	if err := c.request(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.accounts = resp.Data
	c.mu.Unlock()

	return resp.Data, nil
}

// GetCategories fetches all categories for the budget
func (c *Client) GetCategories() ([]Category, error) {
	path := fmt.Sprintf("/v1/budgets/%s/categories", c.syncID)
	var resp APIResponse[[]Category]
	if err := c.request(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.categories = resp.Data
	c.mu.Unlock()

	return resp.Data, nil
}

// EnsureMetadataLoaded loads accounts and categories into cache if empty
func (c *Client) EnsureMetadataLoaded() error {
	c.mu.RLock()
	needAccounts := len(c.accounts) == 0
	needCategories := len(c.categories) == 0
	c.mu.RUnlock()

	if needAccounts {
		if _, err := c.GetAccounts(); err != nil {
			return fmt.Errorf("failed to load accounts: %w", err)
		}
	}
	if needCategories {
		if _, err := c.GetCategories(); err != nil {
			return fmt.Errorf("failed to load categories: %w", err)
		}
	}
	return nil
}

// FindAccountByName matches an account by name (case-insensitive)
func (c *Client) FindAccountByName(name string) (*Account, error) {
	if err := c.EnsureMetadataLoaded(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	target := strings.ToLower(strings.TrimSpace(name))
	for _, acc := range c.accounts {
		if strings.ToLower(strings.TrimSpace(acc.Name)) == target {
			return &acc, nil
		}
	}

	// Partial match fallback
	for _, acc := range c.accounts {
		if strings.Contains(strings.ToLower(acc.Name), target) || strings.Contains(target, strings.ToLower(acc.Name)) {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("account not found in Actual Budget: %q", name)
}

// FindCategoryByName matches a category by name (case-insensitive)
func (c *Client) FindCategoryByName(name string) (*Category, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}

	if err := c.EnsureMetadataLoaded(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	target := strings.ToLower(strings.TrimSpace(name))
	for _, cat := range c.categories {
		if strings.ToLower(strings.TrimSpace(cat.Name)) == target {
			return &cat, nil
		}
	}

	// Partial match fallback
	for _, cat := range c.categories {
		if strings.Contains(strings.ToLower(cat.Name), target) || strings.Contains(target, strings.ToLower(cat.Name)) {
			return &cat, nil
		}
	}

	return nil, fmt.Errorf("category not found in Actual Budget: %q", name)
}

// CreateCategory creates a new category under the first available expense group
func (c *Client) CreateCategory(name string) (*Category, error) {
	if err := c.EnsureMetadataLoaded(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	var defaultGroupID string
	for _, cat := range c.categories {
		if !cat.IsIncome && cat.GroupID != "" {
			defaultGroupID = cat.GroupID
			break
		}
	}
	c.mu.RUnlock()

	if defaultGroupID == "" {
		return nil, fmt.Errorf("no category group found to assign category %q", name)
	}

	path := fmt.Sprintf("/v1/budgets/%s/categories", c.syncID)
	payload := CreateCategoryPayload{
		Category: CreateCategoryInput{
			Name:    name,
			GroupID: defaultGroupID,
		},
	}

	var resp APIResponse[string]
	if err := c.request(http.MethodPost, path, payload, &resp); err != nil {
		return nil, fmt.Errorf("failed to create category %q: %w", name, err)
	}

	// Refresh categories
	if _, err := c.GetCategories(); err != nil {
		return nil, fmt.Errorf("created category but failed to refresh categories: %w", err)
	}

	return c.FindCategoryByName(name)
}

// FindOrCreateCategory searches for a category by name, or creates it if not found
func (c *Client) FindOrCreateCategory(name string) (*Category, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}

	cat, err := c.FindCategoryByName(name)
	if err == nil && cat != nil {
		return cat, nil
	}

	// Auto-create category in Actual Budget
	return c.CreateCategory(name)
}

// CreateTransaction adds a transaction to an account
func (c *Client) CreateTransaction(accountID string, tx Transaction) error {
	path := fmt.Sprintf("/v1/budgets/%s/accounts/%s/transactions", c.syncID, accountID)
	payload := CreateTransactionPayload{
		Transaction: tx,
	}

	var resp APIResponse[any]
	return c.request(http.MethodPost, path, payload, &resp)
}
