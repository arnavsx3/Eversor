# Contributing to Eversor

Read `README.md`, `AGENTS.md`, and `docs/architecture.md` before changing code.

1. Work on one focused change with a clear purpose.
2. For providers, follow `docs/providers.md` and include upstream evidence and
   synthetic cache/preservation tests.
3. Run `make check`, inspect `git diff --check`, and stage explicit files.
4. Use Conventional Commits, such as `feat(providers): detect a supported cache`
   or `fix(cleanup): preserve a newly active application`.
5. Commit and push each completed change before starting the next. For outside
   contributions, push to a fork/branch and open a pull request.

Do not commit personal scans, private reports, credentials, downloaded packages,
or generated binaries. Use disposable synthetic homes when testing deletion.
Do not amend published commits or force-push shared history without agreement.
