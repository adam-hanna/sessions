.PHONY: test-cover-html
PACKAGES = $(shell find ./ -type d -not -path '*/\.*' | grep -v vendor)

fmt:
	bash -c 'go list ./... | grep -v vendor | xargs -n1 go fmt'

test:
	bash -c 'go list ./... | grep -v vendor | xargs -n1 go test -timeout=30s -tags="unit integration e2e"'

.PHONY: deps
deps:
	@go build -v ./...

.PHONY: vendor
vendor:
	@go mod vendor

# thanks!
# https://gist.github.com/skarllot/13ebe8220822bc19494c8b076aabe9fc
test-cover-html:
	echo "mode: count" > coverage-all.out
	$(foreach pkg,$(PACKAGES),\
		go test -tags="unit integration e2e" -coverprofile=coverage.out -covermode=count $(pkg);\
		tail -n +2 coverage.out >> coverage-all.out;)
