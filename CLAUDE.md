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

### Memory Management & Auto-Compaction (v2.10.0)
- [x] **Intelligent Auto-Compaction System**:
  - Automatic conversation compression at 90% token capacity (162k/180k tokens)
  - Seamless background processing without conversation interruption
  - Smart threshold detection with configurable levels (critical/recommended/consider/none)
  - Zero-downtime memory optimization preserving complete conversation history
- [x] **Reusable Compaction Engine**:
  - Unified architecture for both manual and automatic compaction triggers
  - Modular design with reusable core methods (`GetCompactionStatus`, `PerformSessionCompaction`, `TriggerAutoCompaction`)
  - Consistent error handling and comprehensive monitoring across all operations
  - Flexible integration allowing programmatic compaction from any system component
- [x] **Advanced Memory Analytics**:
  - Enhanced `/compact status` with detailed token usage percentages and recommendations
  - Historical tracking with archive timestamps and compaction sequence numbers
  - Capacity planning showing remaining tokens and projected compaction timing
  - Color-coded status indicators for immediate visual assessment
- [x] **Administrative Controls**:
  - `/compact status` - Comprehensive memory analytics with detailed recommendations
  - `/compact now` - Immediate manual compaction with real-time progress notifications
  - `/compact threshold <tokens>` - Configurable auto-compaction triggers (50k-500k range)
  - `/compact help` - Complete command documentation and usage guidelines
  - Admin-only safety controls preventing accidental system-wide impacts
- [x] **Dual-Memory Architecture**:
  - Short-term memory for active conversation steps with real-time token tracking
  - Long-term memory for compressed conversation archives with perfect history preservation
  - PostgreSQL-backed persistence with optimized indexing for performance
  - Session isolation ensuring independent compaction operations

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

- [ ] Update version in `internal/config/config.go` (AppVersion field) - **Single Source of Truth**
- [ ] Update CHANGELOG.md with detailed changes
- [ ] **Update deployment message in `internal/bot/service.go` (changes array)** ⚠️
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
- `internal/bot/service.go` - Deployment message changes array ⚠️
- `internal/config/config.go` - Application version (AppVersion field) - **Single Source of Truth**

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

## 🧪 **TESTING PLAN & STABILITY OBSERVATIONS - v2.10.0 AUTO-COMPACTION**

### **🔍 Critical Testing Areas**

#### **1. Auto-Compaction Triggering (HIGH PRIORITY)**
- [ ] **Token Threshold Testing**: Verify auto-compaction triggers exactly at 162k tokens (90% of 180k)
- [ ] **Multiple Session Isolation**: Ensure compaction in one session doesn't affect others
- [ ] **Concurrent Processing**: Test auto-compaction during active tool execution (Read, Write, Bash)
- [ ] **Threshold Edge Cases**: Test behavior at 161k, 162k, and 163k token boundaries
- [ ] **Rapid Token Growth**: Verify handling of multiple quick tool calls near threshold

#### **2. Memory Data Integrity (CRITICAL)**
- [ ] **Complete History Preservation**: Verify no conversation data is lost during compaction
- [ ] **Step Sequence Integrity**: Ensure step order is maintained in long-term archives
- [ ] **Token Count Accuracy**: Validate token estimation matches actual usage
- [ ] **Session Linkage**: Verify parent/child session relationships remain intact
- [ ] **Metadata Preservation**: Confirm all tool inputs, outputs, and thinking context are archived

#### **3. Performance & Resource Management (HIGH PRIORITY)**
- [ ] **Memory Usage**: Monitor RAM consumption during compaction of large conversations
- [ ] **Database Performance**: Test PostgreSQL performance with multiple concurrent compactions
- [ ] **Background Processing**: Verify auto-compaction doesn't block user interactions
- [ ] **Timeout Handling**: Test 10-minute compaction timeout behavior
- [ ] **Resource Cleanup**: Ensure proper cleanup after successful/failed compactions

#### **4. Error Handling & Recovery (CRITICAL)**
- [ ] **Database Connection Failures**: Test compaction behavior during DB outages
- [ ] **Compaction Service Failures**: Verify graceful handling of compaction errors
- [ ] **Partial Failures**: Test recovery when compaction partially completes
- [ ] **Concurrent Access**: Test handling of simultaneous compaction attempts on same session
- [ ] **Corrupted Data**: Verify error handling for malformed memory records

#### **5. User Experience & Interface (MEDIUM PRIORITY)**
- [ ] **Command Responsiveness**: Verify `/compact` commands work during auto-compaction
- [ ] **Status Accuracy**: Test `/compact status` during active compaction
- [ ] **Notification Delivery**: Ensure Slack notifications reach correct channels
- [ ] **Error Messages**: Verify user-friendly error messages for common failures
- [ ] **Admin Controls**: Test admin-only restrictions for all compact commands

### **📊 Monitoring & Observability Requirements**

#### **Key Metrics to Track**
- **Auto-Compaction Frequency**: How often sessions reach 162k token threshold
- **Compaction Duration**: Average time for compression operations
- **Memory Usage Patterns**: RAM and disk usage during compaction cycles
- **Database Query Performance**: Slow query detection for memory operations
- **Error Rates**: Failed compaction percentage and common failure modes

#### **Log Analysis Focus Areas**
- **Token Growth Patterns**: How quickly sessions approach thresholds
- **Compaction Trigger Events**: Frequency and timing of automatic triggers
- **Performance Bottlenecks**: Slow operations during memory processing
- **Error Clustering**: Patterns in compaction failures
- **Resource Utilization**: CPU, memory, and I/O during background processing

### **🚨 Stability Red Flags to Watch For**

#### **Immediate Action Required If:**
- **Memory Leaks**: Continuously growing RAM usage during compaction cycles
- **Database Locks**: Long-running queries blocking other operations
- **Failed Compactions**: >5% failure rate for auto-compaction attempts
- **Data Loss**: Any indication of missing conversation history after compaction
- **Performance Degradation**: User-visible slowdowns during background compaction

#### **Warning Signs Requiring Investigation:**
- **Uneven Token Distribution**: Some sessions growing much faster than others
- **Compaction Timing Issues**: Auto-compaction triggering too early/late
- **Notification Failures**: Missing or delayed Slack notifications
- **Session Confusion**: Parent/child session ID resolution errors
- **PostgreSQL Performance**: Query times increasing over time

### **🔧 Recommended Testing Scenarios**

#### **Scenario A: Heavy Usage Simulation**
1. Create multiple long conversations (>150k tokens each)
2. Run concurrent tool operations (Read, Write, Bash, Grep)
3. Monitor auto-compaction triggers and performance
4. Verify all sessions remain functional post-compaction

#### **Scenario B: Edge Case Testing**
1. Test with conversations containing large tool outputs (>50MB files)
2. Rapid-fire tool execution near token thresholds
3. Intentional database disconnections during compaction
4. Multiple admin users triggering manual compaction simultaneously

#### **Scenario C: Long-Term Stability**
1. Run continuous conversations for 24+ hours
2. Monitor memory usage trends and compaction patterns
3. Test system behavior after multiple compaction cycles
4. Verify no degradation in response times or accuracy

### **📋 Production Deployment Checklist**

#### **Before Enabling Auto-Compaction in Production:**
- [ ] Complete all critical and high-priority testing scenarios
- [ ] Establish baseline performance metrics
- [ ] Set up comprehensive monitoring and alerting
- [ ] Create rollback procedures for emergencies
- [ ] Document troubleshooting procedures for common issues
- [ ] Train support team on new compaction features and commands

#### **Post-Deployment Monitoring (First 48 Hours):**
- [ ] Monitor auto-compaction trigger frequency and success rates
- [ ] Track database performance and query execution times
- [ ] Verify user experience remains unaffected during background operations
- [ ] Check log aggregation for any unexpected error patterns
- [ ] Validate notification delivery and admin command functionality

**⚠️ CRITICAL**: Do not deploy to production until ALL critical testing areas have been verified and baseline metrics established. The auto-compaction system directly affects conversation data integrity and must be thoroughly validated.

## 🔧 **POTENTIAL BUG FIXES & IMPROVEMENTS - v2.10.0+**

### **🐛 Known Issues to Monitor**

#### **High Priority Fixes Needed**
- [ ] **Token Estimation Accuracy**: Current 4-chars-per-token estimation may be inaccurate for code-heavy conversations
  - **Impact**: Could trigger compaction too early/late
  - **Solution**: Implement more sophisticated token counting using tiktoken or similar
  - **Timeline**: Critical for production deployment

- [ ] **Compaction Service Error Recovery**: Limited retry logic for failed compaction attempts
  - **Impact**: Temporary database issues could cause permanent compaction failures
  - **Solution**: Add exponential backoff retry mechanism
  - **Timeline**: High priority for stability

- [ ] **Concurrent Compaction Safety**: No explicit locking to prevent multiple compactions on same session
  - **Impact**: Race conditions could corrupt memory state
  - **Solution**: Add session-level locking during compaction operations
  - **Timeline**: Critical for multi-user environments

#### **Medium Priority Improvements**
- [ ] **Global Threshold Configuration**: Currently hardcoded to 180k tokens
  - **Enhancement**: Database-stored configuration per workspace/channel
  - **Benefit**: Customizable memory management per use case

- [ ] **Progressive Compaction**: All-or-nothing approach may be inefficient for very large sessions
  - **Enhancement**: Chunk-based compaction for sessions >500k tokens
  - **Benefit**: Better performance and reduced memory usage

- [ ] **Compaction Analytics**: Limited insights into compaction patterns and effectiveness
  - **Enhancement**: Dashboard showing compaction frequency, performance, and savings
  - **Benefit**: Better system optimization and capacity planning

### **🚀 Performance Optimizations**

#### **Database Optimizations**
- [ ] **Batch Operations**: Individual step insertions could be optimized with bulk operations
- [ ] **Index Optimization**: Additional composite indexes for common query patterns
- [ ] **Connection Pooling**: Dedicated connection pool for memory operations
- [ ] **Query Optimization**: Review N+1 query patterns in memory retrieval

#### **Memory Management**
- [ ] **Streaming Compaction**: Process large conversations in chunks to reduce memory footprint
- [ ] **Lazy Loading**: Only load necessary conversation segments during compaction
- [ ] **Garbage Collection**: Automated cleanup of orphaned memory records

### **🎯 Feature Enhancements**

#### **User Experience**
- [ ] **Compaction Progress**: Real-time progress indicators for manual compaction
- [ ] **Recovery Options**: Ability to restore from long-term memory if needed
- [ ] **Compression Ratio**: Show space/token savings from compaction
- [ ] **Historical Analytics**: Per-session compaction history and trends

#### **Administrative Features**
- [ ] **Bulk Operations**: Compact multiple sessions simultaneously
- [ ] **Policy Management**: Configure different compaction policies per channel/user
- [ ] **Scheduled Compaction**: Cron-based compaction during off-peak hours
- [ ] **Emergency Disable**: Circuit breaker to disable auto-compaction if issues arise

#### **Integration Improvements**
- [ ] **Webhooks**: Notify external systems of compaction events
- [ ] **Metrics Export**: Prometheus/Grafana integration for monitoring
- [ ] **Backup Integration**: Automatic backup before compaction operations
- [ ] **Audit Logging**: Complete audit trail for all compaction activities

### **🔒 Security & Compliance**

#### **Data Protection**
- [ ] **Encryption at Rest**: Encrypt sensitive conversation data in long-term storage
- [ ] **PII Detection**: Automatic detection and handling of personal information
- [ ] **Retention Policies**: Configurable data retention periods for archived conversations
- [ ] **Access Controls**: Fine-grained permissions for compaction operations

#### **Compliance Features**
- [ ] **GDPR Compliance**: Right to erasure implementation for archived data
- [ ] **Audit Requirements**: Comprehensive logging for compliance reporting
- [ ] **Data Export**: Standard formats for archived conversation export
- [ ] **Anonymization**: Option to anonymize archived conversations

### **📊 Monitoring & Observability**

#### **Enhanced Metrics**
- [ ] **Custom Dashboards**: Grafana dashboards for compaction monitoring
- [ ] **Alert Rules**: Intelligent alerting for compaction failures and performance issues
- [ ] **Capacity Planning**: Predictive analytics for storage and performance requirements
- [ ] **Health Checks**: Automated verification of compaction system integrity

#### **Debugging Tools**
- [ ] **Compaction Replay**: Ability to replay failed compactions for debugging
- [ ] **Memory Inspector**: Tool to examine memory state before/after compaction
- [ ] **Performance Profiler**: Identify bottlenecks in compaction operations
- [ ] **Log Aggregation**: Centralized logging with correlation IDs

### **🎯 Development & Testing**

#### **Testing Infrastructure**
- [ ] **Automated Test Suite**: Comprehensive integration tests for compaction scenarios
- [ ] **Performance Benchmarks**: Automated performance regression testing
- [ ] **Chaos Engineering**: Fault injection testing for compaction resilience
- [ ] **Load Testing**: High-volume compaction stress testing

#### **Development Tools**
- [ ] **Local Testing**: Docker-based development environment with memory simulation
- [ ] **Migration Tools**: Safe migration path for existing sessions to new compaction system
- [ ] **Configuration Validation**: Startup checks for compaction system configuration
- [ ] **Documentation**: Comprehensive API documentation for compaction methods

**📝 NOTE**: This list represents potential improvements and known areas for enhancement. Priority should be given to critical testing and stability verification before implementing new features. Each improvement should include proper testing, documentation, and rollback procedures.