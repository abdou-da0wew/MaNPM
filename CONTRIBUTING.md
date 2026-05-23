# Contributing

Contributions are welcome. This project aims to keep zero external dependencies and a small codebase. Please open an issue before making significant changes.

## Getting started

```
git clone https://github.com/abdou-da0wew/MaNPM.git
cd MaNPM
go build -ldflags="-s -w" -o manpm ./cmd/manpm/
go test ./...
go vet ./...
```

## Pull requests

- Run `go test ./...` and `go vet ./...` before submitting.
- Keep external dependencies at zero. All imports must be from the Go standard library.
- Match the existing code style: terse comments, `context.Context` as first parameter, error wrapping with `fmt.Errorf`.
- Add tests for new functionality. Existing tests are in `pkg/*/*_test.go` and `cmd/manpm/manpm_test.go`.
- One commit per logical change.

## Reporting issues

Open a GitHub issue with the command you ran, the output, and what you expected instead. For feature requests, describe the use case.
