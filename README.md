# VerustCode

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/verustcode/verustcode)

**DSL-Driven AI-Powered Code Review Webhook Service**

VerustCode is a flexible code review automation platform that uses YAML-based DSL to orchestrate multiple specialized AI reviewers. Unlike traditional tools that apply one-size-fits-all analysis, VerustCode lets you define custom review pipelines with different focus areas, severity filters, and output destinations.

---

## ✨ Key Features

- 🎯 **DSL-Driven Configuration**: Declarative YAML to define review rules, focus areas, and constraints
- 🔄 **Multi-Reviewer Pipeline**: Run multiple specialized reviewers (security, performance, quality) sequentially
- 🎨 **Flexible Output**: Console, files (Markdown/JSON), PR comments, or custom webhooks
- 🤖 **Pluggable AI Backends**: Support for Cursor, Gemini, or custom AI CLIs
- 📊 **Web Dashboard**: Real-time monitoring, queue management, and configuration
- 🔐 **Secure by Default**: JWT authentication, webhook signature validation, password protection

---

## 📋 Prerequisites

Before getting started, ensure you have the following installed on your local machine:

- **Node.js** (v16 or higher) - For frontend development
- **Go** (1.21 or higher) - For backend compilation
- **Git** - For version control
- **AI CLI Tool** - At least one of the following:
  - [Cursor CLI](https://cursor.com/cn/docs/cli/overview) - Recommended
  - [Gemini CLI](https://geminicli.com/)
  - [Qoder CLI](https://docs.qoder.com/cli/quick-start)

---

## 🚀 Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/verustcode/verustcode.git
cd verustcode
make build

# Start server (interactive setup on first run)
./verustcode serve --check
```

### Configure Webhooks

Add webhook to your repository:

- **GitHub**: `http://your-server:8091/api/v1/webhooks/github`
- **GitLab**: `http://your-server:8091/api/v1/webhooks/gitlab`
- **Gitea**: `http://your-server:8091/api/v1/webhooks/gitea`

### Access Dashboard

Navigate to `http://localhost:8091/admin` and set your admin password on first launch.

![VerustCode Dashboard](docs/dashboard.png)

---

## 📝 DSL Configuration

### Simple Example

```yaml
version: "1.0"

rule_base:
  agent: cursor
  output:
    format: markdown
    channels:
      - type: comment
        overwrite: true

rules:
  - id: code-quality
    description: Reviews business logic and code quality
    goals:
      areas:
        - business-logic
        - edge-cases
        - error-handling
      avoid:
        - Pure formatting issues
        - Subjective preferences
```

### Advanced: Multi-Reviewer with Custom Output

```yaml
version: "1.0"

rule_base:
  agent: cursor
  constraints:
    scope_control:
      - Review **only code changed in this PR**
    focus_on_issues_only: true

rules:
  # Quality reviewer - Markdown output
  - id: code-quality
    goals:
      areas: [business-logic, edge-cases, concurrency]
    output:
      format: markdown
      channels:
        - type: comment

  # Security reviewer - JSON output with strict filtering
  - id: security
    reference_docs:
      - docs/security-guidelines.md
    goals:
      areas: [security-vulnerabilities, injection-attacks, authentication]
    constraints:
      severity:
        min_report: medium
    output:
      format: json
      schema:
        # Extra fields extend the base schema's findings with additional fields
        extra_fields:
          - name: vulnerability_type
            type: string
            description: "Type of security vulnerability"
            required: true
          - name: cve_id
            type: string
            description: "CVE identifier if applicable"
      style:
        tone: strict
      channels:
        - type: file
          dir: reports
        - type: webhook
          url: https://security-dashboard.example.com/api/reviews
```

### 🎯 DSL Highlights

**Three-Level Execution Hierarchy:**

```
Review → Rule → Run
   ↓       ↓      ↓
  PR   Quality  Model-1
   ↓       ↓      ↓
       Security Model-2
                  ↓
                Model-3 (merge)
```

- **Review Level**: Triggered by PR/MR webhook
- **Rule Level**: Multiple specialized reviewers (security, quality, performance)
- **Run Level**: Multi-run with different models + consensus merge (optional)

**Key Capabilities:**

- **Inheritance**: `rule_base` defines defaults, rules override as needed
- **Reference Docs**: Attach project guidelines for context-aware review
- **Severity Filtering**: `min_report` to reduce noise
- **Focus Control**: `focus_on_issues_only` to skip explanations
- **Custom Schemas**: Define structured JSON output format
- **Multi-Channel Output**: Send results to multiple destinations simultaneously

---

## 🔧 Configuration

### Configuration Overview

VerustCode uses a two-tier configuration system:

| Source | Purpose | Description |
|--------|---------|-------------|
| `bootstrap.yaml` | System settings | Server, database, logging, telemetry (requires restart) |
| Settings page | Runtime settings | Git providers, agents, review, report, notifications (takes effect immediately) |

**Getting Started**:
1. Copy `config/bootstrap.example.yaml` to `config/bootstrap.yaml`
2. Start the server: `verustcode serve`
3. Configure runtime settings via the admin web interface

### Getting API Keys

- **GitHub**: Settings → Developer settings → Personal access tokens (`repo` scope for private repos)
- **GitLab**: Settings → Access Tokens (`api`, `read_repository`, `write_repository`)
- **Gitea**: Settings → Applications → Access Tokens (`repo`, `issue`)
- **Cursor**: [cursor.com](https://cursor.com)
- **Gemini**: [Google AI Studio](https://makersuite.google.com/app/apikey)

---

## 🛡️ Security

- **JWT Authentication**: All admin/API endpoints require authentication
- **Webhook Validation**: HMAC-SHA256 signature verification (GitHub, Gitea) or token validation (GitLab)
- **Password Policy**: 8+ characters with mixed case, digit, and special character
- **No Default Credentials**: Password must be set via web UI on first launch

---

## 📚 Documentation

- **[API Reference](docs/API.md)**: Complete API documentation
- **[Architecture](docs/ARCHITECTURE.md)**: System architecture and design
- **[Contributing Guide](CONTRIBUTING.md)**: How to contribute to VerustCode
- **[Security Policy](SECURITY.md)**: Security vulnerability reporting
- **Bootstrap Configuration**: See `config/bootstrap.example.yaml`
- **DSL Reference**: See `config/reviews/default.example.yaml`
- **Development**: Run `make dev` for debug mode

---

## ❓ FAQ

### How does VerustCode differ from other code review tools?

VerustCode uses a DSL-driven approach that allows you to:
- Define multiple specialized reviewers (security, quality, performance) in a single configuration
- Customize focus areas and severity filters per reviewer
- Output results to multiple channels simultaneously (PR comments, files, webhooks)
- Use different AI models for different reviewers
- Configure review behavior declaratively without code changes

### What AI providers are supported?

Currently supported:
- **Cursor**: Cursor Agent CLI integration
- **Gemini**: Google Gemini API
- **Qoder**: Qoder CLI integration

More providers can be added by implementing the `base.Agent` interface.

### Can I use VerustCode without webhooks?

Yes! You can trigger reviews via:
- REST API (`POST /api/v1/reviews`)
- Web dashboard
- CLI (if implemented)

### How do I configure multiple reviewers?

Define multiple rules in your DSL configuration:

```yaml
rules:
  - id: security
    goals:
      areas: [security-vulnerabilities, injection-attacks]
  - id: quality
    goals:
      areas: [business-logic, edge-cases, error-handling]
  - id: performance
    goals:
      areas: [performance, efficiency, memory-usage]
```

### Is VerustCode suitable for large codebases?

Yes, VerustCode:
- Clones repositories locally for analysis
- Processes changes incrementally (PR-based reviews)
- Supports concurrent reviews with configurable workers
- Uses efficient data structures and caching

### How do I report security vulnerabilities?

Please see our [Security Policy](SECURITY.md) for details on reporting security issues privately.

### Can I self-host VerustCode?

Yes! VerustCode is designed for self-hosting:
- Single binary deployment
- SQLite database (no external dependencies)
- Docker support
- Configurable via YAML and web UI

---

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

- 🐛 [Report a Bug](https://github.com/verustcode/verustcode/issues/new?template=bug_report.md)
- 💡 [Request a Feature](https://github.com/verustcode/verustcode/issues/new?template=feature_request.md)
- 📖 [Improve Documentation](https://github.com/verustcode/verustcode/issues)

---

## 📄 License

[MIT License](LICENSE) - see LICENSE file for details.

---

## 🔗 Links

- [GitHub Repository](https://github.com/verustcode/verustcode)
- [Issue Tracker](https://github.com/verustcode/verustcode/issues)
- [Discussions](https://github.com/verustcode/verustcode/discussions)
