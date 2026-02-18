# Yor Improvement Suggestions

## 1. Concurrency Model is Self-Defeating
The 10-worker pool looks concurrent but the shared `IntrinsicsLock` serializes all CloudFormation and Serverless parsing. Meanwhile, the workers still cause real problems (data races, fixture corruption). Options:
- **Parse sequentially, tag concurrently** — file parsing (which hits thread-unsafe libraries) runs single-threaded, then tag computation (the CPU-bound part) fans out to workers.
- Or group files by parser type and process each parser's files in a dedicated goroutine, avoiding cross-parser lock contention entirely.

## 2. Tests Mutate Fixtures In-Place
Tests like `Test_TagCFNDir` call `runner.TagDirectory()` against real fixture files and rely on fragile cleanup (we had to add `git checkout` in CI). Every test that tags should:
- Copy fixtures to a temp directory first
- Run against the copy
- This is a one-time refactor that eliminates an entire class of CI failures

## 3. Unmaintained / Thread-Unsafe Dependencies
- `sanathkr/go-yaml` (last commit 2017) — root cause of the concurrent map write crash. Its global mutable state is fundamentally hostile to concurrency.
- `bridgecrewio/goformation/v5` — a Bridgecrew fork no longer under our control.
- Consider replacing these with actively maintained alternatives, or vendoring and patching them.

## 4. Go Module Path
The module is still `github.com/bridgecrewio/yor` in `go.mod`. Since we've forked to `fraserc182/yor`, anyone doing `go install` gets confused. Updating the module path is a large find-and-replace across all imports but makes the fork fully independent.

## 5. Plugin System Fragility
Go's `-buildmode=plugin` requires exact compiler version, flags, and dependency versions. This already caused CI failures (coverage instrumentation mismatch). Alternatives:
- **YAML-based custom taggers** (already supported — promote this as the primary extension mechanism)
- **Hashicorp go-plugin** (RPC-based, version-tolerant)
- **WASM plugins** (emerging Go support)

## 6. Error Handling via `recover()`
Both `goformationParse` and `serverlessParse` use `recover()` to catch panics from third-party libraries. This silently swallows real bugs. Better to fix the root causes (or replace the libraries) and remove the panic recovery.

## 7. Minor but Worthwhile
- **`interface{}` → `any`** — dozens of instances; trivial modernization
- **Structured logging** — currently `fmt.Sprintf` + `logger.Warning/Info`; a structured logger (slog, zerolog) would make debugging much easier
- **CLI framework** — `urfave/cli/v2` is fine but `cobra` has become the Go standard and has richer completion/docs support
