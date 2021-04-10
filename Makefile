gen:
	./hack/update-codegen.sh
lint:
	go fmt ./...
	golangci-lint run
test:
	go test ./...
