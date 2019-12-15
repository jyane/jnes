build:
	go build github.com/jyane/jnes/...

fmt:
	go fmt *.go

run:
	go run github.com/jyane/jnes/...

test:
	go test github.com/jyane/jnes/...

.PHONY: \
	build \
	fmt \
	run \
	test
