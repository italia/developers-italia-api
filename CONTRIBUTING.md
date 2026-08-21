# Contributing

Thanks for taking the time to contribute.

## Reporting a bug

Open an issue describing what you expected, what happened instead and
how to reproduce it. Include the version or commit you are running.

For anything with a security impact do not open an issue. Follow
[SECURITY.md](SECURITY.md) instead.

## Proposing a change

Fork the repository, work on a branch and open a pull request against
`main`. Keep a pull request to one subject. Two unrelated fixes are two
pull requests, and both get reviewed faster than one mixed branch.

Commit messages follow [Conventional
Commits](https://www.conventionalcommits.org/en/v1.0.0/): `feat:`,
`fix:`, `refactor:`, `doc:`, `test:`, `chore:`, `ci:`. Describe the
behavior that changed rather than the code that changed, and explain
in the body why the change was needed.

## Tests

A change to the behavior of the API comes with a test that covers it.
A new endpoint or field gets a test that exercises it, and a bug fix
gets a test that fails before the fix.

A pull request that changes Go code without touching any `*_test.go`
file fails the `tests-added` check. When the change cannot alter
behavior, a rename or a lint fix for example, set the `no-test-needed`
label on the pull request and the check passes.

Tests live next to the code they cover, in `*_test.go`. The YAML
fixtures they load are in `test/testdata/fixtures/`.

## Running the tests

The suite runs against SQLite, so it needs no database:

```sh
go test -race ./...
```

To run against PostgreSQL, the way CI does, start the database from the
compose file and point `DATABASE_DSN` at it:

```sh
docker compose up -d db
DATABASE_DSN='host=localhost user=postgres password=postgres dbname=postgres port=5432' \
  go test -race ./...
```

Before opening a pull request, run the linter as well:

```sh
golangci-lint run
```
