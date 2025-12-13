# Veracode TUI - Technical Specification

**Version:** 1.1  
**Last Updated:** December 11, 2025  
**Language:** Go 1.24  
**Framework:** tview

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [UI Components](#ui-components)
4. [Data Models](#data-models)
5. [Service Layer](#service-layer)
6. [Authentication](#authentication)
7. [Current Features](#current-features)
8. [Annotations System](#annotations-system)
9. [In-Memory Cache Architecture](#in-memory-cache-architecture)
10. [Testing](#testing)
11. [UI Styling Guide](#ui-styling-guide)
12. [Theming](#theming)

---

## Overview

A Terminal User Interface (TUI) application for browsing Veracode applications, scans, and security findings. Built with Go and tview, featuring professional layouts, HMAC-SHA256 authentication, and comprehensive service layer architecture.

### Key Technologies

- **Go 1.24** - Core language
- **tview** - TUI framework with powerful terminal UI primitives
- **tcell** - Terminal handling and styling
- **yaml.v3** - Configuration file parsing

---

## Architecture

```
veracode-tui/
├── main.go                      # Entry point with command-line flags
├── config/                      # Configuration file parser and management
├── veracode/                    # HMAC-SHA256 authentication and HTTP client
│   ├── auth.go                  # HMAC signing implementation
│   └── client.go                # HTTP client with HTTPError type
├── services/                    # Service layer for all API operations
│   ├── applications/            # Applications API (models, service, tests)
│   ├── findings/                # Findings API (models, service, tests, enums)
│   ├── annotations/             # Annotations API (models, service, tests)
│   └── identity/                # User identity API
└── ui/                          # Complete TUI with multiple view components
```

### Design Principles

1. **Service Layer Pattern**: Clean separation between API interaction and UI
2. **Professional UI**: Clean layouts with tview primitives
3. **Space Optimization**: Responsive layouts with calculated dimensions
4. **Read-Only Operations**: No destructive API calls (view-only)
5. **Type Safety**: Strongly typed models for all API responses

---

## UI Components

### View Hierarchy

```
viewApplicationList (root)
  ├── viewApplicationDetail
  │   └── viewScansDetail
  │       └── viewFindingsDetail
  │           └── viewFindingDetail
  └── [search mode]
```

### Navigation Keys

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate lists |
| `Enter` or Double-click | Select/View details |
| `/` | Search/Filter (applications list) |
| `m` | Open mitigation modal (finding detail view) |
| `Ctrl+S` | Submit annotation (in modal) |
| `Tab` | Navigate between fields (in modal) |
| `Esc` | Go back or close modal |
| `q` or `Ctrl+C` | Quit |

### Screen Views

#### 1. Applications List
- **Title**: "Veracode Applications X/Y" (current/total count)
- **Column Headers**: 
  ```
  Application Name                              Last Modified      Policy Status    Scan Status
  ```
- **Features**:
  - Last Modified date format: "2006-01-02 15:04"
  - Search/filter by name with `/`
  - Shows paginated list with tview table component
  - Double-click to view application details

#### 2. Application Details
- **Layout**: Two-column boxed layout + full-width scan contexts box
- **Left Column**:
  - Application Name
  - GUID
  - Business Unit
  - Business Criticality
- **Right Column**:
  - Created date
  - Last Modified date
  - Policy Compliance Status
  - Latest Scan Status
- **Bottom Box**:
  - Scan Contexts (Policy + Sandboxes)
  - Click or double-click to view scan details

#### 3. Scans Detail
- **Title**: "Policy Scans" or "Sandbox: {name}"
- **Content**: List of scans with:
  - Scan Type (STATIC/DYNAMIC/MANUAL/SCA)
  - Status
  - Modified Date
  - Click or double-click to view findings

#### 4. Findings List
- **Page Title**: 
  - "━━━ Policy Scans - Scan Findings ━━━" 
  - or "━━━ Sandbox: {name} - Scan Findings ━━━"
  - (Single leading `\n` for visibility)
- **Box Title**: "Scan Information"
- **Context Box**: Shows scan details (type, date, status)
- **Headers**: 
  - **Static**: `ID, Policy, CWE, Sev, Module, File:Line, Status`
  - **Dynamic**: `ID, Policy, CWE, Sev, URL, Parameter, Status`
- **Policy Indicators**:
  - `✓` - Mitigated (APPROVED resolution OR CLOSED without violation)
  - `❌` - Violates policy (no approved mitigation)
  - ` ` (space) - Never violated policy
- **Viewport**: Scrollable with calculated available lines (height - 21 overhead)
- **Pagination**: Page X of Y, ←/→ to navigate
- **Status Display Priority**: 
  1. Resolution Status (if not "NONE")
  2. Mitigation Review Status
  3. Status (fallback)

#### 5. Finding Detail
- **Header**: "━━━ Finding Details ━━━" (no leading newlines)
- **Layout**: Two-column boxes + optional annotation box + description box
- **Left Column** (Basic Information):
  - Issue ID
  - Scan Type
  - CWE
  - Severity
  - Status
  - Context (Policy/Sandbox)
  - Count
  - Policy section (❌/✓/neutral)
- **Right Column** (Status & Technical):
  - Finding Status (First Found, Last Seen, Resolution, etc.)
  - Technical Details:
    - **Static**: File Path, Line Number, Procedure, Module
    - **Dynamic**: URL, Vulnerable Parameter
- **Mitigation Annotations Box** (if annotations exist):
  - Action, User, Date, Description, Comment
  - Multiple annotations separated by horizontal line
  - Word-wrapped comments
- **Description Box** (full width):
  - Word-wrapped finding description
  - Max width calculated from terminal width
- **Press `m`**: Opens mitigation modal

#### 6. Mitigation Modal (Press `m` on Finding Detail)
- **Title**: "Submit Annotation"
- **Action Dropdown**: 
  - Comment - Add explanatory comment
  - FP - False Positive
  - APPDESIGN - Mitigated by Application Design
  - OSENV - Mitigated by OS Environment
  - NETENV - Mitigated by Network Environment
- **Comment TextArea**: Multi-line text input with 1-char padding
- **Status Line**: Shows success/error messages with color coding
- **Controls**:
  - `Tab` - Navigate between action dropdown and comment field
  - `Ctrl+S` - Submit annotation
  - `ESC` or `q` - Close modal
- **On Success**:
  - In-memory data updated instantly
  - Comment indicator (💬) appears if action=COMMENT
  - Finding row refreshed in table
  - Annotations view updated
  - Text area cleared for next submission
- **On Error**:
  - Displays formatted error: `{HTTP Code}:{Error Detail}`
  - Example: `409:Cannot update a flaw currently in a checked out state`

---

## Data Models

### Application Structure

```go
type Application struct {
    GUID                   string
    ID                     int
    LegacyID               int
    AppProfileURL          string
    Created                *time.Time
    Modified               *time.Time
    LastCompletedScanDate  *time.Time
    OID                    int
    OrganizationID         int
    Profile                *ApplicationProfile
    ResultsURL             string
    Scans                  []ApplicationScan
}

type ApplicationProfile struct {
    Name               string
    Description        string
    BusinessCriticality string
    BusinessUnit       *BusinessUnit
    Policies           []AppPolicy
    Teams              []AppTeam
    GitRepoURL         string
}

type ApplicationScan struct {
    ScanType       string  // STATIC, DYNAMIC, MANUAL, SCA
    Status         string
    InternalStatus string
    ModifiedDate   *time.Time
    ScanURL        string
}
```

### Finding Structure

```go
type Finding struct {
    IssueID        int64
    ScanType       string
    Description    string
    Count          int
    ContextType    string  // APPLICATION or SANDBOX
    ContextGUID    string
    ViolatesPolicy bool
    FindingStatus  *FindingStatus
    FindingDetails interface{}  // Map with scan-type-specific fields
    Annotations    []Annotation
}

type FindingStatus struct {
    FirstFoundDate         *time.Time
    LastSeenDate           *time.Time
    Status                 string  // OPEN, CLOSED
    Resolution             string  // MITIGATED, etc.
    ResolutionStatus       string  // APPROVED, PROPOSED, REJECTED, NONE
    New                    bool
    MitigationReviewStatus string
}

type Annotation struct {
    Action      string  // APPROVED, PROPOSED, REJECTED
    Description string  // Mitigation type
    User        string  // Who created annotation
    Date        *time.Time
    Comment     string  // Detailed comment
}

type FindingSummary struct {
    IssueID        int64
    ScanType       string
    CWE            string
    Severity       string
    Status         string
    ViolatesPolicy bool
    IsMitigated    bool  // True if APPROVED or (CLOSED + !ViolatesPolicy)
    Description    string
    
    // Static scan fields
    Module         string
    SourceFile     string
    LineNumber     int
    
    // Dynamic scan fields
    URL            string
    VulnParameter  string
    
    FullFinding    *findings.Finding  // Complete finding data
}
```

### FindingDetails (interface{}) Contents

**Static Scans:**
```go
{
    "cwe": {"id": 89, "name": "SQL Injection"},
    "severity": 4,
    "module": "app.jar",
    "file_path": "com/example/Controller.java",
    "file_line_number": 123,
    "procedure": "com.example.Controller.method",
    "attack_vector": "...",
    "exploitability": 0
}
```

**Dynamic Scans:**
```go
{
    "cwe": {"id": 79, "name": "XSS"},
    "severity": 3,
    "url": "https://example.com/page",
    "vulnerable_parameter": "search"
}
```

---

## Service Layer

### Applications Service

**Package**: `services/applications`

**Methods**:
```go
func (s *Service) GetApplications(options *GetApplicationsOptions) (*PagedResourceOfApplication, error)
func (s *Service) GetSandboxes(appGUID string, options *GetSandboxesOptions) (*PagedResourceOfSandbox, error)
```

**Options**:
```go
type GetApplicationsOptions struct {
    Size int  // Page size (default: 500)
}

type GetSandboxesOptions struct {
    Size int  // Page size (default: 100)
}
```

### Findings Service

**Package**: `services/findings`

**Methods**:
```go
func (s *Service) GetFindings(appGUID string, options *GetFindingsOptions) (*PagedResourceOfFinding, error)
```

**Options**:
```go
type GetFindingsOptions struct {
    Context            string    // "" for policy, sandbox GUID for sandbox
    ScanType           []string  // ["STATIC"], ["DYNAMIC"], ["MANUAL"], ["SCA"]
    Page               int       // Zero-indexed page number
    Size               int       // Results per page (default: 500)
    SeverityGTE        int       // Minimum severity (0-5, 0=no filter)
    ViolatesPolicy     *bool     // nil=all, true=violations only, false=non-violations
    IncludeAnnotations bool      // Include annotation data
}
```

**Enums**:
```go
// Type-safe enums for findings
type ScanType string
const (
    ScanTypeStatic  ScanType = "STATIC"
    ScanTypeDynamic ScanType = "DYNAMIC"
    ScanTypeSCA     ScanType = "SCA"
    ScanTypeManual  ScanType = "MANUAL"
)

type Status string
const (
    StatusOpen     Status = "OPEN"
    StatusClosed   Status = "CLOSED"
    StatusReopened Status = "REOPENED"
)

type ResolutionStatus string
const (
    ResolutionNone     ResolutionStatus = "NONE"
    ResolutionApproved ResolutionStatus = "APPROVED"
    ResolutionRejected ResolutionStatus = "REJECTED"
    ResolutionProposed ResolutionStatus = "PROPOSED"
)

type ContextType string
const (
    ContextTypeApplication ContextType = "APPLICATION"
    ContextTypeSandbox     ContextType = "SANDBOX"
)

type ScanFilterType string
const (
    ScanFilterStatic  ScanFilterType = "STATIC"
    ScanFilterDynamic ScanFilterType = "DYNAMIC"
)

type PolicyFilterType string
const (
    PolicyFilterAll           PolicyFilterType = "All"
    PolicyFilterViolations    PolicyFilterType = "Violations"
    PolicyFilterNonViolations PolicyFilterType = "Non-Violations"
)
```

### Annotations Service

**Package**: `services/annotations`

**Methods**:
```go
func (s *Service) CreateAnnotation(applicationGUID string, annotation *AnnotationData, opts *CreateAnnotationOptions) (*AnnotationResponse, error)
```

**Models**:
```go
type AnnotationData struct {
    IssueList string  // Comma-separated flaw IDs
    Comment   string  // Annotation comment
    Action    string  // Action type (COMMENT, FP, APPDESIGN, etc.)
}

type CreateAnnotationOptions struct {
    Context string  // Sandbox GUID (empty for policy)
}

type AnnotationAction string
const (
    ActionComment       AnnotationAction = "COMMENT"
    ActionFalsePositive AnnotationAction = "FP"
    ActionAppDesign     AnnotationAction = "APPDESIGN"
    ActionOSEnv         AnnotationAction = "OSENV"
    ActionNetEnv        AnnotationAction = "NETENV"
    ActionRejected      AnnotationAction = "REJECTED"
    ActionAccepted      AnnotationAction = "ACCEPTED"
    ActionLibrary       AnnotationAction = "LIBRARY"
    ActionAcceptRisk    AnnotationAction = "ACCEPTRISK"
)

type AnnotationErrorResponse struct {
    Embedded struct {
        APIErrors []APIError
    }
}

type APIError struct {
    ID     string  // Error UUID
    Code   string  // Error code (e.g., "CONFLICT")
    Title  string  // Error title (e.g., "Conflict")
    Detail string  // Detailed error message
    Status string  // HTTP status as string
}
```

**Direct Construction**:
```go
annotation := &annotations.AnnotationData{
    IssueList: "123",
    Comment:   "This is mitigated by design",
    Action:    string(annotations.ActionAppDesign),
}

opts := &annotations.CreateAnnotationOptions{
    Context: sandboxGUID,
}

response, err := service.CreateAnnotation(appGUID, annotation, opts)
```

### Identity Service

**Package**: `services/identity`

**Methods**:
```go
func (s *Service) GetPrincipal(ctx context.Context) (*Principal, error)
```

**Models**:
```go
type Principal struct {
    Username string
    // Additional user fields
}
```

### Error Handling

**HTTPError Type** (`veracode/client.go`):
```go
type HTTPError struct {
    StatusCode int    // HTTP status code (e.g., 400, 404, 500)
    Status     string // HTTP status text (e.g., "Bad Request")
    Body       []byte // Raw response body (JSON)
}

func (e *HTTPError) Error() string {
    return fmt.Sprintf("HTTP %d: %s", e.StatusCode, string(e.Body))
}
```

**Error Unwrapping in UI**:
```go
var httpErr *veracode.HTTPError
if errors.As(err, &httpErr) {
    // Parse JSON body for structured error
    var errorResp annotations.AnnotationErrorResponse
    json.Unmarshal(httpErr.Body, &errorResp)
    // Format: {HTTP Code}:{Detail}
    errorMsg = fmt.Sprintf("%d:%s", httpErr.StatusCode, errorResp.Embedded.APIErrors[0].Detail)
}
```

---

## Authentication

### HMAC-SHA256 Implementation

**Flow**:
1. Generate random 16-byte nonce
2. Get current timestamp (milliseconds)
3. Create data string: `id={keyID}&host={host}&url={url}&method={method}`
4. Four-level HMAC chain:
   ```
   key1 = HMAC-SHA256(nonce, apiSecret)
   key2 = HMAC-SHA256(timestamp, key1)
   key3 = HMAC-SHA256("vcode_request_version_1", key2)
   signature = HMAC-SHA256(data, key3)
   ```
5. Authorization header: `VERACODE-HMAC-SHA-256 id={keyID},ts={timestamp},nonce={hexNonce},sig={hexSignature}`

**Configuration** (`~/.veracode/veracode.yml`):
```yaml
api:
    key-id: your-api-key-id
    key-secret: your-api-key-secret
oauth:
    enabled: false
    region: ""
```

---

## Current Features

### Read-Only Operations

✅ **Application Management**
- List all applications with pagination
- Search/filter by name
- View detailed application info
- Display policy compliance status
- Show scan history

✅ **Scan Browsing**
- View policy scans
- View sandbox scans
- Filter by scan type (STATIC, DYNAMIC)
- Display scan metadata

✅ **Findings Analysis**
- List findings with pagination (500 per page)
- Filter by scan type (Static, Dynamic)
- Filter by severity (Very High to Very Low)
- Filter by policy compliance (All, Violations, Non-Violations)
- View finding details
- Display mitigation status
- Show policy violation indicators
- View mitigation annotations
- Comment indicator (💬) for recent comments
- Priority status display (Resolution > Mitigation Review > Status)

✅ **Policy Indicators**
- ✓ for APPROVED mitigations (even if still violates policy)
- ✓ for CLOSED findings without policy violation
- ❌ for active policy violations
- Space for findings that never violated policy

### Write Operations

✅ **Mitigation Annotations** 🆕
- Submit annotations via modal dialog (press `m` on finding detail)
- Action types supported:
  - COMMENT - Explanatory comments
  - FP - False Positive
  - APPDESIGN - Mitigated by Application Design
  - OSENV - Mitigated by OS Environment
  - NETENV - Mitigated by Network Environment
- Real-time in-memory data updates
- Automatic UI refresh (annotations view, table row, comment indicator)
- Structured error handling with formatted messages
- Username auto-population from identity service
- Multi-line comment support with text area

### Command-Line Flags

```bash
veracode-tui                  # Start interactive TUI
veracode-tui --healthcheck    # Test API connectivity
veracode-tui --version        # Show version
veracode-tui --no-color       # Disable colors (monochrome mode)
veracode-tui --debug-log FILE # Enable API debug logging
veracode-tui --help           # Show help
```

**Environment Variables:**
- `NO_COLOR` - When set, forces monochrome mode (overrides `--no-color`)

---

## Annotations System

### API Integration

**Endpoint**: `POST /appsec/v2/applications/{guid}/annotations`

**Request Body**:
```json
{
  "issue_list": "123,456",
  "comment": "This is mitigated by application design",
  "action": "APPDESIGN"
}
```

**Query Parameters**:
- `context` - Sandbox GUID (optional, omit for policy scans)

**Response** (Success - 200 OK):
```json
{
  "findings": "Updated"
}
```

**Response** (Error - 4xx/5xx):
```json
{
  "_embedded": {
    "api_errors": [{
      "id": "uuid",
      "code": "CONFLICT",
      "title": "Conflict",
      "detail": "Cannot update a flaw currently in a checked out state. Checked out issue(s) [4]",
      "status": "409"
    }]
  }
}
```

### Error Handling Flow

1. **HTTP Error** → `veracode.HTTPError` returned by client
   - Contains: `StatusCode` (int), `Status` (string), `Body` ([]byte)

2. **Service Layer** → Returns error as-is
   - No parsing at service layer
   - Error propagates with full context

3. **UI Layer** → Unwraps and formats error
   - Uses `errors.As()` to detect `HTTPError`
   - Unmarshals JSON body to `AnnotationErrorResponse`
   - Formats as: `{StatusCode}:{Detail}`
   - Example: `409:Cannot update a flaw currently in a checked out state`

### UI Flow

1. User presses `m` on finding detail view
2. Modal opens with:
   - Action dropdown (5 options: COMMENT, FP, APPDESIGN, OSENV, NETENV)
   - Multi-line text area for comment (1-char padding)
   - Status line for messages
3. User enters comment and selects action
4. User presses `Ctrl+S` to submit
5. Submission happens in goroutine:
   - Fetch username from identity service (synchronous)
   - Create annotation via API
   - On success:
     - Create in-memory annotation object
     - Append to `selectedFinding.Annotations`
     - Update propagates to master `findings` list automatically
     - Refresh annotations view in modal
     - Refresh main annotations view (if visible)
     - Update finding row in table (status/comment indicator)
     - Clear text area for next submission
   - On error:
     - Parse HTTPError for formatted message
     - Display in status line with red color

---

## In-Memory Cache Architecture

### Data Structure

**Master List**:
```go
ui.findings []findings.Finding  // Single source of truth
```

**Current Selection**:
```go
ui.selectedFinding *findings.Finding  // Pointer to element in findings
```

### Pointer-Based Updates

When annotation is submitted successfully:

1. **Create Annotation Object**:
   ```go
   newAnnotation := findings.Annotation{
       Action:   action,
       Comment:  comment,
       Created:  &now,
       UserName: userName,
   }
   ```

2. **Update via Pointer**:
   ```go
   if ui.selectedFinding != nil {
       ui.selectedFinding.Annotations = append(ui.selectedFinding.Annotations, newAnnotation)
   }
   ```

3. **Automatic Propagation**:
   - `selectedFinding` points to an element in `ui.findings`
   - Updating `selectedFinding.Annotations` updates the master list
   - No manual synchronization needed

### UI Refresh Flow

1. **Annotations View** - Rebuild content from updated finding
2. **Table Row** - Redraw specific row with new status/emoji
3. **Comment Indicator** - 💬 appears if most recent annotation is COMMENT

### Key Invariants

- ✅ `selectedFinding` always points into `findings` slice
- ✅ Both variables updated simultaneously via pointer
- ✅ No separate filtered list (server-side filtering only)
- ✅ Updates happen in UI thread via `QueueUpdateDraw`

### Race Condition Prevention

**Username Fetch**:
- ❌ OLD: Async goroutine → race condition → always "Current User"
- ✅ NEW: Synchronous fetch before annotation creation
- Safe because already in background goroutine

---

## Testing

### Integration Tests

**Location**: `services/findings/policy_test.go`

**Test**: `TestMCPVerademoStaticFlaws`
- Verifies policy violation behavior
- Tests mitigation status logic
- Validates APPROVED findings display
- Analyzes MCPVerademo application
- Shows full JSON responses for debugging

**Run Tests**:
```bash
go test -v ./services/findings -run TestMCPVerademoStaticFlaws
```

**Skip Integration Tests**:
```bash
go test -short ./...
```

### Test Results (MCPVerademo)

- Total Static Findings: 204
- Violates Policy: 142 (69.6%)
- APPROVED Mitigations: 1 (Finding #6)
  - Status: CLOSED
  - Resolution: MITIGATED
  - Resolution Status: APPROVED
  - Violates Policy: true (API behavior)
  - Should display: ✓ (TUI behavior)

---

## UI Styling Guide

### Box Style

tview uses built-in styling with borders and color support:

```go
box := tview.NewBox().
    SetBorder(true).
    SetBorderColor(tcell.ColorAqua).
    SetBorderPadding(0, 0, 1, 1)
```

### Colors

| Element | Color | tcell Constant |
|---------|-------|----------------|
| Borders/Headers | Cyan | tcell.ColorAqua |
| Labels | Gray | tcell.ColorGray |
| Errors | Red | tcell.ColorRed |
| Footers | Dark Gray | tcell.ColorDarkGray |
| Separators | Dark Gray | tcell.ColorDarkGray |

### Layout Patterns

**Two-Column Layout**:
```go
// Using tview Flex for responsive layouts
flex := tview.NewFlex().
    AddItem(leftBox, 0, 1, false).
    AddItem(rightBox, 0, 1, false)
```

**Full-Width Box**:
```go
box := tview.NewTextView().
    SetDynamicColors(true).
    SetText(content)
```

### Spacing Rules

1. **Single newline** between sections within boxes
2. **No extra spacing** after section headers (e.g., "Basic Information")
3. **Double newline** (`\n\n`) between major components (boxes, footer)
4. **No leading newlines** in component returns (start content immediately)
5. **Single leading newline** for page titles (visibility)

### Header Style

```go
// Using tview color tags and formatting
header := fmt.Sprintf("[aqua::b]━━━ %s ━━━[-:-:-]", title)
```

### Word Wrapping

Custom word wrap implementation for descriptions:
```go
func wordWrap(text string, width int) string
func splitWords(text string) []string
```

Preserves newlines, handles long words, respects terminal width.

---

## Veracode APIs Used

This application integrates with the following Veracode REST APIs:

### Applications API

**Base URL**: `https://api.veracode.com/appsec/v1/`

**Purpose**: Manage and retrieve application information

- `GET /applications` - List applications with pagination
- `GET /applications/{guid}/sandboxes` - List sandboxes for an application

### Findings API

**Base URL**: `https://api.veracode.com/appsec/v2/`

**Purpose**: Access security findings and vulnerabilities

- `GET /applications/{guid}/findings` - Get findings for an application
  - Query params: `context`, `scan_type`, `page`, `size`, `severity_gte`, `violates_policy`
  - Supports filtering by scan type (STATIC, DYNAMIC, SCA, MANUAL)
  - Pagination and severity filtering

### Annotations API

**Base URL**: `https://api.veracode.com/appsec/v2/`

**Purpose**: Submit mitigation annotations and comments on findings

- `POST /applications/{guid}/annotations` - Create annotation for findings
  - Query params: `context` (optional - sandbox GUID)
  - Body: `{"issue_list": "123", "comment": "...", "action": "COMMENT"}`
  - Supports multiple action types (COMMENT, FP, APPDESIGN, OSENV, NETENV, etc.)

### Identity API

**Base URL**: `https://api.veracode.com/api/authn/v2/`

**Purpose**: Retrieve current user information

- `GET /users/self` - Get current user principal
  - Used for annotation attribution and display

### Healthcheck API

**URL**: `https://api.veracode.com/healthcheck/status`

**Purpose**: Validate API connectivity and credentials

- Returns 200 OK if services operational
- Used by `--healthcheck` command-line flag

---

## Build & Run

### Version Management

The application version is stored as a package-level variable in `main.go`:

```go
// Version is the application version, can be set at build time with -ldflags "-X main.Version=x.y.z"
var Version = "dev"
```

**Build with Version Injection:**
```bash
go build -ldflags "-X main.Version=1.0.0" -o veracode-tui.exe
```

**Default Development Build:**
```bash
go build -o veracode-tui.exe  # Uses "dev" as version
```

**Display Version:**
```bash
.\veracode-tui.exe --version
# Output: Veracode TUI vdev
# or with injected version: Veracode TUI v1.0.0
```

**Accessing at Runtime:**
The `main.Version` variable is exported and can be accessed from other packages:
```go
import "main"
fmt.Println(main.Version)
```

### Build
```bash
go build -o veracode-tui.exe
```

### Run
```bash
.\veracode-tui.exe
```

### Development
```bash
go run main.go
```

### Dependencies
```bash
go mod download
go mod tidy
```

---

## Known Behaviors

### API Behavior

1. **violates_policy remains true**: Even for APPROVED mitigations, the API keeps `violates_policy: true`
2. **Resolution precedence**: Resolution Status takes precedence over Mitigation Review Status
3. **Annotations may be empty**: Not all findings have annotations in the list response
4. **CLOSED + ViolatesPolicy**: Possible for approved mitigations (Finding #6 pattern)

### UI Behavior

1. **Mitigated indicator logic**:
   ```go
   if resolutionStatus == "APPROVED" {
       isMitigated = true  // Show ✓
   } else if status == "CLOSED" && !violatesPolicy {
       isMitigated = true  // Show ✓
   }
   ```

2. **Status display priority**:
   - Resolution Status (if not "NONE") → Display this
   - Else Mitigation Review Status → Display this
   - Else Status → Display this

3. **Viewport calculation**:
   ```go
   availableLines := height - 21  // Page title + context + headers + footer
   ```

---

## Future Enhancements

### Planned Features

- [ ] Create new applications
- [ ] Delete applications
- [ ] Upload builds
- [ ] Modify application settings
- [ ] View detailed scan results
- [ ] Generate reports
- [ ] Export findings to CSV/JSON
- [ ] Filter findings by severity
- [ ] Filter findings by CWE
- [ ] View finding trends over time

### Performance Optimizations

- [ ] Cache application list
- [ ] Lazy load finding details
- [ ] Parallel API requests
- [ ] Configurable page sizes
- [ ] Background refresh

---

## Troubleshooting

### Common Issues

**"Error loading configuration"**
- Ensure `~/.veracode/veracode.yml` exists
- Check YAML syntax
- Verify key-id and key-secret are set

**"API request failed"**
- Test with `--healthcheck` flag
- Verify API credentials are active
- Check internet connectivity
- Confirm API region (US/EU/US Federal)

**"No findings found"**
- Application may not have scans
- Try different scan type filter
- Check scan status in application details

**Page title not visible**
- Terminal height too small
- Increase terminal window size
- Page title uses 21 lines overhead

---

## Dependencies

```go
require (
    github.com/rivo/tview
    github.com/gdamore/tcell/v2
    gopkg.in/yaml.v3 v3.0.1
)
```

---

## Security Notes

⚠️ **Never commit credentials**
- `.gitignore` includes `veracode.yml`
- Credentials read from home directory only
- No logging of sensitive data

🔒 **API Key Security**
- Store in `~/.veracode/` directory
- Set appropriate file permissions (600)
- Rotate keys regularly

---

## Resources

- [Veracode API Docs](https://docs.veracode.com/r/c_about_veracode_api)
- [Findings API Spec](https://app.swaggerhub.com/apis/Veracode/veracode-findings_api_specification/2.1)
- [HMAC Authentication](https://docs.veracode.com/r/c_hmac_signing_example)
- [tview Documentation](https://github.com/rivo/tview)
- [tcell Documentation](https://github.com/gdamore/tcell)

---

## Theming

### Overview

The application supports configurable color themes for accessibility and user preference.

### Theme System

**Theme Structure:**
```go
type Theme struct {
    // Text colors
    DefaultText, SecondaryText, DimmedText string
    
    // Label and header colors
    Label, ColumnHeader, Separator string
    
    // Status and severity colors
    Error, Warning, Info, Success, InfoAlt string
    
    // Interactive element colors
    New, Approved, Rejected, Pending string
    
    // UI component colors
    Border, BorderFocused string
    SelectionBackground, SelectionForeground string
    
    // Severity level colors
    SeverityVeryHigh, SeverityHigh, SeverityMedium string
    SeverityLow, SeverityVeryLow, SeverityDefault string
    
    // Policy compliance colors
    PolicyPass, PolicyFail, PolicyNeutral string
}
```

### Built-in Themes

**Default Theme:**
- Full color palette with bright colors for visibility
- Blue borders and headers
- Red/Orange/Yellow/Green severity indicators
- Optimized for dark terminal backgrounds

**Monochrome Theme:**
- Grayscale only (white, gray, dark gray)
- Relies on symbols and text for distinction
- Activated via `NO_COLOR` environment variable or `--no-color` flag
- Follows https://no-color.org/ standard

### Accessibility

**NO_COLOR Standard:**
The application respects the `NO_COLOR` environment variable:
```bash
export NO_COLOR=1  # Linux/Mac
$env:NO_COLOR=1    # Windows PowerShell
```

**Command-Line Flag:**
```bash
veracode-tui --no-color
```

**Behavior:**
- Environment variable takes precedence over flag
- Monochrome mode uses grayscale colors
- All visual information remains accessible through symbols:
  - Policy indicators: ✓ (pass), ❌ (fail)
  - Status emojis: 🆕 (new), ✅ (approved), ❌ (rejected), ⏳ (pending)
  - Severity shown via text (5, 4, 3, 2, 1)

### Future Enhancements

- [ ] Custom theme file support (~/.veracode/theme.yml)
- [ ] Additional built-in themes (light mode, high contrast)
- [ ] Per-element color overrides
- [ ] Theme preview command

---

## Version History

**v1.1** (December 11, 2025)
- ✅ Annotations API integration
- ✅ Submit mitigation annotations (9 action types)
- ✅ Modal dialog for annotation submission
- ✅ In-memory cache updates with pointer architecture
- ✅ HTTPError type with structured error handling
- ✅ Type-safe enums (ScanFilterType, PolicyFilterType, ContextType)
- ✅ Comment indicator emoji (💬) for recent comments
- ✅ Severity and policy filtering for findings
- ✅ Removed redundant `filteredFindings` field
- ✅ Fixed username race condition
- ✅ Identity service for user principal
- ✅ Error response parsing with formatted display
- ✅ Debug logging support (`--debug-log` flag)

**v1.0** (December 8, 2025)
- Initial specification
- Complete TUI implementation
- Professional boxed layouts
- Mitigation annotations viewing
- Policy violation indicators
- Integration tests for policy behavior
- MCPVerademo test case validation

---

## Contact & Support

For issues, questions, or contributions:
- GitHub Issues: [Create an issue](https://github.com/dipsylala/veracode-tui/issues)
- Documentation: See README.md
- API Support: [Veracode Support](https://help.veracode.com/)

---

*End of Specification*
