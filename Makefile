default: acc-test

.PHONY: acc-test
acc-test:
	TF_ACC=1 go test -race ./akp/... --tags=acc -v -run 'TestAccAll' -parallel 3 $(TESTARGS) -timeout 60m

# Run unit tests
.PHONY: unit-test
unit-test:
	go test ./... --tags=unit -v $(TESTARGS) -timeout 120m

.PHONY: check-api-client-fields
check-api-client-fields:
	go run ./hack/check-api-client-fields

# Generate Documentation
.PHONY: generate
generate:
	cp -r ./docs/guides .
	go generate ./...
	mv guides ./docs
