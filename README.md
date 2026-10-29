# Eversor

An open-source Linux cleanup tool for understanding disk usage and reclaiming
space from known caches and obsolete files.

Eversor is in the planning stage. This repository currently contains project
documentation; the application and its infrastructure have not been implemented.

## Planned workflow

```text
eversor dry-run
eversor clean
```

- **Inspect:** build a directory tree showing where storage is being used.
- **Identify:** detect installed tools and known, rebuildable caches, with an
  explanation of each cleanup candidate.
- **Preview:** show the proposed cleanup, estimated reclaimable space, and skipped
  items without deleting files.
- **Clean:** revalidate the plan, use native cleanup commands where appropriate,
  and report what was removed and the space actually recovered.

Large or unfamiliar directories should remain inspection results until there is
an explicit, supported rule for cleaning them. Personal files, project source,
credentials, browser profiles, and databases must be preserved.

Ubuntu is the proposed initial target. The implementation language, architecture,
repository structure for application code, and infrastructure will be decided
during project planning.

## Contributing

Keep each change focused and review its diff before committing. Use Conventional
Commits, validate the change, and push each completed change before starting the
next one. Project-specific working instructions are in [AGENTS.md](AGENTS.md).

## License

Eversor is licensed under the [MIT License](LICENSE).
