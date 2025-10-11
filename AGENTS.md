# Repository Guidelines

## Project Structure & Module Organization
- `webapp/` hosts language ports (`golang/`, `ruby/`, `php/`, `python/`, `node/`); focus changes on one port and mirror fixes only when needed.
- `webapp/sql/` holds schema and fixtures consumed by `make init`; version migrations and data patches here.
- `benchmarker/` contains the Go load tester, while `provisioning/` tracks Ansible roles for operational parity.

## Build, Test, and Development Commands
- `make init`: download the canonical MySQL dump and image fixtures.
- `cd webapp/golang && make`: build the Go binary to `./app`; runtime config comes from `ISUCONP_*` env vars.
- `cd webapp/node && npm install && npm run build`: install and transpile the TypeScript service; use `npm run dev` for hot reload.
- `cd webapp && docker compose up`: run nginx, the app tier, MySQL, and Memcached locally.
- `cd benchmarker && make && ./bin/benchmarker -t "http://localhost:8080" -u ./userdata`: rebuild and execute the scorer after optimizations.

## Coding Style & Naming Conventions
- Go: enforce `go fmt ./...`; group handlers, repositories, and services by feature under `internal/`.
- Ruby/Python/PHP: follow existing idioms (Ruby 2-space indent, Python PEP 8, PHP PSR-12); avoid mixing tabs and spaces.
- Node/TypeScript: ES modules with strict typing; PascalCase classes, camelCase identifiers; run `tsc` before pushing.
- Shared SQL, template, and static asset filenames stay snake_case; keep uploaded media names space-free.

## Testing Guidelines
- Supplement changes with language-native unit tests when touching core logic (`go test ./...`, `pytest`, `bundle exec rspec`, `npm test`) even if suites are sparse.
- Benchmark every performance tweak and note the score delta in your PR description.
- For schema updates, load `webapp/sql/dump.sql` into local MySQL and smoke-test login plus timeline flows.

## Commit & Pull Request Guidelines
- Use imperative commit subjects (`optimize timeline query`); dependency bumps often follow `chore(deps):`/`fix(deps):` patterns.
- Keep commits narrowly scoped so regressions are traceable; include schema files and generated assets with the change.
- PRs summarize the bottleneck, the fix, benchmark results, and env var updates; link issues when applicable.
- Attach screenshots or shell snippets for benchmark output or UI adjustments and flag manual deploy steps.

## Security & Configuration Tips
- Never commit secrets or dumps; reference required env vars (`ISUCONP_DB_HOST`, etc.) instead.
- Rotate credentials through `provisioning/` and keep sensitive data vaulted.
- Enforce login checks on new endpoints and prefer filesystem-backed images over raw BLOB responses when optimizing.
