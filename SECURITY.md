# Security Policy

## Supported Versions

This provider's version tracks the Graphiant platform/SDK release it targets
(see [CHANGELOG.md](CHANGELOG.md)) rather than an independent SemVer
sequence. Security fixes are applied to the latest released version; we
recommend always running the latest release.

| Version | Supported          | Notes                 |
|---------|--------------------|------------------------|
| 26.8.0  | :white_check_mark: | First tagged release   |
| main    | :white_check_mark: | Active development     |

This table will be updated as new versions are released, the same way
[graphiant-sdk-go](https://github.com/Graphiant-Inc/graphiant-sdk-go/blob/main/SECURITY.md)
tracks its own supported release lines.

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security
vulnerability, please follow these steps:

### How to Report

1. **Do NOT** open a public GitHub issue for security vulnerabilities
2. Email security details to: **security@graphiant.com**
3. Include the following information:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if available)
   - Your contact information

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Resolution**: Depends on severity (see below)

### Severity Levels

| Severity | Response Time | Description                                  |
|----------|----------------|-----------------------------------------------|
| Critical | 24-48 hours    | Remote code execution, credential/token leakage |
| High     | 7 days         | Privilege escalation, data exposure           |
| Medium   | 30 days        | Information disclosure, denial of service     |
| Low      | 90 days        | Best practice violations, minor issues        |

### What to Expect

- **Acknowledgment**: You will receive an acknowledgment email within 48 hours
- **Updates**: Regular updates on the status of the vulnerability
- **Credit**: With your permission, we will credit you in security advisories
- **Disclosure**: We will coordinate public disclosure after a fix is available

## Security Best Practices

### Credential Management

This provider authenticates to the Graphiant API with either a static access
token or a username/password pair, resolved by
[`internal/provider/client.go`](internal/provider/client.go). `access_token`
and `password` are marked `Sensitive` in the provider schema (`username` is
not, since it's not a secret on its own), but Terraform still writes resolved
values to plan/state files in plaintext, so treat those files as secrets too.

- **Never commit secrets**: Never commit `.tfvars`, `.tfstate`, or provider
  blocks containing real tokens, usernames, or passwords to the repository.
- **Prefer environment variables** over hardcoding credentials in `.tf` files:
  ```bash
  export GRAPHIANT_ACCESS_TOKEN="..."
  # or
  export GRAPHIANT_USERNAME="..."
  export GRAPHIANT_PASSWORD="..."
  export GRAPHIANT_API_HOST="https://api.graphiant.com"
  ```
- **Protect state**: Terraform state can contain resolved credentials and
  resource attributes. Use a remote backend with encryption at rest and
  restricted access rather than local `.tfstate` files in version control.
- **Rotate credentials**: Regularly rotate API tokens and passwords, and
  revoke tokens that may have been exposed.

### Code Security

- **Input validation**: Resource and data source schemas validate types and
  required/optional fields via terraform-plugin-framework; avoid bypassing
  this by shelling out to the API directly.
- **Error handling**: Diagnostics returned by resources and data sources
  should not leak tokens or other sensitive values from requests/responses.
- **Dependency management**: Keep `graphiant-sdk-go` and
  `terraform-plugin-framework` up to date.
  ```bash
  go list -u -m all   # check for updates
  go get -u ./...      # update dependencies
  go mod tidy
  ```
- **Vulnerability scanning**:
  ```bash
  go install golang.org/x/vuln/cmd/govulncheck@latest
  govulncheck ./...
  ```

### Go-Specific Security

- **Race conditions**: Run `go test -race ./...` for any code that touches
  shared state.
- **TLS**: The `insecure_skip_verify` provider attribute disables certificate
  validation and should only be used against trusted lab/on-prem controllers,
  never in production.
- **Error handling**: Always check and surface errors via `diag.Diagnostics`
  rather than ignoring them.

### CI/CD and Repository Security

- Use encrypted secrets (e.g. GitHub Actions secrets) for any credentials
  needed by CI, never hardcoded values.
- Require review before merging changes that touch authentication
  (`client.go`) or credential handling.

## Security Checklist for Contributors

Before submitting a pull request, ensure:

- [ ] No secrets, tokens, or credentials are committed
- [ ] No `.tfstate` or `.tfvars` files with real values are committed
- [ ] Error messages and diagnostics don't expose sensitive information
- [ ] Dependencies are up to date (`go list -u -m all`)
- [ ] Tests pass, including `go vet ./...`
- [ ] New sensitive attributes are marked `Sensitive: true` in their schema

## Additional Resources

- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [Terraform Plugin Framework: Sensitive data](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#sensitive)
- [Go Vulnerability Database](https://pkg.go.dev/vuln)

## Contact

For security concerns, please contact: **security@graphiant.com**
