# Pull Request

## 📋 Summary

<!-- Provide a brief description of the changes in this PR -->

## 🎯 Type of Change

Please check the type of change your PR introduces:

- [ ] 🐛 Bug fix (non-breaking change that fixes an issue)
- [ ] ✨ New feature (non-breaking change that adds functionality)  
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📝 Documentation update
- [ ] 🔧 Refactoring (no functional changes)
- [ ] ⚡ Performance improvement
- [ ] 🧪 Test coverage addition/improvement
- [ ] 🏗️ Architecture change
- [ ] 🔒 Security fix

## 🔍 Related Issue

<!-- If this PR addresses an issue, please link it -->
Fixes #(issue_number)
Closes #(issue_number)
Related to #(issue_number)

## 📖 Description

<!-- Provide a more detailed description of the changes -->

### What changed?
- 
- 
- 

### Why was this change necessary?
- 

### How does this change improve the project?
- 

## 🧪 Testing

### Manual Testing Performed
- [ ] Local development testing completed
- [ ] Service compilation successful (`go build -o test ./cmd/slack-claude-bot && rm test`)
- [ ] Slack integration tested (if applicable)
- [ ] Database migrations tested (if applicable)
- [ ] Feature flags tested (if applicable)

### Test Coverage
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] No tests required (documentation/config changes only)

### Test Instructions
<!-- Provide step-by-step instructions for testing this PR -->
1. 
2. 
3. 

## 📝 Documentation

### Updated Documentation
- [ ] README.md updated (if user-facing changes)
- [ ] CHANGELOG.md updated with changes
- [ ] CLAUDE.md updated (if development process changes)
- [ ] Code comments added/updated
- [ ] API documentation updated (if applicable)

### Documentation Changes Required
<!-- List any documentation that needs to be updated -->
- [ ] Setup instructions
- [ ] Configuration changes  
- [ ] New feature usage
- [ ] Breaking change migration guide
- [ ] Other: ________________

## 🚀 Deployment

### Pre-Deployment Checklist
- [ ] Version updated in `internal/config/config.go` (AppVersion field) - **Single Source of Truth**
- [ ] Deployment message updated in `internal/bot/service.go` (changes array)
- [ ] Environment variables documented (if new ones added)
- [ ] Migration files created (if database changes)
- [ ] Backwards compatibility maintained or migration path provided

### Deployment Notes
<!-- Any special deployment considerations -->
- 
- 

## 🔄 Breaking Changes

<!-- If this is a breaking change, describe the impact and migration steps -->

### What breaks?
- 

### Migration Steps
1. 
2. 
3. 

### Rollback Plan
- 

## 📸 Screenshots/Examples

<!-- If applicable, add screenshots or code examples showing the changes -->

### Before
```
<!-- Previous behavior/output -->
```

### After  
```
<!-- New behavior/output -->
```

## 🔍 Code Review Checklist

### Author Checklist
- [ ] Self-reviewed the code changes
- [ ] Code follows project style and conventions
- [ ] No debugging code/console.logs left in
- [ ] Error handling implemented appropriately
- [ ] Security considerations addressed
- [ ] Performance impact considered
- [ ] Dependencies are necessary and up-to-date

### Reviewer Focus Areas
<!-- Guide reviewers on what to focus on -->
- [ ] Logic correctness
- [ ] Error handling
- [ ] Security implications  
- [ ] Performance impact
- [ ] Test coverage adequacy
- [ ] Documentation completeness

## 🏗️ Architecture Impact

<!-- If this PR affects the overall architecture -->

### Components Affected
- [ ] Bot service (`internal/bot/`)
- [ ] Claude executor (`internal/claude/`)
- [ ] Session management (`internal/session/`)
- [ ] Database layer (`internal/repository/`)
- [ ] Configuration (`internal/config/`)
- [ ] Notifications (`internal/notifications/`)
- [ ] File handling (`internal/files/`)
- [ ] Other: ________________

### Database Changes
- [ ] New tables/columns
- [ ] Schema modifications
- [ ] Migration files included
- [ ] Backwards compatibility maintained

## 🚨 Risk Assessment

### Risk Level
- [ ] 🟢 Low (documentation, minor fixes)
- [ ] 🟡 Medium (new features, refactoring)
- [ ] 🔴 High (breaking changes, major architecture changes)

### Potential Issues
<!-- What could go wrong and how to handle it -->
- 

### Rollback Strategy
<!-- How to quickly revert if issues arise -->
- 

## 📅 Release Planning

### Target Release
- [ ] Next patch release (x.y.Z)
- [ ] Next minor release (x.Y.0) 
- [ ] Next major release (X.0.0)
- [ ] Hotfix release

### Release Readiness
- [ ] Ready for immediate release
- [ ] Needs additional testing
- [ ] Requires coordination with other changes
- [ ] Blocked by: ________________

---

## 📋 Additional Notes

<!-- Any other information that would be helpful for reviewers -->

### Dependencies
<!-- List any PRs or external changes this depends on -->
- 

### Follow-up Tasks
<!-- What should be done after this PR is merged -->
- [ ] 
- [ ] 
- [ ] 

---

**🚧 Development Status Reminder**: This project is in active development (Alpha/Beta). All changes should prioritize stability and maintain backwards compatibility where possible.