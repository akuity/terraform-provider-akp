default: acc-test

# Run acceptance tests
.PHONY: acc-test
acc-test:
	TF_ACC=1 go test ./... --tags=acc -v $(TESTARGS) -timeout 120m

# Run unit tests
.PHONY: unit-test
unit-test:
	go test ./... --tags=unit -v $(TESTARGS) -timeout 120m

# Generate Documentation
.PHONY: generate
generate:
	cp -r ./docs/guides .
	go generate ./...
	mv guides ./docs