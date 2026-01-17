# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.0] - 2025-07-30

### Added

- Comprehensive package structure with proper separation of concerns
  - `internal/config/` - Configuration management with proper validation
  - `internal/radar/` - Radar data operations and comparison logic  
  - `internal/notify/` - Notification services (Pushover)
  - `internal/monitor/` - Core monitoring and orchestration logic
- Extensive test coverage with 35+ unit and integration tests
- Performance benchmarking suite for concurrent operations
- Mock interfaces for external dependencies (NWS API, Pushover)
- Race condition detection tests and concurrency safety validation
- Integration tests for real API calls (with build tags)
- Proper dependency injection pattern throughout the application
- Enhanced error handling with context propagation
- Integration tests for main package functionality

### Changed

- **BREAKING**: Refactored from monolithic 413-line main.go to clean package structure
- Main function simplified to just 80 lines focusing on initialization
- Improved type safety with proper type assertion checks
- Enhanced mutex usage patterns for better concurrency safety
- All functions now use proper error handling instead of fatal errors in goroutines

### Fixed

- Unsafe type assertions now include proper safety checks
- Eliminated fatal errors in goroutines that could crash the application
- Silent error ignoring in global variable initialization resolved
- Race conditions through optimized mutex usage patterns

### Technical Improvements

- Full backward compatibility maintained - all environment variables work as before
- Docker builds verified and working
- All existing functionality preserved
- Code formatted with `go fmt` and validated with `go vet`
- Follows Go best practices for package organization

## [1.3.1] - Previous Release

### Fixed

- Various bug fixes and improvements

## [1.3.0] - Previous Release

### Added

- Previous features and enhancements

---

**Migration Notes for v2.0.0:**

- No configuration changes required - full backward compatibility maintained
- All environment variables remain the same
- Docker usage unchanged
- This is a pure internal refactoring with extensive testing improvements
- While internally restructured, all external APIs and behavior remain unchanged
