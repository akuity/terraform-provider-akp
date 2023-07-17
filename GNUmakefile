default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

.PHONY: generate-terraformtypes
generate-terraformtypes:
	go run ./genterraformtypes/main.go
	go fmt ./akp/types/...
