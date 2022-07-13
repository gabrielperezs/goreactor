default: build

build: export CGO_ENABLED = 0
build:
	go env -w GOPRIVATE="github.com/Webjet/*"
	go build -o goreactor -ldflags '-w -s'

arm: export CGO_ENABLED = 0
arm:
	go env -w GOPRIVATE="github.com/Webjet/*"
	env GOOS=linux GOARCH=arm64 GONOSUMDB=github.com/Webjet/engine go build -o goreactor_arm -ldflags '-w -s'

clean:
	go clean
