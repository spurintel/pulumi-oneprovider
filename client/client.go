// client/client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

// VM is the flattened representation returned by client methods.
type VM struct {
	ID       string
	IP       string
	State    string
	Hostname string
}

// vmCreateResponse is the wire format for POST /vm/create.
type vmCreateResponse struct {
	Result   string          `json:"result"`
	Response vmCreateDetail  `json:"response"`
}

type vmCreateDetail struct {
	ID        int    `json:"id"`
	IPAddress string `json:"ip_address"`
	Hostname  string `json:"hostname"`
}

// vmInfoResponse is the wire format for GET /vm/info/{id}.
type vmInfoResponse struct {
	Result   string       `json:"result"`
	Response vmInfoDetail `json:"response"`
}

type vmInfoDetail struct {
	IsInstalling int           `json:"is_installing"`
	ServerInfo   vmServerInfo  `json:"server_info"`
	ServerState  vmServerState `json:"server_state"`
}

type vmServerInfo struct {
	ID        string `json:"id"`
	IPAddress string `json:"ipaddress"`
	Hostname  string `json:"hostname"`
}

type vmServerState struct {
	State string `json:"state"`
}

// CreateVMRequest is the body sent to the create endpoint.
type CreateVMRequest struct {
	Hostname     string   `json:"hostname"`
	LocationID   int      `json:"location_id"`
	InstanceSize int      `json:"instance_size"`
	Template     string   `json:"template"`
	SSHKeyIDs    []string `json:"ssh_keys"`
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
	req.Header.Set("Api-Key", c.apiKey)
	req.Header.Set("Client-Key", c.clientKey)

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

// CreateVM creates a new VM and returns its initial state.
func (c *Client) CreateVM(ctx context.Context, req CreateVMRequest) (*VM, error) {
	resp, err := c.do(ctx, http.MethodPost, "/vm/create", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var wire vmCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return nil, fmt.Errorf("decoding create response: %w", err)
	}
	return &VM{
		ID:       strconv.Itoa(wire.Response.ID),
		IP:       wire.Response.IPAddress,
		Hostname: wire.Response.Hostname,
	}, nil
}

// GetVM returns the current state of a VM by ID.
func (c *Client) GetVM(ctx context.Context, vmID string) (*VM, error) {
	resp, err := c.do(ctx, http.MethodGet, "/vm/info/"+vmID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var wire vmInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return nil, fmt.Errorf("decoding get response: %w", err)
	}
	return &VM{
		ID:       wire.Response.ServerInfo.ID,
		IP:       wire.Response.ServerInfo.IPAddress,
		State:    wire.Response.ServerState.State,
		Hostname: wire.Response.ServerInfo.Hostname,
	}, nil
}

// DeleteVM destroys a VM. confirm_close is always true because Pulumi destroy
// is an explicit user action — we accept any bandwidth overage charges.
func (c *Client) DeleteVM(ctx context.Context, vmID string) error {
	id, err := strconv.Atoi(vmID)
	if err != nil {
		return fmt.Errorf("invalid VM ID %q: %w", vmID, err)
	}
	body := map[string]any{"vm_id": id, "confirm_close": true}
	resp, err := c.do(ctx, http.MethodPost, "/vm/destroy", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// UpdateHostname renames a VM in-place.
func (c *Client) UpdateHostname(ctx context.Context, vmID, hostname string) error {
	id, err := strconv.Atoi(vmID)
	if err != nil {
		return fmt.Errorf("invalid VM ID %q: %w", vmID, err)
	}
	body := map[string]any{"vm_id": id, "hostname": hostname}
	resp, err := c.do(ctx, http.MethodPost, "/vm/hostname", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// WaitForActive polls GetVM every 5 seconds until server_state.state is "online"
// or the context deadline is exceeded (caller should set a 10-minute timeout).
// While the VM is installing, the response omits server_state entirely — those
// ticks are treated as still-pending and do not error.
func (c *Client) WaitForActive(ctx context.Context, vmID string) (*VM, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("VM %s did not become online: %w", vmID, ctx.Err())
		case <-ticker.C:
			vm, err := c.GetVM(ctx, vmID)
			if err != nil {
				return nil, fmt.Errorf("polling VM %s: %w", vmID, err)
			}
			if vm.State == "online" {
				return vm, nil
			}
		}
	}
}
