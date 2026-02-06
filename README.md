# GitLab Python Version Scanner

A Go CLI tool to scan GitLab projects and detect Python versions across an organization.

## Features

- Scans all projects in a GitLab group/organization
- Detects Python versions from common files (.python-version, pyproject.toml, etc.)
- Real-time console output as projects are scanned
- Optional log file output
- Concurrent scanning for performance
- GitLab API authentication support

## Installation

```bash
go install github.com/meganerd/gitlab-python-scanner@latest
```

Or build from source:

```bash
git clone https://github.com/meganerd/gitlab-python-scanner
cd gitlab-python-scanner
go build -o gitlab-python-scanner
```

## Usage

### Basic Usage

```bash
# Scan all projects in an organization
gitlab-python-scanner --url https://gitlab.com/myorg --token YOUR_TOKEN

# With custom GitLab instance
gitlab-python-scanner --url https://gitlab.company.com/engineering --token YOUR_TOKEN

# Save results to log file
gitlab-python-scanner --url https://gitlab.com/myorg --token YOUR_TOKEN --log results.log
```

### Flags

| Flag | Description | Required | Default |
|------|-------------|----------|---------|
| `--url` | GitLab URL including org/group (e.g., `gitlab.com/myorg`) | Yes | - |
| `--token` | GitLab API token | Yes | - |
| `--log` | Path to log file for output | No | - |
| `--concurrency` | Number of concurrent scans | No | 5 |
| `--timeout` | API timeout in seconds | No | 30 |

### Environment Variables

```bash
export GITLAB_TOKEN=your_token_here
gitlab-python-scanner --url https://gitlab.com/myorg
```

## Example Output

```
Scanning GitLab projects at: https://gitlab.com/myorg
Authentication successful
Found 42 projects in organization

[1/42] project-alpha: Python 3.11.5 (from .python-version)
[2/42] legacy-app: Python 2.7.18 (from setup.py)
[3/42] frontend-app: Python not detected
[4/42] backend-api: Python 3.10.0 (from pyproject.toml)
[5/42] data-pipeline: Python 3.9.16 (from Pipfile)
...

Scan complete: 42 projects, 28 Python projects, 14 non-Python
```

## Python Detection

The scanner checks for Python version information in:

1. `.python-version` - pyenv version file
2. `runtime.txt` - Heroku/platform runtime
3. `pyproject.toml` - Modern Python projects
4. `setup.py` - Traditional Python packages
5. `Pipfile` - Pipenv projects
6. `requirements.txt` - Common dependencies file
7. `.gitlab-ci.yml` - CI/CD configuration
8. `Dockerfile` - Container definitions
9. `tox.ini` - Testing configuration

If none found: "Python not detected"

## Development

### Project Structure

```
gitlab-python-scanner/
├── .beads/                 # Beads issue tracker database
├── cmd/
│   └── scanner/
│       └── main.go         # CLI entry point
├── internal/
│   ├── gitlab/
│   │   ├── client.go       # GitLab API client
│   │   └── auth.go         # Authentication
│   ├── scanner/
│   │   ├── scanner.go      # Main scanning logic
│   │   └── detector.go     # Python version detection
│   └── output/
│       ├── console.go      # Console output
│       └── logger.go       # File logging
├── go.mod
├── go.sum
├── README.md
├── AGENTS.md               # AI agent instructions
└── LICENSE
```

### Building

```bash
go build -o gitlab-python-scanner ./cmd/scanner
```

### Testing

```bash
go test ./...
```

## Task Tracking with Beads

This project uses Beads for issue tracking. See [AGENTS.md](AGENTS.md) for details.

```bash
# See available work
bd ready

# Show all tasks
bd list

# View task details
bd show gitlab-python-scanner-1
```

## Contributing

1. Check available tasks: `bd ready`
2. Claim a task: `bd update <task-id> --status in_progress`
3. Complete work and create PR
4. Close task: `bd close <task-id> --reason "PR #XX merged"`

## License

MIT License - see LICENSE file

## Author

Created by gbjohnso

---

*This project uses AI-assisted development tracked with Beads*
