// client/client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://panel.op-net.com/api"

// APIError is returned when the OneProvider API responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("OneProvider API error %d: %s", e.StatusCode, e.Body)
}

// VM represents a OneProvider virtual machine.
// TODO: verify all field names against the actual API response.
type VM struct {
	ID       string `json:"id"`       // TODO: verify field name
	IP       string `json:"ip"`       // TODO: verify field name (may be "main_ip" or "ipv4")
	Status   string `json:"status"`   // TODO: verify field name
	Hostname string `json:"hostname"` // TODO: verify field name
}

// CreateVMRequest is the body sent to the create endpoint.
// TODO: verify all field names against the actual API docs.
type CreateVMRequest struct {
	Hostname  string   `json:"hostname"`  // TODO: verify
	Region    string   `json:"location"`  // TODO: verify ("location", "region", "datacenter"?)
	Plan      string   `json:"plan"`      // TODO: verify
	OsID      string   `json:"os_id"`     // TODO: verify ("os_id", "image", "template"?)
	SSHKeyIDs []string `json:"ssh_keys"`  // TODO: verify ("ssh_keys", "sshkeys", "ssh_key_ids"?)
}

// Client calls the OneProvider API.
type Client struct {
	apiKey    string
	clientKey string
	baseURL   string
	http      *http.Client
}

// New creates a Client pointed at the production OneProvider API.
func New(apiKey, clientKey string) *Client {
	return NewWithBaseURL(apiKey, clientKey, defaultBaseURL)
}

// NewWithBaseURL creates a Client with a custom base URL (used in tests).
func NewWithBaseURL(apiKey, clientKey, baseURL string) *Client {
	return &Client{
		apiKey:    apiKey,
		clientKey: clientKey,
		baseURL:   baseURL,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// TODO: verify the exact header names for both credentials
	req.Header.Set("X-API-Key", c.apiKey)       // TODO: verify header name
	req.Header.Set("X-Client-Key", c.clientKey) // TODO: verify header name

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(b)}
	}
	return resp, nil
}

// CreateVM creates a new VM and returns it in whatever initial state the API returns.
func (c *Client) CreateVM(ctx context.Context, req CreateVMRequest) (*VM, error) {
	// TODO: verify endpoint path (e.g. "/VM", "/VM/create")
	resp, err := c.do(ctx, http.MethodPost, "/VM", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var vm VM
	if err := json.NewDecoder(resp.Body).Decode(&vm); err != nil {
		return nil, fmt.Errorf("decoding create response: %w", err)
	}
	return &vm, nil
}

// GetVM returns the current state of a VM by ID.
func (c *Client) GetVM(ctx context.Context, vmID string) (*VM, error) {
	// TODO: verify endpoint path (e.g. "/VM/{id}", "/VM?id={id}")
	resp, err := c.do(ctx, http.MethodGet, "/VM/"+vmID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var vm VM
	if err := json.NewDecoder(resp.Body).Decode(&vm); err != nil {
		return nil, fmt.Errorf("decoding get response: %w", err)
	}
	return &vm, nil
}

// DeleteVM destroys a VM by ID.
func (c *Client) DeleteVM(ctx context.Context, vmID string) error {
	// TODO: verify endpoint path and method (may be POST /VM/{id}/destroy)
	resp, err := c.do(ctx, http.MethodDelete, "/VM/"+vmID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// UpdateHostname renames a VM in-place.
func (c *Client) UpdateHostname(ctx context.Context, vmID, hostname string) error {
	// TODO: verify endpoint path and request body field name
	body := map[string]string{"hostname": hostname}
	resp, err := c.do(ctx, http.MethodPut, "/VM/"+vmID+"/hostname", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// WaitForActive polls GetVM every 5 seconds until the VM status is "active"
// or the context deadline is exceeded (caller should set a 10-minute timeout).
func (c *Client) WaitForActive(ctx context.Context, vmID string) (*VM, error) {
	// TODO: replace "active" with the actual status string from the API
	const activeStatus = "active"

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("VM %s did not become active: %w", vmID, ctx.Err())
		case <-ticker.C:
			vm, err := c.GetVM(ctx, vmID)
			if err != nil {
				return nil, fmt.Errorf("polling VM %s: %w", vmID, err)
			}
			if vm.Status == activeStatus {
				return vm, nil
			}
		}
	}
}
