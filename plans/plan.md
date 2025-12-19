Code Review Summary

1.  **`main.go` Overload**: The `main.go` file is currently a "God Object" (over 2,000 lines). It handles CLI flag parsing, database initialization, web server setup, WebSocket management, and specific radio import/export logic. This makes the code difficult to maintain and test.
2.  **Dangerous Reordering Logic**: The `handleChannelReorder` function currently reorders channels by shifting their primary keys (`ID`). This is a dangerous practice in relational databases as it can break referential integrity and is generally inefficient.
3.  **Scattered Radio Logic**: Logic for different radio models (DM32UV, AnyTone 890, DB25-D) is split between `main.go` and the `importer`/`exporter` packages. This should be consolidated.
4.  **Data Models**: The models are comprehensive, but the relationship between Channels, Zones, and Contacts could be more robust, especially regarding sort orders within zones.
5.  **TDD Compliance**: While there are good tests, the project rules emphasize a "test-first" approach for all new logic, which we must adhere to.

---

### Proposed Update Plan

#### Phase 1: Structural Refactoring (COMPLETED)
*   **Modularize `main.go`**:
    *   Created `api` package (`api/server.go`, `api/handlers.go`, `api/websocket.go`) to house HTTP handlers and WebSocket logic.
    *   Created `services` package (`services/codeplug.go`) for core business logic like bandwidth fixing and contact resolution.
    *   Updated `main.go` to use these packages, reducing it to ~470 lines.
*   **Consolidate Radio Logic**:
    *   Radio logic is primarily in `importer` and `exporter` packages.
    *   `main.go` and `api` handlers now route to these packages instead of containing inline logic.
*   **Fix Tests**:
    *   Updated all integration tests to use the new `api` package.
    *   Fixed CGO requirement issues in tests by ensuring `modernc.org/sqlite` is used and configuring `sqlite.Dialector` with "sqlite" driver name.
    *   Fixed shared database state issues in tests by using unique in-memory database names.

#### Phase 2: Model & Logic Improvements
*   **Implement `SortOrder`**:
    *   Add a `SortOrder` field to the `Channel` model.
    *   Refactor `handleChannelReorder` to update this field instead of modifying primary keys.
*   **Enhance Zone Management**:
    *   Ensure `ZoneChannel` associations correctly track `SortOrder` within a zone.
    *   Add dedicated API endpoints for managing zone memberships.
*   **Refine Contact Management**:
    *   Improve the "Resolve Contacts" logic to ensure DMR channels are consistently linked to the correct talkgroups during import.

#### Phase 3: Feature Enhancements
*   **Expand Radio Support**:
    *   Complete AnyTone 890 support (ensure all fields are mapped).
    *   Add initial support for Yaesu System Fusion (YSF/C4FM) as requested in `GEMINI.md`.
*   **API Improvements**: Standardize API responses and error handling to better support the React frontend.

#### Phase 4: Testing (TDD)
*   Write failing tests for the new `SortOrder` logic and zone management before implementation.
*   Ensure high coverage for the refactored API handlers.