# AI Agent Usage with Beads

This project uses [Beads](https://github.com/steveyegge/beads), a dependency-aware issue tracker designed for AI-supervised workflows.

## What is Beads?

Beads is a command-line tool that helps track issues and their dependencies, making it perfect for AI agents to:
- Discover and track new work items
- Understand what's ready to work on
- Avoid duplicating effort
- Maintain progress across sessions

## Database Location

The Beads database is located at: `.beads/gitlab-python-scanner.db`

Issues are prefixed with: `gitlab-python-scanner-`
- Example: `gitlab-python-scanner-1`, `gitlab-python-scanner-2`, etc.

## Quick Reference for AI Agents

### Creating Issues
```bash
bd create "Task description"
bd create "Add feature X" -p 0 -t feature
bd create "Write tests" -d "Detailed description" --assignee agent
```

### Finding Work to Do
```bash
bd ready
# Shows issues that are:
# - Status: open
# - No blocking dependencies
# Perfect for claiming next task!
```

### Viewing Issues
```bash
bd list                  # List all issues
bd list --status open    # Filter by status
bd list --priority 0     # Show high-priority items
bd show gitlab-python-scanner-1  # Show details
```

### Managing Dependencies
```bash
bd dep add gitlab-python-scanner-2 gitlab-python-scanner-1
# gitlab-python-scanner-1 blocks gitlab-python-scanner-2
# (gitlab-python-scanner-1 must complete first)

bd dep tree gitlab-python-scanner-1  # Visualize dependencies
bd dep cycles              # Detect circular dependencies
```

### Dependency Types
- **blocks**: Task B must complete before Task A can start
- **related**: Soft connection, informational only
- **parent-child**: Epic/subtask hierarchical relationship
- **discovered-from**: Auto-created when AI discovers related work

### Updating Issues
```bash
bd update gitlab-python-scanner-1 --status in_progress
bd update gitlab-python-scanner-1 --priority 0
bd update gitlab-python-scanner-1 --assignee ai-agent
```

### Closing Issues
```bash
bd close gitlab-python-scanner-1
bd close gitlab-python-scanner-2 --reason "Completed in commit abc123"
```

## Workflow for AI Agents

### 1. Starting a Session
```bash
# See what's ready to work on
bd ready

# Or list all open issues
bd list --status open
```

### 2. Discovering New Work
When you discover a new task while working:
```bash
bd create "New task discovered" -t task

# If it depends on current work:
bd dep add <current-issue> <new-issue>
```

### 3. Claiming and Starting Work
```bash
bd update <issue-id> --status in_progress --assignee goose
```

### 4. Completing Work
```bash
bd close <issue-id> --reason "Implemented feature X"
```

### 5. Reporting Progress
```bash
# Show what's been completed
bd list --status closed

# Show dependency tree
bd dep tree <issue-id>
```

## JSON Output for Programmatic Use

All commands support `--json` flag for machine-readable output:
```bash
bd list --json
bd ready --json
bd show gitlab-python-scanner-1 --json
```

## Git Integration

Beads automatically syncs with git:
- **Auto-export**: Issues exported to JSONL after changes (5s debounce)
- **Auto-import**: JSONL imported when newer than DB (after git pull)
- **Seamless**: Works across machines and team members
- **No manual steps**: Just commit/push/pull as normal

Disable if needed:
```bash
bd create "Task" --no-auto-flush
bd list --no-auto-import
```

## Database Extension

The Beads SQLite database can be extended with custom tables:
```bash
# Located at: .beads/gitlab-python-scanner.db

# Example: Add execution tracking
sqlite3 .beads/gitlab-python-scanner.db "
CREATE TABLE IF NOT EXISTS scan_results (
  id INTEGER PRIMARY KEY,
  issue_id TEXT,
  project_name TEXT,
  python_version TEXT,
  scan_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (issue_id) REFERENCES issues(id)
)
"
```

## Best Practices for This Project

1. **Create issues for major features** before implementing
2. **Use dependencies** to show task ordering
3. **Update status** as work progresses (open → in_progress → closed)
4. **Set priorities** for critical items (0 = highest)
5. **Use bd ready** to find next actionable task
6. **Close with reasons** to maintain history

## Example Workflow

```bash
# Session start
bd ready  # See what's available

# Create new task
bd create "Implement GitLab API authentication" -p 0 -t feature

# Start work
bd update gitlab-python-scanner-1 --status in_progress

# Discover dependency
bd create "Add config file parser" -p 1
bd dep add gitlab-python-scanner-2 gitlab-python-scanner-1
# (Config parser needed before API auth)

# Complete task
bd close gitlab-python-scanner-2 --reason "Implemented viper config"
bd update gitlab-python-scanner-1 --status in_progress
# Continue work...

# End session
bd list --status in_progress  # Show work in progress
```

## Issue Statuses

- `open`: Ready to work on (if no blockers)
- `in_progress`: Currently being worked on
- `blocked`: Cannot proceed due to dependencies
- `closed`: Completed

## Priority Levels

- `0`: Critical (highest priority)
- `1`: High
- `2`: Medium (default)
- `3`: Low
- `4`: Lowest

## Resources

- Beads Repository: https://github.com/steveyegge/beads
- Extension Docs: https://github.com/steveyegge/beads/blob/main/EXTENDING.md
- Issue Tracker: https://github.com/steveyegge/beads/issues

---

*This project was initialized with Beads on 2026-01-26*
