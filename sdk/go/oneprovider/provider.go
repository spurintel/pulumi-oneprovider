// sdk/go/oneprovider/provider.go
package oneprovider

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

// Provider allows consumers to configure OneProvider credentials inline
// rather than relying solely on the stack config.
type Provider struct {
	pulumi.ProviderResourceState
}

// NewProvider creates a configured OneProvider provider instance.
func NewProvider(ctx *pulumi.Context, name string, opts ...pulumi.ResourceOption) (*Provider, error) {
	var resource Provider
	if err := ctx.RegisterResource("pulumi:providers:oneprovider", name, nil, &resource, opts...); err != nil {
		return nil, err
	}
	return &resource, nil
}
