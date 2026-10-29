# Eversor

An open-source Linux cleanup tool for understanding disk usage and reclaiming
space from known caches and obsolete files.

The architecture is documented in [docs/architecture.md](docs/architecture.md).
Implementation is being developed in focused increments; runnable commands and
their supported cleanup rules will be documented as they are delivered.

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

The initial target is Ubuntu. Eversor uses Go and an application-provider registry
to detect the software installed on each machine. Unsupported folders remain
inspection-only rather than being guessed safe to delete.

## Contributing

Keep each change focused and review its diff before committing. Use Conventional
Commits, validate the change, and push each completed change before starting the
next one. Project-specific working instructions are in [AGENTS.md](AGENTS.md).

## License

Eversor is licensed under the [MIT License](LICENSE).
