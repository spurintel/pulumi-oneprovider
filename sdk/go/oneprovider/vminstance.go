// sdk/go/oneprovider/vminstance.go
package oneprovider

import (
	"reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// VMInstanceArgs are the inputs for a OneProvider VMInstance resource.
type VMInstanceArgs struct {
	Region    pulumi.StringInput      `pulumi:"region"`
	Plan      pulumi.StringInput      `pulumi:"plan"`
	OsID      pulumi.StringInput      `pulumi:"osId"`
	Hostname  pulumi.StringInput      `pulumi:"hostname"`
	SSHKeyIDs pulumi.StringArrayInput `pulumi:"sshKeyIds"`
}

// ElementType returns the element type of the VMInstanceArgs.
func (VMInstanceArgs) ElementType() reflect.Type {
	return reflect.TypeOf((*VMInstanceArgs)(nil)).Elem()
}

// VMInstance is the resource handle returned after creation.
type VMInstance struct {
	pulumi.CustomResourceState

	Region    pulumi.StringOutput      `pulumi:"region"`
	Plan      pulumi.StringOutput      `pulumi:"plan"`
	OsID      pulumi.StringOutput      `pulumi:"osId"`
	Hostname  pulumi.StringOutput      `pulumi:"hostname"`
	SSHKeyIDs pulumi.StringArrayOutput `pulumi:"sshKeyIds"`
	VMID      pulumi.StringOutput      `pulumi:"vmId"`
	IP        pulumi.StringOutput      `pulumi:"ip"`
}

// NewVMInstance creates or looks up a VMInstance resource.
func NewVMInstance(ctx *pulumi.Context, name string, args *VMInstanceArgs, opts ...pulumi.ResourceOption) (*VMInstance, error) {
	var resource VMInstance
	err := ctx.RegisterResource("oneprovider:index:VMInstance", name, args, &resource, opts...)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}
