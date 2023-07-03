plugin-controller-all-arch: plugin-controller-amd64 plugin-controller-arm64

plugin-controller-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./out/plugin-controller cmd/controller/main.go

plugin-controller-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./out/plugin-controller cmd/controller/main.go

plugin-controller:
	go build -o ./out/plugin-controller cmd/controller/main.go
