package ui

import (
	"fmt"
	"strings"

	"github.com/dipsylala/veracode-tui/services/applications"
	"github.com/dipsylala/veracode-tui/veracode"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// showApplicationDetail displays the application detail view
func (ui *UI) showApplicationDetail() {
	if ui.selectedApp == nil {
		return
	}

	// Initialize views if first time
	if ui.appInfoView == nil {
		ui.initializeApplicationDetailViews()
	}

	// Reset selection index to policy context
	ui.selectionIndex = -1

	// Clear previous sandboxes immediately
	ui.sandboxes = []applications.Sandbox{}

	// Get application name for title
	appName := "Unknown Application"
	if ui.selectedApp.Profile != nil {
		appName = ui.selectedApp.Profile.Name
	}

	// Create title view
	titleView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	titleView.SetText(fmt.Sprintf("[white::b]Application Details - %s", appName))

	// Always recreate the layout with updated title
	topRow := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.appInfoView, 0, 1, false).
		AddItem(ui.complianceView, 0, 1, false).
		AddItem(ui.recentScansView, 0, 1, false)

	ui.detailFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(titleView, 1, 0, false).
		AddItem(topRow, 0, 0, false).
		AddItem(ui.contextsTable, 0, 1, true)

	// Create keyboard shortcuts bar
	shortcutsBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("[%s]↑/↓[-] Navigate  [%s]Enter/Double-click[-] View Findings  [%s]ESC[-] Back  [%s]q[-] Quit",
			ui.theme.Info, ui.theme.Info, ui.theme.Info, ui.theme.Info))
	shortcutsBar.SetBorder(false)

	ui.detailFlex.AddItem(shortcutsBar, 1, 0, false)

	// Set up input handlers
	ui.setupApplicationDetailInputHandlers()

	// Populate the views with current data
	ui.updateApplicationDetailViews()

	// Fetch full application details to get all scans
	go func() {
		fullApp, err := ui.appService.GetApplication(ui.selectedApp.GUID)
		if err == nil && fullApp != nil {
			// Update the selected app with full details
			ui.selectedApp = fullApp

			// Refresh the views with complete data
			ui.app.QueueUpdateDraw(func() {
				ui.updateApplicationDetailViews()
			})
		}
	}()

	// Load sandboxes for this application
	go func() {
		result, err := ui.appService.GetSandboxes(ui.selectedApp.GUID, &applications.GetSandboxesOptions{
			Size: 100,
		})
		if err == nil && result.Embedded != nil {
			ui.sandboxes = result.Embedded.Sandboxes
		} else {
			ui.sandboxes = []applications.Sandbox{}
		}

		// Refresh the contexts table with sandbox data
		ui.app.QueueUpdateDraw(func() {
			ui.updateContextsTable()
		})
	}()

	// Add or update the page
	if ui.pages.HasPage("detail") {
		ui.pages.RemovePage("detail")
	}
	ui.pages.AddPage("detail", ui.detailFlex, true, false)
	ui.pages.SwitchToPage("detail")
	ui.app.SetFocus(ui.contextsTable)
}

// initializeApplicationDetailViews creates the detail view components
func (ui *UI) initializeApplicationDetailViews() {
	ui.appInfoView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	ui.appInfoView.SetBorder(true).SetTitle(" Application Information ").SetTitleAlign(tview.AlignLeft)

	ui.complianceView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	ui.complianceView.SetBorder(true).SetTitle(" Status & Compliance ").SetTitleAlign(tview.AlignLeft)

	ui.recentScansView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	ui.recentScansView.SetBorder(true).SetTitle(" Recent Scans ").SetTitleAlign(tview.AlignLeft)

	ui.contextsTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	ui.contextsTable.SetBorder(true).SetTitle(" Scan Contexts ").SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1)
	ui.contextsTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.GetColor(ui.theme.SelectionBackground)).
		Foreground(tcell.GetColor(ui.theme.SelectionForeground)))
}

// setupApplicationDetailInputHandlers configures keyboard input for application detail view
func (ui *UI) setupApplicationDetailInputHandlers() {
	ui.detailFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Clear selected application when returning to list
			ui.selectedApp = nil
			ui.pages.SwitchToPage("applications")
			ui.app.SetFocus(ui.applicationsTable)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				ui.app.Stop()
				return nil
			}
		}
		return event
	})

	ui.contextsTable.SetSelectedFunc(func(row, column int) {
		// Handle Enter key on contexts
		// Row 0 is header, row 1 is policy, row 2+ are sandboxes
		if row == 1 {
			// Policy selected
			ui.selectionIndex = -1
			ui.showFindings()
		} else if row > 1 && row-2 < len(ui.sandboxes) {
			// Sandbox selected
			ui.selectionIndex = row - 2
			ui.showFindings()
		}
	})

	// Add double-click support
	ui.contextsTable.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			row, _ := ui.contextsTable.GetSelection()
			if row == 1 {
				// Policy selected
				ui.selectionIndex = -1
				ui.showFindings()
			} else if row > 1 && row-2 < len(ui.sandboxes) {
				// Sandbox selected
				ui.selectionIndex = row - 2
				ui.showFindings()
			}
			return action, nil
		}
		return action, event
	})
}

// updateApplicationDetailViews updates the application info and compliance views
func (ui *UI) updateApplicationDetailViews() {
	if ui.selectedApp == nil {
		return
	}

	// Update titles
	ui.appInfoView.SetTitle(" Application Information ")
	ui.complianceView.SetTitle(" Status & Compliance ")
	ui.recentScansView.SetTitle(" Recent Scans ")

	// Build and set content
	appInfo := ui.buildApplicationInfoContent()
	compliance := ui.buildComplianceContent()
	recentScans := ui.buildRecentScansContent()

	ui.appInfoView.SetText(appInfo)
	ui.complianceView.SetText(compliance)
	ui.recentScansView.SetText(recentScans)

	// Calculate the height needed for the top row based on content
	appInfoLines := strings.Count(appInfo, "\n") + 3 // +3 for border and padding
	complianceLines := strings.Count(compliance, "\n") + 3
	recentScansLines := strings.Count(recentScans, "\n") + 3
	topRowHeight := appInfoLines
	if complianceLines > topRowHeight {
		topRowHeight = complianceLines
	}
	if recentScansLines > topRowHeight {
		topRowHeight = recentScansLines
	}

	// Update the flex layout with the calculated height
	if ui.detailFlex != nil {
		// Get the title from the existing flex (first item)
		var titleView tview.Primitive
		if ui.detailFlex.GetItemCount() >= 1 {
			titleView = ui.detailFlex.GetItem(0)
		}

		// Recreate the top row
		topRow := tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(ui.appInfoView, 0, 1, false).
			AddItem(ui.complianceView, 0, 1, false).
			AddItem(ui.recentScansView, 0, 1, false)

		// Create keyboard shortcuts bar
		shortcutsBar := tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter).
			SetText(fmt.Sprintf("[%s]↑/↓[-] Navigate  [%s]Enter/Double-click[-] View Findings  [%s]ESC[-] Back  [%s]q[-] Quit",
				ui.theme.Info, ui.theme.Info, ui.theme.Info, ui.theme.Info))
		shortcutsBar.SetBorder(false)

		// Clear and rebuild the detail flex
		ui.detailFlex.Clear()
		ui.detailFlex.SetDirection(tview.FlexRow)

		// Re-add title if it exists
		if titleView != nil {
			ui.detailFlex.AddItem(titleView, 1, 0, false)
		}

		ui.detailFlex.AddItem(topRow, topRowHeight, 0, false).
			AddItem(ui.contextsTable, 0, 1, true).
			AddItem(shortcutsBar, 1, 0, false)
	}

	// Update contexts table
	ui.updateContextsTable()
}

// buildApplicationInfoContent builds the application information content string
func (ui *UI) buildApplicationInfoContent() string {
	app := ui.selectedApp

	// Extract data
	businessUnit := TextNotAvailable
	businessCriticality := TextNotAvailable
	description := ""

	if app.Profile != nil {
		if app.Profile.BusinessCriticality != "" {
			businessCriticality = app.Profile.BusinessCriticality
		}
		if app.Profile.BusinessUnit != nil {
			businessUnit = app.Profile.BusinessUnit.Name
		}
		if app.Profile.Description != "" {
			description = app.Profile.Description
		}
	}

	var appInfo strings.Builder
	appInfo.WriteString(fmt.Sprintf("[%s]GUID:[-] %s\n", ui.theme.Label, app.GUID))
	appInfo.WriteString(fmt.Sprintf("[%s]Application ID:[-] %d\n", ui.theme.Label, app.ID))

	// Construct full App Profile URL with hyperlink
	fullAppProfileURL := veracode.BaseWebURL + "auth/index.jsp#" + app.AppProfileURL
	appInfo.WriteString(fmt.Sprintf("[%s]App Profile URL:[-] [:::%s]View Profile[:::-]\n", ui.theme.Label, fullAppProfileURL))

	appInfo.WriteString(fmt.Sprintf("[%s]Business Unit:[-] %s\n", ui.theme.Label, businessUnit))
	appInfo.WriteString(fmt.Sprintf("[%s]Business Criticality:[-] %s\n", ui.theme.Label, businessCriticality))

	if description != "" {
		appInfo.WriteString(fmt.Sprintf("[%s]Description:[-]\n%s\n", ui.theme.Label, description))
	}

	// Additional fields
	if app.Profile != nil {
		if len(app.Profile.Teams) > 0 {
			appInfo.WriteString(fmt.Sprintf("[%s]Teams:[-]\n", ui.theme.Label))
			for _, team := range app.Profile.Teams {
				appInfo.WriteString(fmt.Sprintf("  • %s\n", team.TeamName))
			}
			appInfo.WriteString("\n")
		}

		if len(app.Profile.BusinessOwners) > 0 {
			appInfo.WriteString(fmt.Sprintf("[%s]Business Owners:[-]\n", ui.theme.Label))
			for _, owner := range app.Profile.BusinessOwners {
				appInfo.WriteString(fmt.Sprintf("  • %s <%s>\n", owner.Name, owner.Email))
			}
			appInfo.WriteString("\n")
		}

		if app.Profile.Tags != "" {
			appInfo.WriteString(fmt.Sprintf("[%s]Tags:[-]\n%s\n\n", ui.theme.Label, app.Profile.Tags))
		}

		if app.Profile.Settings != nil {
			appInfo.WriteString(fmt.Sprintf("[%s]Settings:[-]\n", ui.theme.Label))
			appInfo.WriteString(fmt.Sprintf("  SCA Enabled: %v\n", app.Profile.Settings.ScaEnabled))
			appInfo.WriteString(fmt.Sprintf("  Dynamic Scan Approval Required: %v\n", !app.Profile.Settings.DynamicScanApprovalNotRequired))
		}
	}

	return appInfo.String()
}

// buildComplianceContent builds the compliance and status content string
func (ui *UI) buildComplianceContent() string {
	app := ui.selectedApp

	var compliance strings.Builder
	compliance.WriteString(fmt.Sprintf("[%s]Created:[-] %s\n", ui.theme.Label, app.Created.Format("2006-01-02 15:04")))
	if app.Modified != nil {
		compliance.WriteString(fmt.Sprintf("[%s]Modified:[-] %s\n", ui.theme.Label, app.Modified.Format("2006-01-02 15:04")))
	}
	lastScan := "Never"
	if app.LastCompletedScanDate != nil {
		lastScan = app.LastCompletedScanDate.Format("2006-01-02 15:04")
	}
	compliance.WriteString(fmt.Sprintf("[%s]Last Scan:[-] %s\n", ui.theme.Label, lastScan))

	// Policy info
	if app.Profile != nil && len(app.Profile.Policies) > 0 {
		for i, policy := range app.Profile.Policies {
			if i > 0 {
				compliance.WriteString("\n")
			}

			// Policy name
			compliance.WriteString(fmt.Sprintf("[%s]Policy Name:[-] %s", ui.theme.Label, policy.Name))
			if policy.IsDefault {
				compliance.WriteString(fmt.Sprintf(" [%s](Default)[-]", ui.theme.Info))
			}
			compliance.WriteString("\n")

			// Policy compliance status with color coding
			status := policy.PolicyComplianceStatus
			var statusColor string
			switch status {
			case "PASSED", "PASS":
				statusColor = ui.theme.PolicyPass
			case "DID_NOT_PASS", "FAIL":
				statusColor = ui.theme.PolicyFail
			case "CONDITIONAL_PASS":
				statusColor = ui.theme.Warning
			default:
				statusColor = ui.theme.SecondaryText
			}
			compliance.WriteString(fmt.Sprintf("[%s]Policy Compliance:[-] [%s]%s[-]\n", ui.theme.Label, statusColor, status))
		}
	} else {
		compliance.WriteString(fmt.Sprintf("[%s]Policy Name:[-] %s\n", ui.theme.Label, TextNotAvailable))
		compliance.WriteString(fmt.Sprintf("[%s]Policy Compliance:[-] No policy scans found\n", ui.theme.Label))
	}

	return compliance.String()
}

// buildRecentScansContent builds the recent scans content string with hyperlinks
func (ui *UI) buildRecentScansContent() string {
	app := ui.selectedApp

	var scans strings.Builder

	if len(app.Scans) > 0 {
		scans.WriteString(fmt.Sprintf("[%s]Total Scans:[-] %d\n\n", ui.theme.Label, len(app.Scans)))
		for i, scan := range app.Scans {
			published := "Unknown"
			if scan.ModifiedDate != nil {
				published = scan.ModifiedDate.Format("2006-01-02 15:04")
			}

			// Scan type as header
			scans.WriteString(fmt.Sprintf("[%s]%s[-]\n", ui.theme.Label, scan.ScanType))
			scans.WriteString(fmt.Sprintf("  Status: %s\n", scan.Status))
			scans.WriteString(fmt.Sprintf("  Published: %s\n", published))

			// Add hyperlink to scan if URL is available
			if scan.ScanURL != "" {
				fullScanURL := veracode.BaseWebURL + "auth/index.jsp#" + scan.ScanURL
				scans.WriteString(fmt.Sprintf("  [:::%s]View Scan[:::-]\n", fullScanURL))
			}

			if i < len(app.Scans)-1 && i < 4 {
				scans.WriteString("\n")
			}
		}
	} else {
		scans.WriteString("No scans available")
	}

	return scans.String()
}

// updateContextsTable updates the scan contexts table
func (ui *UI) updateContextsTable() {
	if ui.contextsTable == nil || ui.selectedApp == nil {
		return
	}

	ui.contextsTable.Clear()

	// Header row with minimum widths
	headers := []struct {
		name     string
		minWidth int
	}{
		{"Name", 25},
		{"Owner", 20},
		{"Created", 12},
		{"Modified", 12},
		{"Auto-Recreate", 13},
	}

	for col, header := range headers {
		cell := tview.NewTableCell(header.name).
			SetTextColor(tcell.GetColor(ui.theme.ColumnHeader)).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false).
			SetExpansion(1)
		ui.contextsTable.SetCell(0, col, cell)
	}

	// Policy row with minimum widths
	ui.contextsTable.SetCell(1, 0, tview.NewTableCell("Policy").SetExpansion(1))
	ui.contextsTable.SetCell(1, 1, tview.NewTableCell("-").SetExpansion(1))
	ui.contextsTable.SetCell(1, 2, tview.NewTableCell("-").SetExpansion(1))
	ui.contextsTable.SetCell(1, 3, tview.NewTableCell("-").SetExpansion(1))
	ui.contextsTable.SetCell(1, 4, tview.NewTableCell("-").SetExpansion(1))

	// Sandbox rows
	if len(ui.sandboxes) > 0 {
		for i, sandbox := range ui.sandboxes {
			rowNum := i + 2

			name := sandbox.Name
			ui.contextsTable.SetCell(rowNum, 0, tview.NewTableCell(name).SetExpansion(1))
			ui.contextsTable.SetCell(rowNum, 1, tview.NewTableCell(sandbox.OwnerUsername).SetExpansion(1))
			ui.contextsTable.SetCell(rowNum, 2, tview.NewTableCell(sandbox.Created.Format("2006-01-02")).SetExpansion(1))

			modified := "-"
			if sandbox.Modified != nil {
				modified = sandbox.Modified.Format("2006-01-02")
			}
			ui.contextsTable.SetCell(rowNum, 3, tview.NewTableCell(modified).SetExpansion(1))

			autoRecreate := "No"
			if sandbox.AutoRecreate {
				autoRecreate = "[yellow]Yes[-]"
			}
			ui.contextsTable.SetCell(rowNum, 4, tview.NewTableCell(autoRecreate).SetExpansion(1))
		}
	}

	// Select the policy row by default
	ui.contextsTable.Select(1, 0)
}
