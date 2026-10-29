# Working on Eversor

Read `README.md` and any agreed design documents before making changes. The
architecture is documented in `docs/architecture.md`; keep implementation and
infrastructure work within the user's requested scope.

## Commit and push workflow

The owner requires a commit and push after every completed, coherent change.

1. Make one focused change with its related documentation and checks.
2. Run the checks appropriate to that change. Do not add implementation-mirroring
   tests or run unrelated builds for documentation-only edits.
3. Review `git diff` and `git diff --check`.
4. Stage explicit paths and inspect the staged diff. Keep unrelated changes out.
5. Commit with a Conventional Commit message, then push the commit before starting
   the next change.
6. Verify that the push succeeded and the working tree is clean. Report a blocked
   push or failed check honestly rather than treating the change as complete.

Use a short, imperative subject with an appropriate type, such as:

- `feat(scanner): detect known cache directories`
- `fix(cleanup): preserve active build outputs`
- `docs: describe the cleanup workflow`
- `chore: initialize Eversor repository`

Treat the related files for a single change as one commit. Do not create commits
for individual keystrokes or mix independent changes. Do not amend published
commits, rewrite shared history, or force-push unless the owner explicitly asks.

## Repository hygiene

- Never commit credentials, personal filesystem reports, private user data,
  downloaded dependencies, or generated build output.
- Keep documentation accurate about implemented behavior and planned features.
- Use synthetic paths and disposable fixtures when testing cleanup behavior.
- Keep scan and dry-run operations read-only. Cleanup actions must have explicit
  boundaries and preserve personal files, source code, and databases.
- Introduce source directories, dependencies, checks, and infrastructure when
  their purpose has been agreed during planning.
