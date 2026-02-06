
### Self-Hosted GitLab Instances

For self-hosted GitLab instances, you can omit the organization/group path to scan all accessible projects:

```bash
# Scan all projects you have access to
./scanner --url https://gitlab.company.com --token YOUR_TOKEN

# Or scan a specific group
./scanner --url https://gitlab.company.com/engineering --token YOUR_TOKEN
```
