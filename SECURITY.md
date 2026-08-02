# Security Policy

Security issues should be reported privately through GitHub Security Advisories for `fru180/Panestra-cli`. Do not include secrets or private prompts in a public issue.

Panestra CLI intentionally performs no network requests and stores no prompt history. Reports involving command injection, configuration corruption, unintended prompt persistence, or failure to restore tmux state are especially welcome.

## Public Repository History

As of 2026-08-02, all commit author and committer metadata and all deleted blobs reachable from the repository's public refs were reviewed. The metadata and removed design documents and demo asset are approved to remain public. No credentials, secrets, or private prompts were found, so the published history will not be rewritten.

Future command-line commits must use the contributor's GitHub-provided noreply address. Gitleaks scans the full available Git history on pull requests, pushes to `stg` and `main`, and release tags. If a real secret is discovered, revoke or rotate it before removing it from the repository; rewriting Git history alone is not considered remediation.
