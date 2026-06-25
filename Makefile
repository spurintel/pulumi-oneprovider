BINARY      := pulumi-resource-oneprovider
VERSION     := 0.1.0
PLUGIN_DIR  := $(HOME)/.pulumi/plugins/resource-oneprovider-$(VERSION)

.PHONY: build install test clean

build:
	go build -o $(BINARY) .

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY) .

install: build
	mkdir -p $(PLUGIN_DIR)
	cp $(BINARY) $(PLUGIN_DIR)/$(BINARY)
	printf 'name: oneprovider\nversion: $(VERSION)\n' > $(PLUGIN_DIR)/PulumiPlugin.yaml

test:
	go test ./... -v

clean:
	rm -f $(BINARY)
