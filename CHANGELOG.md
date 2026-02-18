# Changelog — Fork Improvements

Summary of changes made after forking from `bridgecrewio/yor` to `fraserc182/yor`.

## CI/CD Modernisation
- Removed all Bridgecrew-specific CI jobs (Docker Hub publish, Jenkins webhooks, GitHub dispatches, terragoat clone, checkov secrets scan)
- Updated all GitHub Actions runners from Bridgecrew self-hosted to `ubuntu-latest`
- Upgraded Go version from 1.19 to 1.26 across all workflows
- Upgraded `reviewdog/action-golangci-lint` to v2.8.0 (golangci-lint v2)
- Replaced unmaintained `edplato/trufflehog-actions-scan` with `trufflesecurity/trufflehog@v3.93.3`
- Switched goreleaser from `secrets.PAT` to `secrets.GITHUB_TOKEN`
- Removed homebrew tap dependency from goreleaser
- Added `windows/arm` to goreleaser ignore list (unsupported GOOS/GOARCH pair)

## Bug Fixes

### Data Race in Git Blame (go-git Repository)
The `GetPreviousBlameResult` function accessed a non-thread-safe `go-git/v5 Repository` object from multiple worker goroutines. Fixed by caching the previous commit (`HEAD~1`) during `GitService` initialisation, before concurrent processing begins.

### Data Race in YAML Intrinsics (goformation / sanathkr/go-yaml)
The CloudFormation and Serverless parsers each had their own mutex (`goformationLock`, `slsParseLock`), but both call `intrinsics.ProcessYAML` which writes to a process-global tag-unmarshaler map. Concurrent workers holding different locks crashed with `fatal error: concurrent map writes`. Fixed by replacing both local mutexes with a single shared `utils.IntrinsicsLock`. Also protected the previously unguarded `sanathyaml.YAMLToJSON` call in CloudFormation's `ValidFile`.

### CI Plugin Version Mismatch
`go test -covermode=count` recompiles packages with coverage instrumentation, making them binary-incompatible with pre-built `.so` plugins. Fixed by running plugin-dependent tests (runner tests) separately without coverage flags.

### CI Test Fixture Contamination
Runner tests call `TagDirectory()` which modifies test fixture files on disk. When tests were split into two `go test` invocations, modified fixtures persisted between runs. Fixed by adding `git checkout -- tests/` between the two invocations.

### gosec VCS Stamping Error
Added `GOFLAGS: "-buildvcs=false"` to the gosec CI job to prevent build failures in shallow clones.

## Code Quality
- Fixed 8 staticcheck issues (redundant nil checks, De Morgan's law, tagged switches, embedded field selectors)
- Suppressed revive exported-comment rules in `.golangci.yaml` (50 pre-existing issues)
- Configured errcheck exclusion for test cleanup code (`defer os.RemoveAll`, `os.Unsetenv`)
- Removed deprecated `--skip-dirs` flag from golangci-lint invocations
- Updated pre-commit hooks: removed golint (deprecated), migrated golangci-lint to v2

## Repository Cleanup
- Updated Dockerfile to reference `fraserc182/yor`
- Updated `.goreleaser.yml` to reference `fraserc182/homebrew-tap` (since removed)
- Updated `docs/_config.yml` to reference `fraserc182`
- Updated integration tests to run against yor repo instead of terragoat
