package main

import (
	"context"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// VMInstanceArgs are the inputs for creating a OneProvider VM.
type VMInstanceArgs struct {
	Region    string   `pulumi:"region"`
	Plan      string   `pulumi:"plan"`
	OsID      string   `pulumi:"osId"`
	Hostname  string   `pulumi:"hostname"`
	SSHKeyIDs []string `pulumi:"sshKeyIds"`
}

// VMInstanceState is the full state (inputs + computed outputs) of a VM.
type VMInstanceState struct {
	VMInstanceArgs
	VMID string `pulumi:"vmId"`
	IP   string `pulumi:"ip"`
}

// VMInstance is the Pulumi resource type.
type VMInstance struct{}

func (*VMInstance) Create(ctx context.Context, req infer.CreateRequest[VMInstanceArgs]) (infer.CreateResponse[VMInstanceState], error) {
	state := VMInstanceState{VMInstanceArgs: req.Inputs}
	return infer.CreateResponse[VMInstanceState]{
		ID:     req.Name,
		Output: state,
	}, nil
}
