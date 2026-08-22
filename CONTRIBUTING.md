# Contributing

Thank you for helping improve `gu`.

## Before opening a pull request

1. Explain the user-visible behavior and failure cases.
2. Add or update tests before changing the implementation when practical.
3. Keep filesystem, network, and external-command boundaries injectable in tests.
4. Do not include credentials, private module contents, local paths, or generated binaries.
5. Run the same checks used by CI:

```bash
gofmt -w <changed-go-files>
go test ./... -count=1
go vet ./...
go test -race ./...
```

For behavior changes, update `README.md` and `CHANGE_LOG.md` as appropriate.

## Pull requests

Keep each pull request focused and describe:

- what changed and why;
- how it was tested;
- any platform-specific behavior or compatibility concerns.

Please use a clear commit prefix such as `fix:`, `test:`, `refactor:`, or `ci:`.
