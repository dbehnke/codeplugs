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

#### Phase 2: Model & Logic Improvements (COMPLETED)
*   **Implement `SortOrder`**:
    *   Added `SortOrder` field to the `Channel` model.
    *   Refactored `HandleChannelReorder` to update this field instead of modifying primary keys.
*   **Enhance Zone Management**:
    *   Updated `ZoneChannel` model to include `Channel` association.
    *   Added `ZoneChannels` one-to-many relationship to `Zone` model.
    *   Refactored `HandleZoneAssignment` to explicitly save sort order in the join table.
    *   Refactored `HandleZones` to preload `ZoneChannels` sorted by `SortOrder` and map them to the response.
*   **Refine Contact Management**:
    *   Improved `ResolveContacts` to use strict case-insensitive trimming for name matching.
    *   Implemented auto-creation of missing contacts with temporary negative DMR IDs to avoid unique constraint violations.

#### Phase 3: Feature Enhancements & API Standardization (COMPLETED)
*   **AnyTone 890 & DM32UV Completion**:
    *   **Scan Lists**: Implemented Import/Export for `ScanList.CSV` (AnyTone) and `scan_lists.csv` (DM32UV) and linked channels.
    *   **Roaming Support**: Implemented `RoamingChannel` and `RoamingZone` models and CSV parsing/exporting for both platforms.
*   **API Standardization**:
    *   Refactored HTTP API to use a consistent JSON response wrapper (`{ success: true, data: ..., error: ... }`).
    *   Centralized error handling with `api.RespondError` and success with `api.RespondJSON`.

#### Phase 4: Ongoing Testing (TDD) (Active)
*   Write failing tests for new features (Scan Lists, Roaming, API wrappers) before implementation.
*   Maintain high coverage for all new logic.

#### Phase 5: API & Integration (COMPLETED)
*   **Roaming & Scan List API**:
    *   Implemented `HandleRoamingChannels` (GET/POST/DELETE) with JSON tag fixes.
    *   Implemented `HandleRoamingZones` (GET/POST/DELETE).
    *   Implemented `HandleRoamingAssignment` (POST) to assign channels to roaming zones.
    *   Implemented `HandleScanLists` (GET/POST/DELETE).
*   **Import Logic Upgrade**:
    *   Updated `HandleImport` to support ZIP uploads containing `ScanList.CSV`, `RoamChannel.CSV`, and `RoamZone.CSV` (AnyTone structure) and their DM32UV equivalents (`scan_lists.csv`, `roaming_channels.csv`, `roaming_zones.csv`).
    *   Verified with `TestHandleImport_Zip_RoamingScanList`.
*   **Web UI Readiness**:
    *   Backend endpoints are now fully implemented and tested for future UI integration.

> [!NOTE]
> **Out of Scope**: Yaesu System Fusion (YSF) support is explicitly excluded from this project plan at this time.