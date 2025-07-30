# DRAS Refactoring Plan

## Overview
This document outlines a comprehensive refactoring plan for the DRAS (Doppler Radar Notification Service) project based on the Go code review findings. The plan focuses on improving code structure, safety, testability, and maintainability while preserving existing functionality.

## Current Issues Summary

### High Priority Issues
- **Unsafe type assertions** without safety checks (main.go:283)
- **Silent error ignoring** in global variable initialization
- **Fatal errors in goroutines** causing potential program crashes
- **Inefficient mutex usage** with long critical sections

### Medium Priority Issues  
- **Monolithic structure** - everything in single main.go file
- **Limited test coverage** - missing tests for critical functions
- **Hard-coded mappings** - VCP radar modes in switch statements
- **Global variable dependencies** making testing difficult

### Low Priority Issues
- **Missing build-time version information**
- **Potential dependency maintenance burden**
- **No linting/security scanning in CI/CD**

## Refactoring Strategy

### Phase 1: Safety and Error Handling (High Priority)
**Duration: 1-2 days**

1. **Fix unsafe type assertions**
   - Add safety checks to `lastRadarData.(*RadarData)` cast
   - Implement proper error handling for type mismatches

2. **Improve global variable initialization**
   - Create configuration struct with proper error handling
   - Move from global variables to dependency injection pattern

3. **Remove fatal errors from goroutines**
   - Replace `log.Fatalf` with error channels or returns
   - Implement proper error propagation to main goroutine

4. **Optimize mutex usage**
   - Reduce critical section sizes
   - Consider using `sync.Map` or `sync.RWMutex` where appropriate

### Phase 2: Structural Refactoring (Medium Priority)
**Duration: 3-4 days**

1. **Package organization**
   ```
   dras/
   ├── main.go (minimal, just main function)
   ├── internal/
   │   ├── config/     # Configuration management
   │   ├── radar/      # Radar data operations
   │   ├── notify/     # Notification services
   │   └── monitor/    # Monitoring logic
   └── pkg/           # Public interfaces (if any)
   ```

2. **Extract core types and interfaces**
   - Define `Config` struct for all configuration
   - Create `RadarService` interface for radar operations
   - Implement `NotificationService` interface for alerts

3. **Implement dependency injection**
   - Remove global variables
   - Pass dependencies through constructors
   - Enable better testing and mocking

### Phase 3: Testing and Quality (Medium Priority)
**Duration: 2-3 days**

1. **Expand test coverage**
   - Add unit tests for all core functions
   - Implement integration tests for radar API calls
   - Add concurrency tests for goroutine behavior

2. **Add benchmarks**
   - Performance tests for concurrent radar fetching
   - Memory usage benchmarks for long-running monitoring

3. **Mock external dependencies**
   - Mock NWS API calls for testing
   - Mock Pushover notifications for testing

### Phase 4: Enhancement and Polish (Low Priority)
**Duration: 1-2 days**

1. **Add build-time information**
   - Version, build time, commit hash
   - Runtime configuration display

2. **Improve error messages and logging**
   - Structured logging with levels
   - Better error context and user messaging

3. **Configuration validation**
   - Validate station IDs format
   - Check Pushover credentials at startup

## Implementation Details

### New Package Structure

#### `internal/config/config.go`
```go
type Config struct {
    StationIDs         []string
    PushoverAPIToken   string
    PushoverUserKey    string
    CheckInterval      time.Duration
    DryRun            bool
    AlertConfig       AlertConfig
}

type AlertConfig struct {
    VCP         bool
    Status      bool
    Operability bool
    PowerSource bool
    GenState    bool
}

func Load() (*Config, error) {
    // Proper error handling for all env vars
}
```

#### `internal/radar/service.go`
```go
type Service struct {
    client HTTPClient
    cache  *Cache
}

type Cache struct {
    mu   sync.RWMutex
    data map[string]*RadarData
}

func (s *Service) FetchRadarData(ctx context.Context, stationID string) (*RadarData, error)
func (s *Service) CompareRadarData(old, new *RadarData, config AlertConfig) (bool, string)
```

#### `internal/notify/pushover.go`
```go
type Service struct {
    apiToken string
    userKey  string
}

func (s *Service) SendNotification(ctx context.Context, title, message string) error
```

#### `internal/monitor/monitor.go`
```go
type Monitor struct {
    radar   radar.Service
    notify  notify.Service
    config  *config.Config
}

func (m *Monitor) Start(ctx context.Context) error
func (m *Monitor) processStation(ctx context.Context, stationID string) error
```

### Testing Strategy

1. **Unit Tests**
   - Test each package independently
   - Mock external dependencies
   - Cover error paths and edge cases

2. **Integration Tests**
   - Test with real API calls (limited)
   - Verify notification delivery
   - Test configuration loading

3. **Concurrency Tests**
   - Race condition detection
   - Deadlock detection
   - Performance under load

## Migration Plan

### Step 1: Preparation
1. Create feature branch: `refactor/code-structure`
2. Backup current working state
3. Set up new package directories

### Step 2: Incremental Migration
1. **Start with config package**
   - Extract configuration logic
   - Update main.go to use new config
   - Ensure tests pass

2. **Extract radar package**
   - Move radar-related functions
   - Implement proper interfaces
   - Update main.go integration

3. **Extract notification package**
   - Move Pushover logic
   - Add interface for future notification types
   - Update main.go integration

4. **Extract monitor package**
   - Move monitoring loop logic
   - Implement proper error handling
   - Update main.go to minimal launcher

### Step 3: Testing and Validation
1. Run all existing tests
2. Add new tests for extracted packages
3. Perform manual integration testing
4. Verify Docker build still works

### Step 4: Documentation and Cleanup
1. Update README.md with new architecture
2. Add package documentation
3. Update examples if needed

## Success Criteria

### Functional Requirements
- [ ] All existing functionality preserved
- [ ] Docker image builds and runs correctly  
- [ ] All environment variables work as before
- [ ] Notifications continue to work properly

### Quality Requirements
- [ ] Test coverage > 80%
- [ ] No race conditions detected
- [ ] No unsafe type assertions
- [ ] All errors properly handled
- [ ] Code passes `go vet` and `golangci-lint`

### Structural Requirements
- [ ] Clear package separation
- [ ] Minimal main.go file
- [ ] No global variables (except flags/version)
- [ ] Dependency injection implemented
- [ ] Interfaces defined for testability

## Risks and Mitigation

### Risk: Breaking existing functionality
**Mitigation**: Incremental refactoring with tests at each step

### Risk: Performance degradation
**Mitigation**: Benchmarking before and after refactoring

### Risk: Docker build issues
**Mitigation**: Test Docker builds throughout process

### Risk: Configuration changes affecting users
**Mitigation**: Maintain backward compatibility for all env vars

## Timeline

- **Week 1**: Phase 1 (Safety and Error Handling)
- **Week 2**: Phase 2 (Structural Refactoring)  
- **Week 3**: Phase 3 (Testing and Quality)
- **Week 4**: Phase 4 (Enhancement and Polish)

**Total Estimated Duration: 3-4 weeks**

## Next Steps

1. Review and approve this plan
2. Create feature branch
3. Begin Phase 1 implementation
4. Set up automated testing in CI/CD
5. Establish code review process for changes