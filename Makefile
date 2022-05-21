build:
	go build github.com/jyane/jnes/...

fmt:
	gofmt -w .

run:
	go run github.com/jyane/jnes/... -logtostderr

test:
	go test -v github.com/jyane/jnes/...

.PHONY: \
	build \
	fmt \
	run \
	test
