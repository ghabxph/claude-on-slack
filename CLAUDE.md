# Claude on Slack Project Knowledge Base

## 🚧 **DEVELOPMENT STATUS: ALPHA/BETA**

**⚠️ CRITICAL**: This project is in active development and **NOT production-ready**.

### Current Development State
- **Stability**: Known bugs and ongoing stability issues
- **Testing**: Limited test coverage, manual testing only
- **Breaking Changes**: Expected with each update
- **Production Use**: **NOT RECOMMENDED**

### Path to Production Readiness
The project will be considered **stable** when:
1. **TDD Implementation**: Full Test-Driven Development practices
2. **Test Coverage**: Comprehensive automated test suite
3. **CI/CD Pipeline**: Automated testing and deployment
4. **Regression Protection**: New features don't break existing functionality

**Until TDD is fully implemented, expect bugs and instability.**

## 🎯 Implemented Features

### Core Functionality
- [x] **Natural Language Processing**: Direct message processing without command parsing
- [x] **Session Management**: Session tracking with IDs, `/session` command, resumption with `--resume` flag
- [x] **Permission System**: `/permission` command with `default`, `acceptEdits`, `bypassPermissions`, `plan` modes
- [x] **Message Queuing**: Queue messages while processing, combine rapid sequential messages
- [x] **Image Processing**: Auto download/analysis of Slack uploads (JPEG, PNG, GIF, WebP), intelligent cleanup
- [x] **Working Directory Display**: Show current working directory in responses
- [x] **Enhanced Error Reporting** (v2.2.4): Smart categorization, detailed stderr, troubleshooting suggestions
- [x] **Deployment**: SystemD service, redeploy script, tunnel coordination
- [x] **Events API Integration**: Support for Events API and Socket Mode, signature verification

### Memory Management & Auto-Compaction (v2.10.0)
- [x] **Intelligent Auto-Compaction**: Automatic compression at 90% token capacity (162k/180k tokens)
- [x] **Reusable Compaction Engine**: Unified architecture for manual/automatic triggers
- [x] **Advanced Memory Analytics**: Enhanced `/compact status` with token usage percentages
- [x] **Administrative Controls**: `/compact status/now/threshold/help` commands, admin-only safety
- [x] **Dual-Memory Architecture**: Short-term memory + long-term compressed archives, PostgreSQL-backed

## 🏗️ **v3.0 HIGH AVAILABILITY ARCHITECTURE PLAN**

### Vision: RAT (Remote Access Tool) Architecture
**Version 3.0** represents a complete architectural overhaul to achieve **High Availability (HA)** through a distributed **RAT (Remote Access Tool)** design transforming claude-on-slack into a resilient, scalable, distributed system.

### Core Architecture Components

#### 1. Master Service (3-Pod HA)
- **Role**: Central orchestration and state management
- **Deployment**: 3-pod Kubernetes cluster for maximum availability
- **Responsibilities**: Session routing, global state sync, health monitoring, database cluster management
- **Key Feature**: **No Claude Code execution** - purely orchestration layer

#### 2. Slave Nodes (Distributed Workers)
- **Role**: Claude Code execution and user interaction
- **Deployment**: Distributed across multiple machines/regions
- **Responsibilities**: Direct Claude Code CLI execution, local file operations, tool execution, session handling
- **Connection**: Persistent bidirectional communication with Master

#### 3. Distributed Database Architecture
- **Primary**: PostgreSQL cluster with read replicas
- **Caching**: Redis cluster for session state and hot data
- **Search**: Elasticsearch for conversation search and analytics
- **Storage**: Object storage (S3/MinIO) for file attachments and archives

### Communication Protocol Design
- **Protocol**: gRPC with protobuf, secure TLS with mutual authentication
- **Heartbeat**: 30-second intervals with health metrics
- **Load Balancing**: Weighted round-robin based on slave capacity
- **Message Flow**: Slack Event → Master → Session Router → Slave → Claude Code → Response

### Scalability & Performance Goals
- **Concurrent Users**: 10,000+ simultaneous conversations
- **Response Latency**: <500ms for simple commands, <2s for complex operations
- **Availability**: 99.9% uptime with planned maintenance windows
- **Throughput**: 1000+ concurrent tool executions across slave cluster

### RAG (Retrieval-Augmented Generation) Integration
**RAG Integration** transforms claude-on-slack from a stateless conversation tool into an **intelligent knowledge-aware platform** with:
- **Contextual Memory**: Access to conversation history, project knowledge, organizational documentation
- **Code Intelligence**: Understanding of codebase patterns, architectural decisions, development workflows
- **Institutional Knowledge**: Preservation and retrieval of team practices, troubleshooting guides
- **Dynamic Learning**: Continuous knowledge base updates from successful conversation patterns

#### RAG Architecture Components
- **Distributed Vector Database**: Qdrant/Pinecone cluster for semantic search
- **Knowledge Sources**: Conversation archives, code repositories, Slack history, external docs
- **Multi-Modal Processing**: Text, code, images, structured data
- **Real-Time Enhancement**: Query intent detection, multi-vector search, relevance scoring

## 🔄 **CRITICAL: Release Process & Development Guidelines**

### **🚨 MANDATORY: Update Deployment Message for Every Release**
**Before releasing any new version, you MUST update the deployment notification message:**
1. **File to update**: `internal/notifications/deploy.go`
2. **Function**: `FormatDeploymentMessage()` - Update the `else` block
3. **What to include**: Key features, user-visible improvements, technical changes, requirements, CHANGELOG.md link

### Release Checklist:
- [ ] Update version in `internal/config/config.go` (AppVersion field) - **Single Source of Truth**
- [ ] Update CHANGELOG.md with detailed changes
- [ ] **Update deployment message in `internal/bot/service.go` (changes array)** ⚠️
- [ ] Update README.md if needed
- [ ] Test compilation: `go build -o test ./cmd/slack-claude-bot && rm test`
- [ ] Deploy and test functionality
- [ ] Monitor deployment notifications in Slack

## 🏗️ **Architecture Guidelines**

### File Organization:
- `internal/bot/` - Core Slack bot logic and message handling
- `internal/files/` - Image processing, download, and cleanup
- `internal/session/` - Session management (memory + database)
- `internal/claude/` - Claude Code CLI integration
- `internal/notifications/` - Deployment and system notifications

### Key Design Patterns:
- **Repository Pattern**: Database abstraction in `internal/repository/`
- **Service Pattern**: Business logic separation
- **Event-Driven**: Slack events trigger appropriate handlers
- **Background Services**: Cleanup and maintenance tasks

### Dependencies Management:
- Go 1.21+ required
- PostgreSQL for session persistence
- Slack Go SDK for API integration
- Zap for structured logging

## 🧪 **Testing Strategy**

### Before Every Commit:
1. **Compilation Test**: `go build -o test ./cmd/slack-claude-bot && rm test`
2. **Component Tests**: Test new features in isolation
3. **Integration Test**: Deploy and test with real Slack workspace
4. **Permission Test**: Verify Slack OAuth scopes work correctly

### Image Processing Testing:
- Test MIME type validation (accept: JPEG, PNG, GIF, WebP)
- Test file size limits (50MB max)
- Test storage directory creation
- Test cleanup service functionality
- Test with real Slack image uploads

## 📋 **Development Workflow**

### Adding New Features:
1. **Plan**: Update todo list and create implementation plan
2. **Version**: Increment semantic version appropriately
3. **Code**: Follow existing patterns and conventions
4. **Test**: Verify functionality works as expected
5. **Document**: Update CHANGELOG.md and README.md
6. **Deploy Message**: Update `internal/notifications/deploy.go` ⚠️
7. **Deploy**: Use `./scripts/redeploy.sh` for production updates

### Permission Requirements:
When adding features that need new Slack permissions:
- Document required OAuth scopes in README.md
- Update setup instructions
- Test that missing permissions fail gracefully
- Provide clear error messages for permission issues

## 🔒 **Security Considerations**

### File Handling:
- Always validate file types and sizes
- Sanitize filenames to prevent directory traversal
- Use temporary storage with automatic cleanup
- Never store files permanently without explicit user consent

### Slack Integration:
- Verify all webhook signatures
- Use least-privilege OAuth scopes
- Implement rate limiting for API calls
- Log security-relevant events

## 📝 **Documentation Requirements**

### Always Update These Files:
- `CHANGELOG.md` - Detailed change documentation
- `README.md` - User-facing setup and usage instructions
- `internal/bot/service.go` - Deployment message changes array ⚠️
- `internal/config/config.go` - Application version (AppVersion field) - **Single Source of Truth**

### Documentation Standards:
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

## 🧪 **v2.10.0 AUTO-COMPACTION TESTING PLAN**

### Critical Testing Areas

#### 1. Auto-Compaction Triggering (HIGH PRIORITY)
- [ ] **Token Threshold Testing**: Verify auto-compaction triggers exactly at 162k tokens (90% of 180k)
- [ ] **Multiple Session Isolation**: Ensure compaction in one session doesn't affect others
- [ ] **Concurrent Processing**: Test auto-compaction during active tool execution
- [ ] **Threshold Edge Cases**: Test behavior at 161k, 162k, and 163k token boundaries

#### 2. Memory Data Integrity (CRITICAL)
- [ ] **Complete History Preservation**: Verify no conversation data is lost during compaction
- [ ] **Step Sequence Integrity**: Ensure step order is maintained in long-term archives
- [ ] **Token Count Accuracy**: Validate token estimation matches actual usage
- [ ] **Session Linkage**: Verify parent/child session relationships remain intact

#### 3. Performance & Resource Management (HIGH PRIORITY)
- [ ] **Memory Usage**: Monitor RAM consumption during compaction of large conversations
- [ ] **Database Performance**: Test PostgreSQL performance with multiple concurrent compactions
- [ ] **Background Processing**: Verify auto-compaction doesn't block user interactions
- [ ] **Timeout Handling**: Test 10-minute compaction timeout behavior

#### 4. Error Handling & Recovery (CRITICAL)
- [ ] **Database Connection Failures**: Test compaction behavior during DB outages
- [ ] **Compaction Service Failures**: Verify graceful handling of compaction errors
- [ ] **Partial Failures**: Test recovery when compaction partially completes
- [ ] **Concurrent Access**: Test handling of simultaneous compaction attempts on same session

#### 5. User Experience & Interface (MEDIUM PRIORITY)
- [ ] **Command Responsiveness**: Verify `/compact` commands work during auto-compaction
- [ ] **Status Accuracy**: Test `/compact status` during active compaction
- [ ] **Notification Delivery**: Ensure Slack notifications reach correct channels
- [ ] **Error Messages**: Verify user-friendly error messages for common failures

### Stability Red Flags to Watch For

#### Immediate Action Required If:
- **Memory Leaks**: Continuously growing RAM usage during compaction cycles
- **Database Locks**: Long-running queries blocking other operations
- **Failed Compactions**: >5% failure rate for auto-compaction attempts
- **Data Loss**: Any indication of missing conversation history after compaction
- **Performance Degradation**: User-visible slowdowns during background compaction

### Production Deployment Checklist

#### Before Enabling Auto-Compaction in Production:
- [ ] Complete all critical and high-priority testing scenarios
- [ ] Establish baseline performance metrics
- [ ] Set up comprehensive monitoring and alerting
- [ ] Create rollback procedures for emergencies
- [ ] Document troubleshooting procedures for common issues
- [ ] Train support team on new compaction features and commands

**⚠️ CRITICAL**: Do not deploy to production until ALL critical testing areas have been verified and baseline metrics established. The auto-compaction system directly affects conversation data integrity and must be thoroughly validated.

## 🔧 **KNOWN ISSUES & IMPROVEMENTS - v2.10.0+**

### High Priority Fixes Needed
- [ ] **Token Estimation Accuracy**: Current 4-chars-per-token estimation may be inaccurate for code-heavy conversations
- [ ] **Compaction Service Error Recovery**: Limited retry logic for failed compaction attempts
- [ ] **Concurrent Compaction Safety**: No explicit locking to prevent multiple compactions on same session

### Performance Optimizations
- [ ] **Batch Operations**: Individual step insertions could be optimized with bulk operations
- [ ] **Index Optimization**: Additional composite indexes for common query patterns
- [ ] **Streaming Compaction**: Process large conversations in chunks to reduce memory footprint

### Feature Enhancements
- [ ] **Compaction Progress**: Real-time progress indicators for manual compaction
- [ ] **Policy Management**: Configure different compaction policies per channel/user
- [ ] **Emergency Disable**: Circuit breaker to disable auto-compaction if issues arise

**📝 NOTE**: Priority should be given to critical testing and stability verification before implementing new features. Each improvement should include proper testing, documentation, and rollback procedures.