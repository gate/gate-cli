# Release notes extraction

This tool prints only the body of the CHANGELOG section matching an explicit release tag. It never reads Git commit messages. The optional `v` prefix is ignored when matching, so `v1.2.3` matches `[v1.2.3]` or historical `[1.2.3]` headings. Version matching is exact, including prerelease suffixes.

```sh
# Preview one version without publishing or creating a release.
go run ./tools/release-notes --version v0.9.0 --changelog CHANGELOG.md
```

`--version` is required; `--changelog` defaults to `CHANGELOG.md`. Notes go to stdout; errors go to stderr and return exit 1. Missing, duplicate or empty sections, invalid UTF-8 and oversized input files stop extraction. The input file is limited to 1 MiB. Markdown fenced-code examples are preserved and do not create version boundaries.

The tag-triggered release workflow writes these notes to the runner's temporary directory and supplies the file through GoReleaser's `--release-notes` option. An extraction failure stops publication. When retrying an existing release, its description is replaced with the selected section. No automatic commit-message notes are appended.

```sh
# Run the formal unit and workflow-process E2E cases through the repository's Go test entry point.
GOMAXPROCS=2 go test -p 2 ./tools/release-notes -count=1
# Validate Go code and the GitHub Actions workflow.
GOMAXPROCS=2 go vet -p 2 ./tools/release-notes
actionlint .github/workflows/release.yaml
```

The E2E suite executes the actual workflow shell, the Go extraction command, version checks and output file. Only the publishing action is replaced with a local capture process; no GitHub release, upload or real publication is performed. Cases cover exact and historical version headings, the repository changelog, missing/empty/duplicate versions, missing files, unclosed fences, shell-shaped tag input and exclusion of private Git subjects and neighboring versions.
