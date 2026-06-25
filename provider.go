package main

import (
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/middleware/schema"
)

// Config holds provider-level credentials. Both fields are required.
type Config struct {
	APIKey    string `pulumi:"apiKey"    provider:"secret"`
	ClientKey string `pulumi:"clientKey" provider:"secret"`
}

func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c.APIKey, "OneProvider API key")
	a.Describe(&c.ClientKey, "OneProvider client key")
}

func NewProvider() p.Provider {
	return infer.Provider(infer.Options{
		Metadata: schema.Metadata{
			DisplayName: "OneProvider",
			Description: "Pulumi provider for managing OneProvider VMs",
			Homepage:    "https://github.com/spurintel/pulumi-oneprovider",
			Repository:  "https://github.com/spurintel/pulumi-oneprovider",
		},
		Resources: []infer.InferredResource{
			infer.Resource[*VMInstance, VMInstanceArgs, VMInstanceState](&VMInstance{}),
		},
		Config: infer.Config[*Config](&Config{}),
	})
}
