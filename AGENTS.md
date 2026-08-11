# Repository Guidelines

## Preserve the Exercise Before Improving the Code

`private-isu` is an ISUCON practice problem for learning Web application performance tuning. The initial implementations' inefficiency and slowness are part of the exercise, not ordinary technical debt. In this repository, a generally "good" cleanup can be a regression because it removes a bottleneck that participants are meant to discover.

- Unless the user explicitly requests it, do not perform performance tuning, refactoring, architecture changes, or changes to how data is stored.
- Do not helpfully fix known teaching examples such as N+1 queries, SQL without `LIMIT`, images stored in the database, hash calculation through external commands, or parsing templates on every request.
- Distinguish intentionally slow code from code that has actually stopped working after an environment update. If the distinction is unclear, ask the user instead of changing it.
- Make only changes required by the request. Do not include opportunistic fixes, cleanup, formatting, or modernization.

## Cross-Implementation Compatibility

`webapp/` contains Ruby, Go, PHP, Python, and Node.js implementations. They do not need identical performance, but their externally observable features and behavior must remain aligned.

- For changes affecting routes, the database schema, initialization, sessions, cookies, CSRF, redirects, templates, or time handling, inspect the impact on every language implementation.
- Passing tests or the benchmark with one language implementation does not prove that the other implementations, or the intended exercise, remain intact.
- Preserve existing behavior rather than reorganizing an implementation into a preferred handler/repository/service architecture.

## Benchmarker Is Part of the Problem

`benchmarker/` contains the exercise's workload and correctness checks. Its scenarios, concurrency, scoring, and validation behavior are part of the existing problem setting.

- Do not improve, reorganize, or otherwise alter the benchmarker unless explicitly requested.
- The benchmarker is not a complete specification. A passing benchmark alone is not evidence that a behavior change is safe.

## Dependency and Environment Updates

For updates to dependencies, language runtimes, the OS, MySQL, nginx, Docker, or Ansible, make only the minimum changes necessary to restore or maintain compatibility. Do not combine unrelated application improvements or exercise changes with an environment update.

## Repository Layout

- `webapp/{ruby,golang,php,python,node}/` contains the five application implementations.
- `webapp/sql/` contains the schema and initial data used by setup.
- `benchmarker/` contains the Go load generator and correctness checks.
- `provisioning/` contains Ansible configuration for the exercise environment.

## Verified Commands

- `make init`: download the canonical MySQL dump and image fixtures.
- `cd webapp/golang && make`: build the Go application as `webapp/golang/app`.
- `cd webapp/node && npm install && npm run build`: install Node.js dependencies and compile TypeScript. Use `npm run dev` for the development server.
- `cd webapp && docker compose up`: start the local application, nginx, MySQL, and Memcached stack.
- `cd benchmarker && make`: build `benchmarker/bin/benchmarker`.
- `cd benchmarker && ./bin/benchmarker -t "http://localhost:8080" -u ./userdata`: run the benchmark against the local target after initialization.

Use the language-native formatter or tests relevant to the requested change, but do not treat their success as authorization to broaden the change. Never commit secrets or downloaded dumps.
