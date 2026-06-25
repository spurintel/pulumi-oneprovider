package main

import (
	"context"
	"fmt"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/spurintel/pulumi-oneprovider/client"
)

// VMInstanceArgs are the inputs declared in Pulumi config.
type VMInstanceArgs struct {
	Region    string   `pulumi:"region"`
	Plan      string   `pulumi:"plan"`
	OsID      string   `pulumi:"osId"`
	Hostname  string   `pulumi:"hostname"`
	SSHKeyIDs []string `pulumi:"sshKeyIds"`
}

// VMInstanceState is the full persisted state (inputs + API-computed outputs).
type VMInstanceState struct {
	VMInstanceArgs
	VMID string `pulumi:"vmId"`
	IP   string `pulumi:"ip"`
}

// VMInstance is the Pulumi resource type registered in provider.go.
type VMInstance struct{}

func newClient(ctx context.Context) *client.Client {
	cfg := infer.GetConfig[Config](ctx)
	return client.New(cfg.APIKey, cfg.ClientKey)
}

// Create provisions a new VM. Returns the VM ID as the Pulumi resource ID.
func (*VMInstance) Create(ctx context.Context, req infer.CreateRequest[VMInstanceArgs]) (infer.CreateResponse[VMInstanceState], error) {
	state := VMInstanceState{VMInstanceArgs: req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[VMInstanceState]{
			ID:     req.Name,
			Output: state,
		}, nil
	}

	c := newClient(ctx)
	vm, err := c.CreateVM(ctx, client.CreateVMRequest{
		Hostname:  req.Inputs.Hostname,
		Region:    req.Inputs.Region,
		Plan:      req.Inputs.Plan,
		OsID:      req.Inputs.OsID,
		SSHKeyIDs: req.Inputs.SSHKeyIDs,
	})
	if err != nil {
		return infer.CreateResponse[VMInstanceState]{}, fmt.Errorf("creating VM %s: %w", req.Name, err)
	}

	// Poll until active with a 10-minute deadline.
	pollCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	active, err := c.WaitForActive(pollCtx, vm.ID)
	if err != nil {
		return infer.CreateResponse[VMInstanceState]{
			ID:     vm.ID,
			Output: state,
		}, fmt.Errorf("waiting for VM %s to become active: %w", vm.ID, err)
	}

	state.VMID = active.ID
	state.IP = active.IP
	return infer.CreateResponse[VMInstanceState]{
		ID:     active.ID,
		Output: state,
	}, nil
}

// Read refreshes state from the API. Used by `pulumi refresh` and `pulumi import`.
func (*VMInstance) Read(ctx context.Context, req infer.ReadRequest[VMInstanceArgs, VMInstanceState]) (infer.ReadResponse[VMInstanceArgs, VMInstanceState], error) {
	c := newClient(ctx)
	vm, err := c.GetVM(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[VMInstanceArgs, VMInstanceState]{
			ID:     req.ID,
			Inputs: req.Inputs,
			State:  req.State,
		}, fmt.Errorf("reading VM %s: %w", req.ID, err)
	}
	state := req.State
	state.VMID = vm.ID
	state.IP = vm.IP
	return infer.ReadResponse[VMInstanceArgs, VMInstanceState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

// Delete destroys the VM.
func (*VMInstance) Delete(ctx context.Context, req infer.DeleteRequest[VMInstanceState]) (infer.DeleteResponse, error) {
	c := newClient(ctx)
	if err := c.DeleteVM(ctx, req.ID); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("deleting VM %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// Update handles in-place changes. Only Hostname is updatable; all other
// changes are handled by Diff returning UpdateReplace.
func (*VMInstance) Update(ctx context.Context, req infer.UpdateRequest[VMInstanceArgs, VMInstanceState]) (infer.UpdateResponse[VMInstanceState], error) {
	state := VMInstanceState{VMInstanceArgs: req.Inputs, VMID: req.State.VMID, IP: req.State.IP}
	if req.DryRun {
		return infer.UpdateResponse[VMInstanceState]{Output: state}, nil
	}
	if req.State.Hostname != req.Inputs.Hostname {
		c := newClient(ctx)
		if err := c.UpdateHostname(ctx, req.ID, req.Inputs.Hostname); err != nil {
			return infer.UpdateResponse[VMInstanceState]{Output: req.State}, fmt.Errorf("updating hostname for VM %s: %w", req.ID, err)
		}
	}
	return infer.UpdateResponse[VMInstanceState]{Output: state}, nil
}

// Diff marks all fields except Hostname as requiring VM replacement on change.
func (*VMInstance) Diff(_ context.Context, req infer.DiffRequest[VMInstanceArgs, VMInstanceState]) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}

	if req.State.Region != req.Inputs.Region {
		diff["region"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if req.State.Plan != req.Inputs.Plan {
		diff["plan"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if req.State.OsID != req.Inputs.OsID {
		diff["osId"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if !stringSlicesEqual(req.State.SSHKeyIDs, req.Inputs.SSHKeyIDs) {
		diff["sshKeyIds"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}

	return p.DiffResponse{
		DeleteBeforeReplace: true,
		DetailedDiff:        diff,
	}, nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
