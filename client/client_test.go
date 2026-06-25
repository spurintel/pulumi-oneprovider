// client/client_test.go
package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spurintel/pulumi-oneprovider/client"
)

func TestCreateVM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// TODO: verify path matches actual create endpoint e.g. "/VM"
		if r.URL.Path != "/VM" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		// TODO: verify header names match actual API
		if r.Header.Get("X-API-Key") == "" {
			t.Error("missing X-API-Key header")
		}
		w.Header().Set("Content-Type", "application/json")
		// TODO: replace field names with actual API response shape
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "vm-123",
			"status": "pending",
			"ip":     "1.2.3.4",
		})
	}))
	defer srv.Close()

	c := client.NewWithBaseURL("test-api-key", "test-client-key", srv.URL)
	vm, err := c.CreateVM(context.Background(), client.CreateVMRequest{
		Hostname:  "test-host",
		Region:    "ash",
		Plan:      "SSD-1",
		OsID:      "ubuntu-22",
		SSHKeyIDs: []string{"key-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm.ID != "vm-123" {
		t.Errorf("expected ID vm-123, got %q", vm.ID)
	}
}

func TestGetVM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		// TODO: verify path pattern matches actual API e.g. "/VM/vm-123"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "vm-123",
			"status": "active", // TODO: verify actual active status string
			"ip":     "1.2.3.4",
		})
	}))
	defer srv.Close()

	c := client.NewWithBaseURL("test-api-key", "test-client-key", srv.URL)
	vm, err := c.GetVM(context.Background(), "vm-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm.Status != "active" { // TODO: match actual status string
		t.Errorf("expected status active, got %q", vm.Status)
	}
	if vm.IP != "1.2.3.4" {
		t.Errorf("expected IP 1.2.3.4, got %q", vm.IP)
	}
}

func TestDeleteVM(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// TODO: verify method and path match actual delete endpoint
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.NewWithBaseURL("test-api-key", "test-client-key", srv.URL)
	if err := c.DeleteVM(context.Background(), "vm-123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("delete endpoint was never called")
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := client.NewWithBaseURL("test-api-key", "test-client-key", srv.URL)
	_, err := c.GetVM(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("expected *client.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}
