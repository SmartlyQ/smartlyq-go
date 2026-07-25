# Contributing

Thanks for your interest in the SmartlyQ Go SDK!

## How this repo works

Most of this SDK is **generated** from the [SmartlyQ OpenAPI spec](https://docs.smartlyq.com):

- `resources_gen.go` - the resource surface, emitted by `scripts/generate`. Never edit by hand.
- `endpoints_gen_test.go` - endpoint tests, emitted by `scripts/generate`. Never edit by hand.
- The README's API Reference section (between the `GENERATED REFERENCE` markers) is emitted by `scripts/generate`.

Hand-written code lives in `client.go`, `client_test.go`, and `scripts/generate/`. Fixes to generated output belong in the generator, or in the OpenAPI spec itself.

```bash
go run ./scripts/generate   # regenerate from openapi.json
go build ./... && go vet ./... && go test ./...
```

## Never commit secrets

This is a **public** repository. Never commit real API keys (`sqk_live_...` / `sqk_test_...`), credentials, tokens, internal hostnames, or customer data. Use placeholders like `sqk_live_xxxxxxxxxxxx` or `YOUR_API_KEY` in examples.

Enable the local pre-commit scan once per clone:

```bash
git config core.hooksPath .githooks
```

CI also runs a gitleaks scan on every push and pull request. If you believe a secret has been exposed, rotate it immediately in your Developer Dashboard.
