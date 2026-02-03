# Security Policy

## Supported Versions

We actively support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability, please follow these steps:

### 1. **Do NOT** create a public GitHub issue

Security vulnerabilities should be reported privately to prevent exploitation.

### 2. Report via Email

Send an email to: **security@verustcode.com** (or use GitHub Security Advisories if available)

Include the following information:

- **Description**: Clear description of the vulnerability
- **Impact**: Potential impact and severity assessment
- **Steps to Reproduce**: Detailed steps to reproduce the issue
- **Affected Versions**: Which versions are affected
- **Suggested Fix**: If you have a suggested fix (optional)

### 3. Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Timeline**: Depends on severity:
  - **Critical**: As soon as possible (typically within 24-48 hours)
  - **High**: Within 1 week
  - **Medium**: Within 2 weeks
  - **Low**: Next scheduled release

### 4. Disclosure Policy

- We will acknowledge receipt of your report within 48 hours
- We will keep you informed of the progress toward resolving the issue
- We will notify you when the vulnerability is fixed
- We will credit you in the security advisory (unless you prefer to remain anonymous)

## Security Best Practices

### For Users

1. **Keep VerustCode Updated**: Always use the latest version
2. **Secure Configuration**: 
   - Use strong passwords (8+ characters, mixed case, digits, special characters)
   - Generate secure JWT secrets (32+ characters): `openssl rand -base64 32`
   - Keep API keys and tokens secure
   - Use environment variables for sensitive configuration
3. **Network Security**:
   - Run behind a reverse proxy (nginx, Traefik) in production
   - Use HTTPS/TLS for all connections
   - Restrict access to admin endpoints
   - Configure firewall rules appropriately
4. **Webhook Security**:
   - Always configure webhook secrets
   - Validate webhook signatures (GitHub, Gitea use HMAC-SHA256)
   - Use token validation for GitLab webhooks
5. **Database Security**:
   - Protect SQLite database files with proper file permissions
   - Regular backups
   - Consider encryption for sensitive data
6. **Access Control**:
   - Limit admin access to trusted users
   - Use strong authentication
   - Regularly rotate credentials

### For Developers

1. **Dependency Management**:
   - Regularly update dependencies: `go mod tidy`, `npm audit`
   - Review security advisories for dependencies
   - Use Dependabot for automated updates
2. **Code Security**:
   - Follow secure coding practices (see `.cursor/rules/project.mdc`)
   - Never commit secrets or credentials
   - Validate and sanitize all inputs
   - Use parameterized queries for database operations
   - Implement proper error handling (don't leak sensitive info)
3. **Authentication & Authorization**:
   - All API endpoints require authentication by default
   - Use JWT for stateless authentication
   - Implement proper authorization checks
   - Follow principle of least privilege
4. **Input Validation**:
   - Validate all user inputs
   - Sanitize file paths to prevent path traversal
   - Validate webhook payloads
   - Check YAML/JSON parsing errors

## Known Security Features

### Authentication

- **JWT-based authentication** for admin/API endpoints
- **Password hashing** using bcrypt
- **Token expiration** configurable (default: 24 hours)
- **No default credentials** - password must be set on first launch

### Webhook Security

- **HMAC-SHA256 signature verification** for GitHub and Gitea webhooks
- **Token validation** for GitLab webhooks
- **Secret configuration** required for all webhook providers

### API Security

- **CORS protection** with configurable allowed origins
- **Error message sanitization** in production mode (debug mode disabled)
- **Request ID tracking** for security auditing
- **Rate limiting** (consider implementing for production)

### Data Protection

- **Sensitive data masking** in logs and API responses
- **No hardcoded secrets** - all secrets via environment variables or config
- **SQL injection protection** via GORM parameterized queries
- **Path traversal protection** in file operations

## Security Updates

Security updates will be:

1. Released as patch versions (e.g., 1.0.1)
2. Documented in release notes
3. Tagged with security advisory labels
4. Prioritized over feature releases

## Security Audit

We recommend:

- Regular security audits of dependencies
- Code security reviews for major changes
- Penetration testing for production deployments
- Monitoring security advisories for Go, Node.js, and dependencies

## Contact

For security-related questions or concerns:

- **Email**: security@verustcode.com
- **GitHub Security Advisories**: Use GitHub's security advisory feature if available

## Acknowledgments

We appreciate responsible disclosure of security vulnerabilities. Contributors who report security issues will be credited (unless they prefer anonymity).

Thank you for helping keep VerustCode secure!
