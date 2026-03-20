# CI/CD and Quality Gates

## Workflows

- `CI` (`.github/workflows/ci.yml`): runs on every pull request and on pushes to `main`.
- `Release` (`.github/workflows/release.yml`): runs on tag pushes that match `v*` and can be run manually.

## Required Branch Protection Checks

Configure GitHub branch protection for `main` to require:

- `Quality Gates`

Also recommended:

- Require pull request before merging
- Require approvals
- Dismiss stale approvals on new commits

## Local Preflight

Run all gates locally:

```bash
make quality
```
