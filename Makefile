build:
	go build

fmt:
	go fmt *.go

run:
	go run *.go

test:
	echo 'test'

.PHONY: \
	build \
	fmt \
	run \
	test
