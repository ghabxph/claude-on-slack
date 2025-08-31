# Claude on Slack Project Knowledge Base

## 🚧 **DEVELOPMENT STATUS: ALPHA/BETA**

**⚠️ CRITICAL**: This project is in active development and **NOT production-ready**. 

### **Current Development State**
- **Stability**: Known bugs and ongoing stability issues
- **Testing**: Limited test coverage, manual testing only
- **Breaking Changes**: Expected with each update
- **Production Use**: **NOT RECOMMENDED**

### **Path to Production Readiness**
The project will be considered **stable** when:
1. **TDD Implementation**: Full Test-Driven Development practices
2. **Test Coverage**: Comprehensive automated test suite
3. **CI/CD Pipeline**: Automated testing and deployment
4. **Regression Protection**: New features don't break existing functionality

**Until TDD is fully implemented, expect bugs and instability.**

## 🎯 Implemented Features

### Core Functionality
- [x] **Natural Language Processing**: Direct message processing without command parsing
- [x] **Session Management**:
  - Session tracking with IDs
  - `/session` command for manual control
  - Session resumption with `--resume` flag
  - Display of session IDs in responses

### Permission System
- [x] **Permission Modes**:
  - `/permission` command for mode control
  - Support for `default`, `acceptEdits`, `bypassPermissions`, `plan`
  - Automatic reset to default after each message
  - Help functionality with mode descriptions

### Message Handling
- [x] **Message Queuing**:
  - Queue messages while Claude is processing
  - Combine rapid sequential messages
  - Smart timing-based message combining
- [x] **Image Processing**:
  - Automatic download and analysis of Slack image uploads
  - Support for JPEG, PNG, GIF, and WebP formats
  - Seamless integration with text conversations
  - Intelligent file cleanup and storage management

### User Experience
- [x] **Working Directory Display**:
  - Show current working directory in responses
  - Path tracking and management
- [x] **Response Formatting**:
  - Slack-friendly markdown formatting
  - Deletion of "Thinking..." messages
  - Clean session and directory display
- [x] **Enhanced Error Reporting** (v2.2.4):
  - Smart categorization of Claude Code execution errors
  - Detailed stderr output preservation and display
  - Contextual troubleshooting suggestions for each error type
  - Rich formatted error messages with markdown and emojis
  - Faster debugging with comprehensive error context

### Infrastructure
- [x] **Deployment**:
  - SystemD service setup
  - Redeploy script with service management
  - Automatic tunnel coordination
- [x] **Events API Integration**:
  - Support for both Events API and Socket Mode
  - Proper signature verification
  - Slash command handling

## 🚀 Potential Future Enhancements

### 1. Enhanced Session Management
- [ ] **Session Browser**:
  - Visual session history viewer
  - Search through past conversations
  - Export conversation history
- [ ] **Context Management**:
  - Save/load context presets
  - Share contexts between users
  - Context tagging system

### 2. Code Intelligence
- [ ] **Project Context**:
  - Automatic repository analysis
  - Language/framework detection
  - Project structure understanding
- [ ] **Smart Suggestions**:
  - Code style recommendations
  - Best practices hints
  - Security check suggestions

### 3. Team Collaboration
- [ ] **Shared Workspaces**:
  - Team-level session sharing
  - Collaborative editing sessions
  - Permission inheritance
- [ ] **Review System**:
  - Code review automation
  - PR description generation
  - Commit message suggestions

### 4. Development Workflow
- [ ] **Git Integration**:
  - Branch management
  - Commit organization
  - PR workflow automation
- [ ] **CI/CD Support**:
  - Pipeline suggestions
  - Test coverage analysis
  - Deployment checks

### 5. Security and Compliance
- [ ] **Advanced Access Control**:
  - Fine-grained permissions
  - Role templates
  - Audit logging
- [ ] **Compliance Features**:
  - PII detection
  - License compliance
  - Security scanning

### 6. User Experience
- [ ] **Code Edit Visualization**:
  - Show relevant code changes in chat
  - Diff-style formatting for edits
  - Context-aware change summaries
  - File path and line number references
- [ ] **Interactive Components**:
  - Button actions
  - Drop-down menus
  - Modal dialogs
- [ ] **Rich Responses**:
  - Code block folding
  - Syntax highlighting
  - Inline documentation

### 7. Monitoring and Analytics
- [ ] **Usage Analytics**:
  - Command patterns
  - Response times
  - Error tracking
- [ ] **Performance Metrics**:
  - Resource utilization
  - API latency
  - Queue statistics

### 8. Documentation
- [ ] **Auto-Documentation**:
  - Code comment generation
  - README updates
  - API documentation
- [ ] **Knowledge Base**:
  - FAQ generation
  - Error solutions
  - Best practices

## 🔧 Technical Debt & Improvements

### Short-term
1. **Error Handling**:
   - More granular error types
   - Better error messages
   - Recovery strategies

2. **Testing**:
   - Unit test coverage
   - Integration tests
   - E2E test suite

3. **Configuration**:
   - Better default values
   - Configuration validation
   - Environment templates

### Long-term
1. **Architecture**:
   - Microservices split
   - Event sourcing
   - Queue system

2. **Scalability**:
   - Load balancing
   - Session distribution
   - Cache layer

3. **Maintainability**:
   - Code documentation
   - Modular design
   - Dependency updates

## 📝 Notes

### Current Development Focus

#### 1. Concurrent Multi-Session Support
- **Current Status**: In development
- **Priority**: High
- **Implementation Plan**:
  - Phase 1: Memory-based session management
    - Concurrent session handling
    - Latest session tracking and cleanup
    - Per-session mode settings
    - Session status monitoring (in-progress/completed)
  - Phase 2: PostgreSQL integration
    - Persistent session storage
    - Enhanced session switching
    - Historical session access
    - Rich session metadata

#### 2. Current Limitations
- Session management needs concurrent support
- Message queuing timing could be refined
- Working directory tracking relies on config

### Best Practices
- Always test changes locally
- Follow error handling patterns
- Keep documentation updated
- Use consistent commit messages

## 🔄 **CRITICAL: Release Process & Development Guidelines**

### **🚨 MANDATORY: Update Deployment Message for Every Release**

**Before releasing any new version, you MUST update the deployment notification message:**

1. **File to update**: `internal/notifications/deploy.go`
2. **Function**: `FormatDeploymentMessage()` - Update the `else` block
3. **What to include**:
   - Key features added/changed in this release
   - User-visible improvements  
   - Important technical changes
   - New requirements or setup steps
   - **Link to CHANGELOG.md** for full details

### **Example Update Process:**

```go
// OLD (v2.0.0 message)
} else {
    message += "• PostgreSQL migration and session persistence\n"
    message += "• Enhanced database-backed conversation chains\n"
}

// NEW (v2.1.0 message) 
} else {
    message += "• 🖼️ **Image Processing Support** - Upload and analyze images\n"
    message += "• 🔄 **Natural Integration** - Combine image analysis with text\n"
    message += "• 🧹 **Smart Cleanup** - Automatic file management\n"
}

// Always include CHANGELOG link at the end (using Slack's link format):
message += "\n📋 *Full details*: See <https://github.com/ghabxph/claude-on-slack/blob/main/CHANGELOG.md|CHANGELOG.md>\n"
```

### **Release Checklist:**

- [ ] Update version in `internal/version/version.go`
- [ ] Update CHANGELOG.md with detailed changes
- [ ] **Update deployment message in `internal/notifications/deploy.go`** ⚠️
- [ ] Update README.md if needed
- [ ] Test compilation: `go build -o test ./cmd/slack-claude-bot && rm test`
- [ ] Deploy and test functionality
- [ ] Monitor deployment notifications in Slack

## 🏗️ **Architecture Guidelines**

### **File Organization:**
- `internal/bot/` - Core Slack bot logic and message handling
- `internal/files/` - Image processing, download, and cleanup
- `internal/session/` - Session management (memory + database)
- `internal/claude/` - Claude Code CLI integration
- `internal/notifications/` - Deployment and system notifications

### **Key Design Patterns:**
- **Repository Pattern**: Database abstraction in `internal/repository/`
- **Service Pattern**: Business logic separation
- **Event-Driven**: Slack events trigger appropriate handlers
- **Background Services**: Cleanup and maintenance tasks

### **Dependencies Management:**
- Go 1.21+ required
- PostgreSQL for session persistence
- Slack Go SDK for API integration
- Zap for structured logging

## 🧪 **Testing Strategy**

### **Before Every Commit:**
1. **Compilation Test**: `go build -o test ./cmd/slack-claude-bot && rm test`
2. **Component Tests**: Test new features in isolation
3. **Integration Test**: Deploy and test with real Slack workspace
4. **Permission Test**: Verify Slack OAuth scopes work correctly

### **Image Processing Testing:**
- Test MIME type validation (accept: JPEG, PNG, GIF, WebP)
- Test file size limits (50MB max)
- Test storage directory creation
- Test cleanup service functionality
- Test with real Slack image uploads

## 📋 **Development Workflow**

### **Adding New Features:**
1. **Plan**: Update todo list and create implementation plan
2. **Version**: Increment semantic version appropriately
3. **Code**: Follow existing patterns and conventions
4. **Test**: Verify functionality works as expected
5. **Document**: Update CHANGELOG.md and README.md
6. **Deploy Message**: Update `internal/notifications/deploy.go` ⚠️
7. **Deploy**: Use `./scripts/redeploy.sh` for production updates

### **Permission Requirements:**
When adding features that need new Slack permissions:
- Document required OAuth scopes in README.md
- Update setup instructions
- Test that missing permissions fail gracefully
- Provide clear error messages for permission issues

## 🔒 **Security Considerations**

### **File Handling:**
- Always validate file types and sizes
- Sanitize filenames to prevent directory traversal
- Use temporary storage with automatic cleanup
- Never store files permanently without explicit user consent

### **Slack Integration:**
- Verify all webhook signatures
- Use least-privilege OAuth scopes
- Implement rate limiting for API calls
- Log security-relevant events

## 📝 **Documentation Requirements**

### **Always Update These Files:**
- `CHANGELOG.md` - Detailed change documentation
- `README.md` - User-facing setup and usage instructions  
- `internal/notifications/deploy.go` - Deployment message ⚠️
- `internal/version/version.go` - Semantic version number

### **Documentation Standards:**
- Use semantic versioning (MAJOR.MINOR.PATCH)
- Include setup instructions for new features
- Document all required permissions and environment variables
- Provide troubleshooting guides for common issues

## 🚨 **Critical Reminders**

1. **ALWAYS update the deployment message** when releasing
2. **Test with real Slack workspace** before production deployment
3. **Follow semantic versioning** for all releases
4. **Update documentation** for user-facing changes
5. **Monitor logs** after deployment for any issues

---

**Remember: The deployment message is often the first thing users see about new features. Make it informative and exciting! 🚀**