package main

import (
	"context"
	"fmt"
	"os"

	p "github.com/pulumi/pulumi-go-provider"
)

var version = "0.1.0"

func main() {
	if err := p.RunProvider(context.Background(), "oneprovider", version, NewProvider()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
