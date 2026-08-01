# Contributing

Please open an issue before large behavioral changes. Pull requests should keep Panestra CLI fail-open, avoid persistent prompt storage and external communication, and preserve existing user configuration.

Run before submitting:

```bash
gofmt -w cmd internal scripts/generate-demo
go test ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
sh -n scripts/*.sh
```
