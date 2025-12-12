package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/dipsylala/veracode-tui/services/annotations"
	"github.com/dipsylala/veracode-tui/services/findings"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// showFindings displays findings for the selected context (policy or sandbox)
func (ui *UI) showFindings() {
	if ui.selectedApp == nil {
		return
	}

	// Create the findings view components if they don't exist
	if ui.findingsTable == nil {
		ui.initializeFindingsView()
	}

	// Determine context name for title
	contextName := "Policy Scan"
	if ui.selectionIndex >= 0 && ui.selectionIndex < len(ui.sandboxes) {
		contextName = ui.sandboxes[ui.selectionIndex].Name
	}

	// Update title with application name and context
	appName := ui.selectedApp.Profile.Name
	ui.findingsTitleView.SetText(fmt.Sprintf("[white::b]Latest Findings - %s - %s", appName, contextName))
	ui.findingsTable.SetTitle("") // Clear the table title

	// Clear existing data and reset filters
	ui.findings = []findings.Finding{}
	ui.selectedFinding = nil
	ui.findingsScanFilter = findings.ScanFilterStatic
	ui.findingsSeverityFilter = 0
	ui.findingsPolicyFilter = findings.PolicyFilterAll
	ui.scaExpandedComponents = make(map[string]bool)
	ui.findingsFilter.SetCurrentOption(0)                 // Reset to STATIC
	ui.findingsSeverityFilterDropdown.SetCurrentOption(0) // Reset to All
	ui.findingsPolicyFilterDropdown.SetCurrentOption(0)   // Reset to All

	// Set up the filter callbacks (do this after SetCurrentOption to avoid triggering during init)
	ui.setupFindingsFilterCallbacks()

	// Render table with loading message
	ui.findingsTable.Clear()
	loadingCell := tview.NewTableCell("Loading findings...").
		SetTextColor(tcell.GetColor(ui.theme.Pending)).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	ui.findingsTable.SetCell(0, 0, loadingCell)

	// Add or update the page
	if ui.pages.HasPage("findings") {
		ui.pages.RemovePage("findings")
	}
	ui.pages.AddPage("findings", ui.findingsFlex, true, false)
	ui.pages.SwitchToPage("findings")
	ui.app.SetFocus(ui.findingsTable)

	// Load findings with initial filter after UI is ready
	// The count for the loaded scan type will come from the response
	go func() {
		ui.loadFindingsWithFilter("STATIC")
	}()
}

// initializeFindingsView creates all the findings view components
func (ui *UI) initializeFindingsView() {
	ui.findingsTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	ui.findingsTable.SetBorder(true).SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border)).
		SetBorderPadding(0, 0, 1, 1)
	ui.findingsTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.GetColor(ui.theme.SelectionBackground)).
		Foreground(tcell.GetColor(ui.theme.SelectionForeground)))
	ui.findingsTable.SetFocusFunc(func() {
		ui.findingsTable.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.findingsTable.SetBlurFunc(func() {
		ui.findingsTable.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create title view
	ui.findingsTitleView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	ui.findingsTitleView.SetBorder(false)

	// Create filter dropdowns
	ui.findingsFilter = tview.NewDropDown().
		SetLabel("Scan Type: ").
		SetOptions([]string{"STATIC", "DYNAMIC", "SCA"}, nil).
		SetCurrentOption(0).
		SetLabelColor(tcell.GetColor(ui.theme.Label)).
		SetFieldTextColor(tcell.GetColor(ui.theme.DropDownText)).
		SetFieldBackgroundColor(tcell.GetColor(ui.theme.DropDownBackground))
	ui.findingsFilter.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.GetColor(ui.theme.DropDownText)).Background(tcell.GetColor(ui.theme.DropDownBackground)),
		tcell.StyleDefault.Foreground(tcell.GetColor(ui.theme.DropDownSelectedForeground)).Background(tcell.GetColor(ui.theme.DropDownSelectedBackground)))
	ui.findingsFilter.SetBorder(true).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	ui.findingsFilter.SetFocusFunc(func() {
		ui.findingsFilter.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.findingsFilter.SetBlurFunc(func() {
		ui.findingsFilter.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	ui.findingsSeverityFilterDropdown = tview.NewDropDown().
		SetLabel("Min Severity: ").
		SetOptions([]string{"All", "5-Very High", "4-High", "3-Medium", "2-Low", "1-Very Low"}, nil).
		SetCurrentOption(0).
		SetLabelColor(tcell.GetColor(ui.theme.Label)).
		SetFieldTextColor(tcell.GetColor(ui.theme.DropDownText)).
		SetFieldBackgroundColor(tcell.GetColor(ui.theme.DropDownBackground))
	ui.findingsSeverityFilterDropdown.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.GetColor(ui.theme.DropDownText)).Background(tcell.GetColor(ui.theme.DropDownBackground)),
		tcell.StyleDefault.Foreground(tcell.GetColor(ui.theme.DropDownSelectedForeground)).Background(tcell.GetColor(ui.theme.DropDownSelectedBackground)))
	ui.findingsSeverityFilterDropdown.SetBorder(true).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	ui.findingsSeverityFilterDropdown.SetFocusFunc(func() {
		ui.findingsSeverityFilterDropdown.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.findingsSeverityFilterDropdown.SetBlurFunc(func() {
		ui.findingsSeverityFilterDropdown.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	ui.findingsPolicyFilterDropdown = tview.NewDropDown().
		SetLabel("Policy: ").
		SetOptions([]string{"All", "Violations", "Non-Violations"}, nil).
		SetCurrentOption(0).
		SetLabelColor(tcell.GetColor(ui.theme.Label)).
		SetFieldTextColor(tcell.GetColor(ui.theme.DropDownText)).
		SetFieldBackgroundColor(tcell.GetColor(ui.theme.DropDownBackground))
	ui.findingsPolicyFilterDropdown.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.GetColor(ui.theme.DropDownText)).Background(tcell.GetColor(ui.theme.DropDownBackground)),
		tcell.StyleDefault.Foreground(tcell.GetColor(ui.theme.DropDownSelectedForeground)).Background(tcell.GetColor(ui.theme.DropDownSelectedBackground)))
	ui.findingsPolicyFilterDropdown.SetBorder(true).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	ui.findingsPolicyFilterDropdown.SetFocusFunc(func() {
		ui.findingsPolicyFilterDropdown.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.findingsPolicyFilterDropdown.SetBlurFunc(func() {
		ui.findingsPolicyFilterDropdown.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create counts label
	ui.findingsCountsLabel = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	ui.findingsCountsLabel.SetBorder(false)

	// Create flex container for filters and table
	filtersRow := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.findingsFilter, 22, 0, false).
		AddItem(ui.findingsSeverityFilterDropdown, 30, 0, false).
		AddItem(ui.findingsPolicyFilterDropdown, 28, 0, false)

	// Create keyboard shortcuts bar
	shortcutsBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("[%s]Enter/Double-click[-] Details  [%s]Tab[-] Filter  [%s]ESC[-] Back  [%s]q[-] Quit",
			ui.theme.Info, ui.theme.Info, ui.theme.Info, ui.theme.Info))
	shortcutsBar.SetBorder(false)

	ui.findingsFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.findingsTitleView, 1, 0, false).
		AddItem(ui.findingsCountsLabel, 1, 0, false).
		AddItem(filtersRow, 3, 0, false).
		AddItem(ui.findingsTable, 0, 1, true).
		AddItem(shortcutsBar, 1, 0, false)
	ui.findingsFlex.SetBorder(false)

	// Set up input handlers
	ui.setupFindingsInputHandlers()

	// Handle finding selection to show detail view
	ui.findingsTable.SetSelectedFunc(func(row, column int) {
		// For SCA, handle expand/collapse of component groups
		if ui.findingsScanFilter == findings.ScanFilterSCA {
			ui.handleSCARowSelection(row)
			return
		}
		if row > 0 && row-1 < len(ui.findings) {
			ui.selectedFinding = &ui.findings[row-1]
			if ui.selectedFinding.ScanType == findings.ScanTypeSCA {
				ui.showSCAFindingDetail()
			} else {
				ui.showFindingDetail()
			}
		}
	})

	// Add double-click support for non-SCA views
	ui.findingsTable.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			row, _ := ui.findingsTable.GetSelection()
			// Only handle double-click for non-SCA views
			if row > 0 && ui.findingsScanFilter != findings.ScanFilterSCA {
				if row-1 < len(ui.findings) {
					ui.selectedFinding = &ui.findings[row-1]
					ui.showFindingDetail()
				}
			}
		}
		return action, event
	})
}

// setupFindingsInputHandlers configures keyboard input for findings view and filters
//
//nolint:gocyclo // Input handlers for 4 UI components with multiple key bindings each
func (ui *UI) setupFindingsInputHandlers() {
	ui.findingsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.pages.SwitchToPage("detail")
			ui.app.SetFocus(ui.contextsTable)
			return nil
		case tcell.KeyTab:
			ui.app.SetFocus(ui.findingsFilter)
			return nil
		case tcell.KeyBacktab:
			ui.app.SetFocus(ui.findingsPolicyFilterDropdown)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				ui.app.Stop()
				return nil
			} else if event.Rune() == 'f' {
				ui.app.SetFocus(ui.findingsFilter)
				return nil
			}
		}
		return event
	})

	ui.findingsFilter.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.app.SetFocus(ui.findingsTable)
			return nil
		case tcell.KeyTab:
			ui.app.SetFocus(ui.findingsSeverityFilterDropdown)
			return nil
		case tcell.KeyBacktab:
			ui.app.SetFocus(ui.findingsTable)
			return nil
		}
		return event
	})

	ui.findingsSeverityFilterDropdown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.app.SetFocus(ui.findingsTable)
			return nil
		case tcell.KeyTab:
			ui.app.SetFocus(ui.findingsPolicyFilterDropdown)
			return nil
		case tcell.KeyBacktab:
			ui.app.SetFocus(ui.findingsFilter)
			return nil
		}
		return event
	})

	ui.findingsPolicyFilterDropdown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.app.SetFocus(ui.findingsTable)
			return nil
		case tcell.KeyTab:
			ui.app.SetFocus(ui.findingsTable)
			return nil
		case tcell.KeyBacktab:
			ui.app.SetFocus(ui.findingsSeverityFilterDropdown)
			return nil
		}
		return event
	})
}

// setupFindingsFilterCallbacks configures the filter change callbacks
func (ui *UI) setupFindingsFilterCallbacks() {
	ui.findingsFilter.SetSelectedFunc(func(text string, index int) {
		go func() {
			// Convert dropdown text to ScanFilterType
			var scanFilter findings.ScanFilterType
			switch text {
			case "STATIC":
				scanFilter = findings.ScanFilterStatic
			case "DYNAMIC":
				scanFilter = findings.ScanFilterDynamic
			case "SCA":
				scanFilter = findings.ScanFilterSCA
			default:
				scanFilter = findings.ScanFilterStatic
			}
			ui.loadFindingsWithFilter(scanFilter)
		}()
	})

	ui.findingsSeverityFilterDropdown.SetSelectedFunc(func(text string, index int) {
		if index == 0 {
			ui.findingsSeverityFilter = 0 // All
		} else {
			ui.findingsSeverityFilter = 6 - index // Convert index to severity
		}
		go func() {
			ui.loadFindingsWithFilter(ui.findingsScanFilter)
		}()
	})

	ui.findingsPolicyFilterDropdown.SetSelectedFunc(func(text string, index int) {
		// Convert dropdown text to PolicyFilterType
		switch text {
		case "All":
			ui.findingsPolicyFilter = findings.PolicyFilterAll
		case "Violations":
			ui.findingsPolicyFilter = findings.PolicyFilterViolations
		case "Non-Violations":
			ui.findingsPolicyFilter = findings.PolicyFilterNonViolations
		}
		go func() {
			ui.loadFindingsWithFilter(ui.findingsScanFilter)
		}()
	})
}

// loadFindingsWithFilter loads findings with the specified scan type filter
func (ui *UI) loadFindingsWithFilter(scanType findings.ScanFilterType) {
	if ui.selectedApp == nil {
		return
	}

	ui.findingsScanFilter = scanType

	// Determine context value
	contextValue := ""
	if ui.selectionIndex >= 0 && ui.selectionIndex < len(ui.sandboxes) {
		sandbox := ui.sandboxes[ui.selectionIndex]
		contextValue = sandbox.GUID
	}

	// Capture variables for the goroutine
	appGUID := ui.selectedApp.GUID
	capturedContextValue := contextValue
	capturedScanType := string(scanType)
	capturedSeverity := ui.findingsSeverityFilter
	capturedPolicyFilter := ui.findingsPolicyFilter

	// Show loading
	ui.app.QueueUpdateDraw(func() {
		ui.findingsTable.Clear()
		loadingCell := tview.NewTableCell(fmt.Sprintf("Loading %s findings...", capturedScanType)).
			SetTextColor(tcell.GetColor(ui.theme.Pending)).
			SetAlign(tview.AlignCenter).
			SetExpansion(1)
		ui.findingsTable.SetCell(0, 0, loadingCell)
	})

	go func() {
		opts := &findings.GetFindingsOptions{
			Context:            capturedContextValue,
			ScanType:           []string{capturedScanType},
			Size:               500,
			IncludeAnnotations: capturedScanType != "SCA", // Not valid for SCA scan type per API spec
		}

		// Apply severity filter if set
		if capturedSeverity > 0 {
			opts.SeverityGTE = capturedSeverity
		}

		// Apply policy filter if set
		switch capturedPolicyFilter {
		case findings.PolicyFilterViolations:
			violates := true
			opts.ViolatesPolicy = &violates
		case findings.PolicyFilterNonViolations:
			violates := false
			opts.ViolatesPolicy = &violates
		}

		result, err := ui.findingsService.GetFindings(appGUID, opts)

		if err != nil {
			ui.app.QueueUpdateDraw(func() {
				// Show error in the table
				ui.findings = []findings.Finding{}

				// Show error message in table
				errorMsg := fmt.Sprintf("Error loading findings: %s", err.Error())
				errorCell := tview.NewTableCell(errorMsg).
					SetTextColor(tcell.GetColor(ui.theme.Error)).
					SetAlign(tview.AlignCenter).
					SetExpansion(1)
				ui.findingsTable.SetCell(0, 0, errorCell)

				ui.findingsTable.SetTitle(" [ERROR] ")
			})
			return
		}

		if result != nil && result.Embedded != nil {
			ui.findings = result.Embedded.Findings

			// Sort findings by severity (highest first)
			ui.sortFindingsBySeverity()

			// Update the count for this scan type from the response
			if result.Page != nil {
				if capturedScanType == string(findings.ScanTypeStatic) {
					ui.staticCount = result.Page.TotalElements
					// Fetch dynamic count in background
					go ui.loadDynamicCount()
				} else if capturedScanType == string(findings.ScanTypeDynamic) {
					ui.dynamicCount = result.Page.TotalElements
					// Fetch static count in background
					go ui.loadStaticCount()
				} else if capturedScanType == string(findings.ScanTypeSCA) {
					ui.scaCount = result.Page.TotalElements
					// Fetch other counts in background
					go ui.loadStaticCount()
				}
			}
		} else {
			ui.findings = []findings.Finding{}
		}

		// Update the table with findings
		ui.app.QueueUpdateDraw(func() {
			ui.findingsTable.SetTitle(fmt.Sprintf(" %s ", capturedScanType))

			ui.renderFindingsTable()
			ui.updateCountsLabel()
			// Auto-select first finding if available
			if len(ui.findings) > 0 {
				ui.findingsTable.Select(1, 0)
			}
			// Set focus to the findings table after loading
			ui.app.SetFocus(ui.findingsTable)
		})
	}()
}

// renderFindingsTable renders the findings table
func (ui *UI) renderFindingsTable() {
	ui.findingsTable.Clear()

	if len(ui.findings) == 0 {
		ui.renderEmptyFindingsTable()
		return
	}

	// Render headers
	headers := ui.getFindingsTableHeaders(ui.findingsScanFilter)
	ui.renderTableHeaders(headers)

	// Render rows
	if ui.findingsScanFilter == findings.ScanFilterSCA {
		ui.renderSCAGroupedFindings()
	} else {
		for i, finding := range ui.findings {
			ui.renderFindingRow(i+1, &finding)
		}
	}

	// Auto-select first row if nothing selected and scroll to top
	if len(ui.findings) > 0 {
		row, _ := ui.findingsTable.GetSelection()
		if row == 0 {
			ui.findingsTable.Select(1, 0)
		}
		ui.findingsTable.ScrollToBeginning()
	}
}

// renderEmptyFindingsTable shows a message when no findings are available
func (ui *UI) renderEmptyFindingsTable() {
	headers := []string{"ID", "Policy", "CWE", "Sev", "Location", "Status"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.GetColor(ui.theme.ColumnHeader)).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false)
		ui.findingsTable.SetCell(0, col, cell)
	}

	cell := tview.NewTableCell("No findings to display").
		SetTextColor(tcell.GetColor(ui.theme.SecondaryText)).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	ui.findingsTable.SetCell(1, 0, cell)
}

// getFindingsTableHeaders returns the appropriate headers based on scan type
func (ui *UI) getFindingsTableHeaders(scanFilter findings.ScanFilterType) []string {
	switch scanFilter {
	case findings.ScanFilterStatic:
		return []string{"ID", "Policy", "CWE", "Sev", "Module", "File:Line", "Attack Vector", "First Found", "Status"}
	case findings.ScanFilterDynamic:
		return []string{"ID", "Policy", "CWE", "Sev", "URL", "Parameter", "First Found", "Status"}
	case findings.ScanFilterSCA:
		return []string{"Component", "Version", "Policy", "Sev:5", "Sev:4", "Sev:3", "Sev:2", "Sev:1", "CVEs", "First Found", "Status"}
	default:
		return []string{"ID", "Type", "Policy", "CWE", "Sev", "Location", "First Found", "Status"}
	}
}

// renderTableHeaders renders the header row
func (ui *UI) renderTableHeaders(headers []string) {
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.GetColor(ui.theme.ColumnHeader)).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false).
			SetExpansion(1)
		ui.findingsTable.SetCell(0, col, cell)
	}
}

// updateFindingRowInTable updates a specific finding row in the findings table
func (ui *UI) updateFindingRowInTable(finding *findings.Finding) {
	if finding == nil {
		return
	}

	// Find the row index in the current findings list
	rowIndex := -1
	for i := range ui.findings {
		if ui.findings[i].IssueID == finding.IssueID {
			rowIndex = i
			break
		}
	}

	if rowIndex == -1 {
		return // Finding not in current view
	}

	// Re-render the row (rowIndex + 1 because row 0 is headers)
	ui.renderFindingRow(rowIndex+1, finding)
}

// renderFindingRow renders a single finding row based on scan type
func (ui *UI) renderFindingRow(rowNum int, finding *findings.Finding) {
	switch ui.findingsScanFilter {
	case findings.ScanFilterStatic:
		ui.renderStaticRow(rowNum, finding)
	case findings.ScanFilterDynamic:
		ui.renderDynamicRow(rowNum, finding)
	case findings.ScanFilterSCA:
		ui.renderSCARow(rowNum, finding)
	}
}

// renderStaticRow renders a row for static scan findings
//
//nolint:dupl // Intentionally separate from renderDynamicRow for scan-type-specific clarity
func (ui *UI) renderStaticRow(rowNum int, finding *findings.Finding) {
	col := 0

	// Issue ID
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(fmt.Sprintf("%d", finding.IssueID)).SetExpansion(1))
	col++

	// Policy indicator
	col = ui.renderPolicyIndicator(rowNum, col, finding)

	// CWE
	cwe := extractCWE(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(cwe).SetExpansion(1))
	col++

	// Severity
	severity := extractSeverity(finding)
	sevColor := ui.getSeverityColor(severity)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(severity).SetTextColor(sevColor).SetExpansion(1))
	col++

	// Module
	module := extractModule(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(module).SetExpansion(1))
	col++

	// File:Line
	fileLine := extractFileLine(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(fileLine).SetExpansion(1))
	col++

	// Attack Vector
	attackVector := extractAttackVector(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(attackVector).SetExpansion(1))
	col++

	// First Found Date
	firstFound := extractFirstFoundDate(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(firstFound).SetExpansion(1))
	col++

	// Status
	status := extractStatus(finding)
	statusColor := ui.getStatusColor(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(status).SetTextColor(statusColor).SetExpansion(1))
}

// renderDynamicRow renders a row for dynamic scan findings
//
//nolint:dupl // Intentionally separate from renderStaticRow for scan-type-specific clarity
func (ui *UI) renderDynamicRow(rowNum int, finding *findings.Finding) {
	col := 0

	// Issue ID
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(fmt.Sprintf("%d", finding.IssueID)).SetExpansion(1))
	col++

	// Policy indicator
	col = ui.renderPolicyIndicator(rowNum, col, finding)

	// CWE
	cwe := extractCWE(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(cwe).SetExpansion(1))
	col++

	// Severity
	severity := extractSeverity(finding)
	sevColor := ui.getSeverityColor(severity)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(severity).SetTextColor(sevColor).SetExpansion(1))
	col++

	// URL
	url := extractURL(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(url).SetExpansion(1))
	col++

	// Parameter
	param := extractParameter(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(param).SetExpansion(1))
	col++

	// First Found Date
	firstFound := extractFirstFoundDate(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(firstFound).SetExpansion(1))
	col++

	// Status
	status := extractStatus(finding)
	statusColor := ui.getStatusColor(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(status).SetTextColor(statusColor).SetExpansion(1))
}

// renderSCARow renders a row for SCA scan findings
func (ui *UI) renderSCARow(rowNum int, finding *findings.Finding) {
	col := 0

	// Policy indicator (no Issue ID for SCA)
	col = ui.renderPolicyIndicator(rowNum, col, finding)

	// Component
	component := extractComponent(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(component).SetExpansion(1))
	col++

	// Version
	version := extractVersion(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(version).SetExpansion(1))
	col++

	// Policy
	status := extractStatus(finding)
	statusColor := ui.getStatusColor(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(status).SetTextColor(statusColor).SetExpansion(1))
	col++

	// CVE
	cve := extractCVE(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(cve).SetExpansion(1))
	col++

	// Severity
	severity := extractSeverity(finding)
	sevColor := ui.getSeverityColor(severity)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(severity).SetTextColor(sevColor).SetExpansion(1))
}

// SCAComponent represents a grouped component with its CVEs
type SCAComponent struct {
	Name         string
	Version      string
	CVEs         []*findings.Finding
	SevCounts    map[int]int
	WorstSev     int
	HasViolation bool
}

// renderSCAGroupedFindings renders SCA findings grouped by component
func (ui *UI) renderSCAGroupedFindings() {
	// Group findings by component+version
	components := ui.groupSCAByComponent()

	rowNum := 1
	for _, comp := range components {
		// Render component summary row
		ui.renderSCAComponentRow(rowNum, comp)
		rowNum++

		// If expanded, render CVE detail rows
		componentKey := comp.Name + "|" + comp.Version
		if ui.scaExpandedComponents[componentKey] {
			for _, cve := range comp.CVEs {
				ui.renderSCACVERow(rowNum, cve)
				rowNum++
			}
		}
	}
}

// groupSCAByComponent groups findings by component name and version
func (ui *UI) groupSCAByComponent() []*SCAComponent {
	componentMap := make(map[string]*SCAComponent)

	for i := range ui.findings {
		finding := &ui.findings[i]
		component := extractComponent(finding)
		version := extractVersion(finding)
		key := component + "|" + version

		comp, exists := componentMap[key]
		if !exists {
			comp = &SCAComponent{
				Name:      component,
				Version:   version,
				CVEs:      []*findings.Finding{},
				SevCounts: make(map[int]int),
				WorstSev:  0,
			}
			componentMap[key] = comp
		}

		comp.CVEs = append(comp.CVEs, finding)

		// Track severity counts
		sev := ui.getFindingSeverity(finding)
		comp.SevCounts[sev]++
		if sev > comp.WorstSev {
			comp.WorstSev = sev
		}

		// Track if any CVE violates policy
		if finding.ViolatesPolicy {
			comp.HasViolation = true
		}
	}

	// Convert map to sorted slice (by severity counts, then name)
	components := make([]*SCAComponent, 0, len(componentMap))
	for _, comp := range componentMap {
		components = append(components, comp)
	}

	// Sort by severity counts (5, 4, 3, 2, 1 in descending order), then by name
	for i := 0; i < len(components); i++ {
		for j := i + 1; j < len(components); j++ {
			if ui.shouldSwapComponents(components[i], components[j]) {
				components[i], components[j] = components[j], components[i]
			}
		}
	}

	return components
}

// shouldSwapComponents determines if two components should be swapped during sorting
func (ui *UI) shouldSwapComponents(a, b *SCAComponent) bool {
	// Compare severity counts from 5 down to 1
	for sev := 5; sev >= 1; sev-- {
		if a.SevCounts[sev] < b.SevCounts[sev] {
			return true
		} else if a.SevCounts[sev] > b.SevCounts[sev] {
			return false
		}
	}

	// If all severity counts are equal, sort by name
	return a.Name > b.Name
}

// getEarliestCVEDate finds the earliest first found date across all CVEs
func (ui *UI) getEarliestCVEDate(cves []*findings.Finding) string {
	var earliestDate *time.Time
	for _, cve := range cves {
		if cve.FindingStatus != nil && cve.FindingStatus.FirstFoundDate != nil {
			if earliestDate == nil || cve.FindingStatus.FirstFoundDate.Before(*earliestDate) {
				earliestDate = cve.FindingStatus.FirstFoundDate
			}
		}
	}
	if earliestDate != nil {
		return earliestDate.Format("2006-01-02 15:04")
	}
	return TextNotAvailable
}

// getWorstCVEStatus finds the worst status across all CVEs
func (ui *UI) getWorstCVEStatus(cves []*findings.Finding) *findings.Finding {
	var worstStatus *findings.Finding
	for _, cve := range cves {
		if worstStatus == nil {
			worstStatus = cve
		} else if cve.FindingStatus != nil && worstStatus.FindingStatus != nil {
			// NEW takes priority, then OPEN
			if cve.FindingStatus.New || (cve.FindingStatus.Status == findings.StatusOpen && !worstStatus.FindingStatus.New) {
				worstStatus = cve
			}
		}
	}
	return worstStatus
}

// renderSeverityCounts renders the severity count columns for SCA component row
func (ui *UI) renderSeverityCounts(rowNum int, col int, comp *SCAComponent) int {
	for sev := 5; sev >= 1; sev-- {
		count := comp.SevCounts[sev]
		countText := "-"
		if count > 0 {
			countText = fmt.Sprintf("%d", count)
		}
		sevColor := ui.getSeverityColor(fmt.Sprintf("%d", sev))
		ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(countText).
			SetTextColor(sevColor).
			SetExpansion(1))
		col++
	}
	return col
}

// renderSCAComponentRow renders a component summary row
func (ui *UI) renderSCAComponentRow(rowNum int, comp *SCAComponent) {
	col := 0
	componentKey := comp.Name + "|" + comp.Version
	expanded := ui.scaExpandedComponents[componentKey]

	// Expand/collapse indicator + Component name
	expandChar := "▶"
	if expanded {
		expandChar = "▼"
	}
	componentText := fmt.Sprintf("%s %s", expandChar, comp.Name)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(componentText).
		SetTextColor(tcell.GetColor(ui.theme.DefaultText)).
		SetAttributes(tcell.AttrBold).
		SetExpansion(1))
	col++

	// Version
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(comp.Version).SetExpansion(1))
	col++

	// Policy indicator (whether component affects policy)
	statusText := " "
	statusColor := tcell.GetColor(ui.theme.PolicyNeutral)
	if comp.HasViolation {
		statusText = EmojiViolatesPolicy
		statusColor = tcell.GetColor(ui.theme.PolicyFail)
	}
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(statusText).
		SetTextColor(statusColor).
		SetAlign(tview.AlignCenter).
		SetExpansion(1))
	col++

	// Severity counts (5, 4, 3, 2, 1)
	col = ui.renderSeverityCounts(rowNum, col, comp)

	// Total CVEs count
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(fmt.Sprintf("%d", len(comp.CVEs))).
		SetExpansion(1))
	col++

	// First Found Date (earliest from all CVEs)
	firstFound := ui.getEarliestCVEDate(comp.CVEs)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(firstFound).SetExpansion(1))
	col++

	// Status (aggregate status across CVEs - show worst status)
	worstStatus := ui.getWorstCVEStatus(comp.CVEs)
	status := "-"
	statusTextColor := tcell.GetColor(ui.theme.DefaultText)
	if worstStatus != nil {
		status = extractStatus(worstStatus)
		statusTextColor = ui.getStatusColor(worstStatus)
	}
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(status).SetTextColor(statusTextColor).SetExpansion(1))
}

// renderSCACVERow renders a CVE detail row (indented under component)
func (ui *UI) renderSCACVERow(rowNum int, finding *findings.Finding) {
	col := 0

	// Indented CVE name with hyperlink
	cve := extractCVE(finding)
	cveHref := extractCVEHref(finding)

	var cveText string
	if cveHref != "" {
		// Create clickable hyperlink using tview's format: [:::URL]text[:::-]
		cveText = fmt.Sprintf("  └─ [:::%s]%s[:::-]", cveHref, cve)
	} else {
		cveText = fmt.Sprintf("  └─ %s", cve)
	}

	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(cveText).
		SetTextColor(tcell.GetColor(ui.theme.SecondaryText)).
		SetExpansion(1))
	col++

	// Skip version column (empty for CVE rows)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell("").SetExpansion(1))
	col++

	// Policy indicator for this CVE
	policyIndicator := getPolicyIndicator(finding)
	policyColor := tcell.GetColor(ui.theme.PolicyNeutral)
	switch policyIndicator {
	case "✓":
		policyColor = tcell.GetColor(ui.theme.PolicyPass)
	case "❌":
		policyColor = tcell.GetColor(ui.theme.PolicyFail)
	}
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(policyIndicator).
		SetTextColor(policyColor).
		SetAlign(tview.AlignCenter).
		SetExpansion(1))
	col++

	// Severity columns - show "*" in the appropriate column
	severity := extractSeverity(finding)
	sevValue := 0
	if severity != "-" {
		_, _ = fmt.Sscanf(severity, "%d", &sevValue)
	}
	sevColor := ui.getSeverityColor(severity)

	for sev := 5; sev >= 1; sev-- {
		if sev == sevValue {
			ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell("*").
				SetTextColor(sevColor).
				SetExpansion(1))
		} else {
			ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell("").SetExpansion(1))
		}
		col++
	}

	// CVEs column - empty for detail rows
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell("").
		SetExpansion(1))
	col++

	// First Found Date
	firstFound := extractFirstFoundDate(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(firstFound).
		SetTextColor(tcell.GetColor(ui.theme.SecondaryText)).
		SetExpansion(1))
	col++

	// Status
	status := extractStatus(finding)
	statusColor := ui.getStatusColor(finding)
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(status).
		SetTextColor(statusColor).
		SetExpansion(1))
}

// handleSCARowSelection handles row selection for SCA grouped view
func (ui *UI) handleSCARowSelection(row int) {
	if row == 0 {
		return // Header row
	}

	// Group findings and find which component this row corresponds to
	components := ui.groupSCAByComponent()

	currentRow := 1
	for _, comp := range components {
		if currentRow == row {
			// Toggle expansion for this component
			componentKey := comp.Name + "|" + comp.Version
			ui.scaExpandedComponents[componentKey] = !ui.scaExpandedComponents[componentKey]
			ui.renderFindingsTable()
			// Keep the same row selected
			ui.findingsTable.Select(row, 0)
			return
		}
		currentRow++

		// Skip expanded CVE rows
		if ui.scaExpandedComponents[comp.Name+"|"+comp.Version] {
			if row >= currentRow && row < currentRow+len(comp.CVEs) {
				// Clicked on a CVE row - show SCA detail
				cveIndex := row - currentRow
				if cveIndex < len(comp.CVEs) {
					ui.selectedFinding = comp.CVEs[cveIndex]
					ui.showSCAFindingDetail()
				}
				return
			}
			currentRow += len(comp.CVEs)
		}
	}
}

// renderPolicyIndicator renders the policy indicator column
func (ui *UI) renderPolicyIndicator(rowNum, col int, finding *findings.Finding) int {
	policyIndicator := getPolicyIndicator(finding)
	policyColor := tcell.GetColor(ui.theme.PolicyNeutral)
	switch policyIndicator {
	case "✓":
		policyColor = tcell.GetColor(ui.theme.PolicyPass)
	case "❌":
		policyColor = tcell.GetColor(ui.theme.PolicyFail)
	}
	ui.findingsTable.SetCell(rowNum, col, tview.NewTableCell(policyIndicator).SetTextColor(policyColor).SetExpansion(1))
	return col + 1
}

// Helper functions for extracting finding data

func getPolicyIndicator(finding *findings.Finding) string {
	if finding.FindingStatus != nil {
		// APPROVED mitigations
		if finding.FindingStatus.ResolutionStatus == findings.ResolutionApproved {
			return EmojiPassesPolicy
		}
		// CLOSED without policy violation
		if finding.FindingStatus.Status == findings.StatusClosed && !finding.ViolatesPolicy {
			return EmojiPassesPolicy
		}
	}

	// Violates policy
	if finding.ViolatesPolicy {
		return EmojiViolatesPolicy
	}
	// Space for findings that never violated policy
	return " "
}

func extractCWE(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	if cweData, ok := details["cwe"].(map[string]interface{}); ok {
		if cweID, ok := cweData["id"].(float64); ok {
			return fmt.Sprintf("%d", int(cweID))
		}
	}

	return "-"
}

func extractSeverity(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	if severity, ok := details["severity"].(float64); ok {
		return fmt.Sprintf("%d", int(severity))
	}

	return "-"
}

func (ui *UI) getSeverityColor(severity string) tcell.Color {
	switch severity {
	case "5":
		return tcell.GetColor(ui.theme.SeverityVeryHigh)
	case "4":
		return tcell.GetColor(ui.theme.SeverityHigh)
	case "3":
		return tcell.GetColor(ui.theme.SeverityMedium)
	case "2":
		return tcell.GetColor(ui.theme.SeverityLow)
	case "1", "0":
		return tcell.GetColor(ui.theme.SeverityVeryLow)
	default:
		return tcell.GetColor(ui.theme.SeverityDefault)
	}
}

func extractModule(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	if module, ok := details["module"].(string); ok {
		return module
	}
	return "-"
}

func extractAttackVector(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	if attackVector, ok := details["attack_vector"].(string); ok && attackVector != "" {
		// Truncate if too long (e.g., show first 50 chars)
		if len(attackVector) > 50 {
			return attackVector[:47] + "..."
		}
		return attackVector
	}
	return "-"
}

// extractFilePathWithLine extracts file path and line number from finding details
func extractFilePathWithLine(details map[string]interface{}) string {
	filePath := ""
	fileName := ""
	lineNum := ""

	if fp, ok := details["file_path"].(string); ok && fp != "" {
		filePath = fp
	}
	if fn, ok := details["file_name"].(string); ok && fn != "" {
		fileName = fn
	}
	if ln, ok := details["file_line_number"].(float64); ok {
		lineNum = fmt.Sprintf(":%d", int(ln))
	}

	// If we have a file path or file name, use that
	if filePath != "" {
		// Show just filename, not full path
		parts := strings.Split(filePath, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1] + lineNum
		}
		return filePath + lineNum
	} else if fileName != "" {
		return fileName + lineNum
	}

	return ""
}

// extractProcedureLocation extracts procedure and relative location from finding details
func extractProcedureLocation(details map[string]interface{}) string {
	procedure := ""
	if proc, ok := details["procedure"].(string); ok && proc != "" {
		procedure = proc
		// Simplify long procedure names - just take the last part
		if strings.Contains(procedure, ".") {
			parts := strings.Split(procedure, ".")
			procedure = parts[len(parts)-1]
		}
		// Truncate if still too long
		if len(procedure) > 30 {
			procedure = procedure[:27] + "..."
		}
	}

	percentage := ""
	if relLoc, ok := details["relative_location"].(float64); ok {
		percentage = fmt.Sprintf("%.0f%%", relLoc)
	}

	// Combine procedure and percentage
	if procedure != "" && percentage != "" {
		return fmt.Sprintf("%s:%s", procedure, percentage)
	} else if procedure != "" {
		return procedure
	} else if percentage != "" {
		return percentage
	}

	return ""
}

func extractFileLine(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	// Try file path first
	if fileInfo := extractFilePathWithLine(details); fileInfo != "" {
		return fileInfo
	}

	// Fall back to procedure/location
	if procInfo := extractProcedureLocation(details); procInfo != "" {
		return procInfo
	}

	return "-"
}

func extractURL(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	if url, ok := details["url"].(string); ok {
		// Truncate long URLs
		if len(url) > 50 {
			return url[:47] + "..."
		}
		return url
	}
	return "-"
}

func extractParameter(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	if param, ok := details["vulnerable_parameter"].(string); ok {
		return param
	}
	return "-"
}

// extractComponent extracts the component filename from SCA finding details
func extractComponent(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	if component, ok := details["component_filename"].(string); ok {
		return component
	}
	return "-"
}

// extractVersion extracts the version from SCA finding details
func extractVersion(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	if version, ok := details["version"].(string); ok {
		return version
	}
	return "-"
}

// extractCVE extracts the CVE identifier from SCA finding details
func extractCVE(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return "-"
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return "-"
	}

	// CVE information is nested in the cve object
	if cveData, ok := details["cve"].(map[string]interface{}); ok {
		if cveName, ok := cveData["name"].(string); ok {
			return cveName
		}
	}
	return "-"
}

// extractCVEHref extracts the CVE href URL from SCA finding details
func extractCVEHref(finding *findings.Finding) string {
	if finding.FindingDetails == nil {
		return ""
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return ""
	}

	// CVE information is nested in the cve object
	if cveData, ok := details["cve"].(map[string]interface{}); ok {
		if href, ok := cveData["href"].(string); ok {
			return href
		}
	}
	return ""
}

func extractFirstFoundDate(finding *findings.Finding) string {
	if finding.FindingStatus == nil || finding.FindingStatus.FirstFoundDate == nil {
		return TextNotAvailable
	}
	return finding.FindingStatus.FirstFoundDate.Format("2006-01-02 15:04")
}

func extractStatus(finding *findings.Finding) string {
	if finding.FindingStatus == nil {
		return EmojiUnknown + "    " // Four spaces for two missing emojis
	}

	// First character: flaw status (new/open/closed)
	var flawStatusChar string
	if finding.FindingStatus.New {
		flawStatusChar = EmojiNew
	} else {
		switch finding.FindingStatus.Status {
		case findings.StatusOpen:
			flawStatusChar = EmojiOpen
		case findings.StatusReopened:
			flawStatusChar = EmojiReopened
		case findings.StatusClosed:
			flawStatusChar = EmojiPassesPolicy
		default:
			flawStatusChar = EmojiUnknown
		}
	}

	// Second character: mitigation status (proposed/approved/rejected)
	mitigationChar := getMitigationStatusChar(finding.FindingStatus)

	// Third character: comment indicator if most recent annotation is a comment
	commentChar := "  " // Two spaces to match emoji width
	if len(finding.Annotations) > 0 {
		// Find the most recent annotation by Created date
		var mostRecent *findings.Annotation
		for i := range finding.Annotations {
			ann := &finding.Annotations[i]
			if ann.Created != nil {
				if mostRecent == nil || (mostRecent.Created != nil && ann.Created.After(*mostRecent.Created)) {
					mostRecent = ann
				}
			}
		}
		// Check if most recent annotation is a comment
		if mostRecent != nil && mostRecent.Action == string(annotations.ActionComment) {
			commentChar = EmojiComment
		}
	}

	return flawStatusChar + mitigationChar + commentChar
}

func getMitigationStatusChar(status *findings.FindingStatus) string {
	// Check Resolution Status first (higher priority than Mitigation Review Status)
	if status.ResolutionStatus != "" && status.ResolutionStatus != findings.ResolutionNone {
		return resolutionStatusToEmoji(status.ResolutionStatus)
	}
	if status.MitigationReviewStatus != "" && status.MitigationReviewStatus != findings.ResolutionNone {
		return resolutionStatusToEmoji(status.MitigationReviewStatus)
	}
	return "  " // Two spaces to match emoji width
}

func resolutionStatusToEmoji(status findings.ResolutionStatus) string {
	switch status {
	case findings.ResolutionApproved:
		return EmojiApproved
	case findings.ResolutionRejected:
		return EmojiRejected
	case findings.ResolutionProposed:
		return EmojiPending
	default:
		return "  " // Two spaces to match emoji width
	}
}

// getResolutionStatusColor returns the tcell color for a resolution status
func (ui *UI) getResolutionStatusColor(status findings.ResolutionStatus) tcell.Color {
	switch status {
	case findings.ResolutionApproved:
		return tcell.GetColor(ui.theme.Approved)
	case findings.ResolutionRejected:
		return tcell.GetColor(ui.theme.Rejected)
	case findings.ResolutionProposed:
		return tcell.GetColor(ui.theme.Pending)
	default:
		return tcell.GetColor(ui.theme.PolicyNeutral)
	}
}

func (ui *UI) getStatusColor(finding *findings.Finding) tcell.Color {
	if finding.FindingStatus == nil {
		return tcell.GetColor(ui.theme.PolicyNeutral)
	}

	hasNewFlag := finding.FindingStatus.New

	// Check for mitigation status (Resolution Status > Mitigation Review Status)
	if finding.FindingStatus.ResolutionStatus != "" && finding.FindingStatus.ResolutionStatus != findings.ResolutionNone {
		if hasNewFlag {
			return tcell.GetColor(ui.theme.New)
		}
		return ui.getResolutionStatusColor(finding.FindingStatus.ResolutionStatus)
	}

	if finding.FindingStatus.MitigationReviewStatus != "" && finding.FindingStatus.MitigationReviewStatus != findings.ResolutionNone {
		if hasNewFlag {
			return tcell.GetColor(ui.theme.New)
		}
		return ui.getResolutionStatusColor(finding.FindingStatus.MitigationReviewStatus)
	}

	if finding.FindingStatus.Status != "" {
		switch finding.FindingStatus.Status {
		case findings.StatusOpen:
			if hasNewFlag {
				return tcell.GetColor(ui.theme.New)
			}
			return tcell.GetColor(ui.theme.Rejected)
		case findings.StatusClosed:
			return tcell.GetColor(ui.theme.Approved)
		case findings.StatusReopened:
			return tcell.GetColor(ui.theme.Warning)
		default:
			return tcell.GetColor(ui.theme.PolicyNeutral)
		}
	}

	// If no other status but it's new
	if hasNewFlag {
		return tcell.GetColor(ui.theme.New)
	}

	return tcell.GetColor(ui.theme.PolicyNeutral)
}

// loadFindingsCount fetches the count for a specific scan type
func (ui *UI) loadFindingsCount(scanType findings.ScanType, updateCount func(int64)) {
	if ui.selectedApp == nil {
		return
	}

	// Determine context value
	contextValue := ""
	if ui.selectionIndex >= 0 && ui.selectionIndex < len(ui.sandboxes) {
		contextValue = ui.sandboxes[ui.selectionIndex].GUID
	}

	result, err := ui.findingsService.GetFindings(ui.selectedApp.GUID, &findings.GetFindingsOptions{
		Context:  contextValue,
		ScanType: []string{string(scanType)},
		Size:     1, // We only need the count
	})
	if err == nil && result.Page != nil {
		updateCount(result.Page.TotalElements)
		ui.app.QueueUpdateDraw(func() {
			ui.updateCountsLabel()
		})
	}
}

// loadStaticCount fetches only the static findings count
func (ui *UI) loadStaticCount() {
	ui.loadFindingsCount(findings.ScanTypeStatic, func(count int64) {
		ui.staticCount = count
	})
}

// loadDynamicCount fetches only the dynamic findings count
func (ui *UI) loadDynamicCount() {
	ui.loadFindingsCount(findings.ScanTypeDynamic, func(count int64) {
		ui.dynamicCount = count
	})
	ui.loadFindingsCount(findings.ScanTypeSCA, func(count int64) {
		ui.scaCount = count
	})
}

func (ui *UI) updateCountsLabel() {
	ui.findingsCountsLabel.SetText(fmt.Sprintf("  [white]Static: [%s]%d[white]  |  Dynamic: [%s]%d[white]  |  SCA: [%s]%d", ui.theme.Label, ui.staticCount, ui.theme.Label, ui.dynamicCount, ui.theme.Label, ui.scaCount))
}

func (ui *UI) sortFindingsBySeverity() {
	// Sort by severity in descending order (5 = highest, 0 = lowest)
	for i := 0; i < len(ui.findings); i++ {
		for j := i + 1; j < len(ui.findings); j++ {
			sevI := ui.getFindingSeverity(&ui.findings[i])
			sevJ := ui.getFindingSeverity(&ui.findings[j])
			if sevI < sevJ {
				ui.findings[i], ui.findings[j] = ui.findings[j], ui.findings[i]
			}
		}
	}
}

func (ui *UI) getFindingSeverity(finding *findings.Finding) int {
	if finding.FindingDetails == nil {
		return 0
	}

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return 0
	}

	if severity, ok := details["severity"].(float64); ok {
		return int(severity)
	}

	return 0
}
