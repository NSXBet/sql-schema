# Security Policy

## Supported Versions

This project is currently in beta. We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

## Reporting a Vulnerability

We take the security of sql-schema seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### How to Report

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

1. **GitHub Security Advisories** (Preferred)
   - Navigate to the [Security tab](https://github.com/NSXBet/sql-schema/security/advisories)
   - Click "Report a vulnerability"
   - Provide detailed information about the vulnerability

2. **Email**
   - Send an email to security@nsxbet.com (if available)
   - Include detailed information about the vulnerability
   - Use "sql-schema Security Vulnerability" in the subject line

### What to Include

Please include the following information in your report:

- Type of vulnerability (e.g., SQL injection, information disclosure, etc.)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit/direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### Response Timeline

- **Acknowledgment**: Within 48 hours of report submission
- **Initial Assessment**: Within 5 business days
- **Status Updates**: Every 7 days until resolution
- **Fix Timeline**: Depends on severity
  - Critical: Within 7 days
  - High: Within 14 days
  - Medium: Within 30 days
  - Low: Next scheduled release

## Security Scope

### In Scope

This library focuses on schema extraction, comparison, and export for MySQL and PostgreSQL databases. Security considerations include:

- **SQL Injection Prevention**: All database queries use parameterized statements
- **Credential Security**: Database credentials should be handled securely by consuming applications
- **Schema Information Disclosure**: Be cautious when exporting or logging schema information
- **Dependency Vulnerabilities**: We monitor and update dependencies regularly

### Out of Scope (Beta Limitations)

As a beta library, the following are currently out of scope:

- Production database workload handling guarantees
- Backward compatibility guarantees between beta versions
- Performance-related security issues (unless they lead to resource exhaustion)
- Vulnerabilities in third-party dependencies (we will update them, but not provide immediate patches)

## Security Best Practices for Users

When using this library:

1. **Credentials Management**
   - Never hardcode database credentials
   - Use environment variables or secure credential management systems
   - Apply the principle of least privilege for database accounts

2. **Schema Information**
   - Treat exported schema files as sensitive information
   - Be cautious when sharing schema comparisons or reports
   - Consider schema details as potential information disclosure

3. **Database Permissions**
   - Use read-only database accounts when possible
   - Grant only necessary permissions (e.g., `SELECT` on `information_schema`)
   - Avoid using administrative accounts

4. **Network Security**
   - Use encrypted connections (TLS/SSL) to databases
   - Restrict network access to databases
   - Consider using SSH tunnels for remote connections

## Security Updates

Security updates will be published through:

- GitHub Security Advisories
- Release notes with `[SECURITY]` prefix
- Git tags with security patch version bumps

## Known Security Considerations

### Current Beta Status

As this library is in beta (v0.1.x):

- API breaking changes may occur without advance notice
- Security patches may require updating to the latest beta version
- Full security audit has not been completed

### Safe by Design

The library is designed with security in mind:

- All SQL queries use parameterized statements via `QueryContext`
- No dynamic SQL construction from user input
- Read-only operations by default
- No execution of extracted SQL without explicit user action

## Future Security Enhancements

Planned for v1.0 release:

- Comprehensive third-party security audit
- Formal threat modeling documentation
- Security-focused integration tests
- Automated vulnerability scanning in CI/CD
- Security champions program

## Disclosure Policy

When a security vulnerability is confirmed:

1. We will coordinate a fix with the reporter
2. A security advisory will be published on GitHub
3. A patch release will be created
4. Release notes will clearly indicate the security fix
5. Public disclosure will occur after the patch is available

## Contact

For security-related questions or concerns:

- GitHub Security Advisories: https://github.com/NSXBet/sql-schema/security/advisories
- General Security: security@nsxbet.com (if configured)

---

Thank you for helping keep sql-schema and its users safe!
