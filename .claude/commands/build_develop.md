---
description: Tag current commit as v0.0.0-develop and trigger CI build
allowed-tools: Bash(git:*), Bash(gh browse:*)
---

Tag the current git ref as v0.0.0-develop and force push the tag to trigger CI/CD build.

```bash
git add .
git commit -m '-'
git tag -f v0.0.0-develop
git push -f origin v0.0.0-develop
git reset --soft HEAD~1
gh browse --branch v0.0.0-develop
```