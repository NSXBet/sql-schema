# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-01-03

### Added

#### Schema Extraction
- MySQL schema extractor with support for MySQL 5.7+, 8.0+, and MariaDB 10.3+
- PostgreSQL schema extractor with support for PostgreSQL 10+
- Complete schema metadata extraction including:
  - Tables, columns, indexes, constraints
  - Views, materialized views (PostgreSQL)
  - Functions, procedures, triggers
  - Sequences (PostgreSQL), events (MySQL)
  - Extensions (PostgreSQL), enum types (PostgreSQL)
  - Partitions, foreign keys, check constraints
  - Table statistics (row count, data size, index size)

#### Schema Comparison
- Engine-aware schema comparison with registry pattern
- Support for 11+ database object types
- Type alias recognition (INTEGER=INT=INT4, VARCHAR=CHARACTER VARYING, etc.)
- Semantic expression comparison with normalization
- Sub-object change tracking (columns within tables, etc.)
- Detailed diff output with CREATE/DROP/ALTER actions
- Comparison options (ignore comments, charset, collation)
- Auto-import feature - engines registered automatically

#### Migration Strategy
- Risk assessment with 4 levels (NONE, LOW, MEDIUM, HIGH)
- Automatic risk detection based on operation type
- Dependency ordering using topological sort
- Circular dependency detection
- Data loss warnings and reversibility tracking
- Special case handling (NOT NULL columns, foreign keys, etc.)
- 12 operation types supported

#### Output Formats
- **Text Format**: Console-friendly hierarchical output
- **Markdown Format**: GitHub-flavored markdown with tables and badges
- **JSON Format**: Full export and summary options
- Strategy reports in all formats
- Risk indicators with emoji support

#### Testing
- 36+ comprehensive tests with 100% pass rate
- Integration tests with Docker containers
- Type alias recognition tests
- Complex schema comparison tests
- End-to-end workflow tests

#### Documentation
- Comprehensive README with examples
- Complete API reference
- Multiple implementation summaries
- Working examples for all features
- Quick start guide

### Technical Details

**Architecture:**
- Clean separation of concerns (extraction, comparison, strategy, formatting)
- Engine abstraction layer with registry pattern
- Thread-safe operations
- Deterministic, sorted output
- Proper error handling with error wrapping

**Test Coverage:**
- postgres: 70.0%
- mysql: 60.2%
- util/format: 51.0%
- util: 39.2%
- util/migration: 37.5%

**Performance:**
- O(n) schema comparison
- O(n + e) dependency ordering
- O(1) type alias detection
- Typical schema (50 tables): <10ms
- Large schema (500 tables): <100ms

### Known Limitations

- AST-based comparison is stubbed (uses normalization fallback)
- Engine packages lack direct unit tests (tested via integration)
- Currently supports PostgreSQL and MySQL (extensible to others)

### Breaking Changes

None - Initial release

### Migration Guide

This is the initial release. To use:

```go
import (
    "github.com/nsxbet/sql-schema/util"
    "github.com/nsxbet/sql-schema/util/engine"
)

// Engines auto-registered - just use them
opts := &util.CompareOptions{Engine: engine.PostgreSQL}
diff, err := util.CompareSchemasDetailed(oldSchema, newSchema, opts)
```

[0.1.0]: https://github.com/nsxbet/sql-schema/releases/tag/v0.1.0
