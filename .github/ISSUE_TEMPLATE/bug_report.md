---
name: Bug report
about: Create a report to help us improve
title: ''
labels: ''
assignees: ''

---

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Terraform configuration and steps to reproduce the behavior:
```hcl
# minimal .tf snippet that reproduces the issue
```
1. `terraform init`
2. `terraform plan`/`apply`
3. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Actual behavior**
What happened instead — paste the relevant error/diff output.

**Environment (please complete the following information):**
 - Provider version: [e.g. 26.8.2, from the `required_providers` block or `terraform version`]
 - Terraform version: [output of `terraform version`]
 - OS/Arch: [e.g. darwin/arm64, linux/amd64]

**Debug logs**
If applicable, re-run with `TF_LOG=DEBUG terraform apply` and attach the relevant excerpt (redact any credentials/tokens first).

**Additional context**
Add any other context about the problem here.
