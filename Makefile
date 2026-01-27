default: acc-test

# Run all acceptance tests
.PHONY: acc-test
acc-test: acc-test-argocd acc-test-kargo

# Run ArgoCD acceptance tests (excludes Kargo tests)
.PHONY: acc-test-argocd
acc-test-argocd:
	TF_ACC=1 go test -race ./... --tags=acc -v -skip 'Kargo' $(TESTARGS) -timeout 60m -parallel 2

# Run Kargo acceptance tests
.PHONY: acc-test-kargo
acc-test-kargo:
	TF_ACC=1 go test -race ./... --tags=acc -v -run 'Kargo' $(TESTARGS) -timeout 60m -parallel 2

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
