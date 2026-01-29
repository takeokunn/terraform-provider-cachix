default: build

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

fmt:
	gofmt -s -w .
	terraform fmt -recursive ./examples/

test:
	go test -v -cover -timeout=120s -parallel=4 -skip '^TestAcc' ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

generate:
	cd tools && go generate ./...

docs: generate
	go generate ./...

clean:
	go clean -testcache
	rm -f terraform-provider-cachix

.PHONY: default build install lint fmt test testacc generate docs clean
