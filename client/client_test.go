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
		if r.URL.Path != "/vm/create" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Api-Key") == "" {
			t.Error("missing Api-Key header")
		}
		if r.Header.Get("Client-Key") == "" {
			t.Error("missing Client-Key header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"result": "success",
			"response": map[string]any{
				"message":    "Virtual server created",
				"id":         123456,
				"ip_address": "1.2.3.4",
				"hostname":   "test-host",
				"password":   "fAk3Passw0Rd",
			},
		})
	}))
	defer srv.Close()

	c := client.NewWithBaseURL("test-api-key", "test-client-key", srv.URL)
	vm, err := c.CreateVM(context.Background(), client.CreateVMRequest{
		Hostname:     "test-host",
		LocationID:   1,
		InstanceSize: 2,
		Template:     "linux-ubuntu-22.04-x86_64",
		SSHKeyIDs:    []string{"key-uuid-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm.ID != "123456" {
		t.Errorf("expected ID 123456, got %q", vm.ID)
	}
}

func TestGetVM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/vm/info/123456" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"result": "success",
			"response": map[string]any{
				"server_info": map[string]any{
					"id":        "123456",
					"ipaddress": "1.2.3.4",
					"hostname":  "test-host.example.com",
				},
				"server_state": map[string]any{
					"state": "online",
				},
			},
		})
	}))
	defer srv.Close()

	c := client.NewWithBaseURL("test-api-key", "test-client-key", srv.URL)
	vm, err := c.GetVM(context.Background(), "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm.State != "online" {
		t.Errorf("expected state online, got %q", vm.State)
	}
	if vm.IP != "1.2.3.4" {
		t.Errorf("expected IP 1.2.3.4, got %q", vm.IP)
	}
	if vm.ID != "123456" {
		t.Errorf("expected ID 123456, got %q", vm.ID)
	}
}

func TestDeleteVM(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/vm/destroy" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body["vm_id"] != float64(123456) {
			t.Errorf("expected vm_id 123456, got %v", body["vm_id"])
		}
		if body["confirm_close"] != true {
			t.Errorf("expected confirm_close true, got %v", body["confirm_close"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"result":   "success",
			"response": map[string]any{"message": "Virtual server destroyed"},
		})
	}))
	defer srv.Close()

	c := client.NewWithBaseURL("test-api-key", "test-client-key", srv.URL)
	if err := c.DeleteVM(context.Background(), "123456"); err != nil {
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
