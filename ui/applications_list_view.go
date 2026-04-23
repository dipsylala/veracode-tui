package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/dipsylala/veracode-tui/services/applications"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// setupApplicationsView creates the applications list view
func (ui *UI) setupApplicationsView() {
	// Create all widgets
	headerWidget := ui.createHeaderWidget()
	filtersWidget := ui.createFiltersWidget()
	applicationsWidget := ui.createApplicationsTableWidget()
	statusWidget := ui.createStatusBarWidget()

	// Create keyboard shortcuts bar
	shortcutsBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("[%s]Enter/Double-click[-] Details  [%s]n/s/t/m/a[-] Filters  [%s]PgDn/PgUp[-] Next/Prev Page  [%s]q/ESC[-] Quit",
			ui.theme.Info, ui.theme.Info, ui.theme.Info, ui.theme.Info))
	shortcutsBar.SetBorder(false)

	// Layout: header, filters (with all fields on one line), status bar, table, shortcuts
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(headerWidget, 3, 0, false).
		AddItem(filtersWidget, 3, 0, false).
		AddItem(statusWidget, 1, 0, false).
		AddItem(applicationsWidget, 0, 1, true).
		AddItem(shortcutsBar, 1, 0, false)

	// Set initial focus to table
	ui.app.SetFocus(ui.applicationsTable)

	// Set up key bindings
	ui.setupApplicationsInputHandlers(flex)

	ui.pages.AddPage("applications", flex, true, true)
}

func (ui *UI) createHeaderWidget() *tview.TextView {
	header := tview.NewTextView().
		SetText("[" + ui.theme.ColumnHeader + "::b]🛡️  Veracode TUI[::-]\n\n").
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)
	header.SetBorder(false)
	return header
}

func (ui *UI) createFiltersWidget() *tview.Flex {
	// Name search input
	ui.searchInput = tview.NewInputField().
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.GetColor(ui.theme.Separator))

	ui.searchInput.SetBlurFunc(func() {
		// Update search query and trigger search when field loses focus
		ui.searchQuery = ui.searchInput.GetText()
		ui.triggerApplicationsSearch()
	})

	// Wrap name input in container with border
	nameContainer := tview.NewFlex().
		AddItem(ui.searchInput, 0, 1, false)
	nameContainer.SetBorder(true).
		SetTitle(" Name (n) ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border)).
		SetBorderPadding(0, 0, 1, 1)

	ui.searchInput.SetFocusFunc(func() {
		nameContainer.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.searchInput.SetBlurFunc(func() {
		nameContainer.SetBorderColor(tcell.GetColor(ui.theme.Border))
		// Update search query and trigger search when field loses focus
		ui.searchQuery = ui.searchInput.GetText()
		ui.triggerApplicationsSearch()
	})

	// Scan Status dropdown - matches ApplicationScan.status enum from Swagger spec
	ui.scanStatusFilter = tview.NewDropDown().
		SetOptions([]string{
			"All",
			"PUBLISHED",
			"INCOMPLETE",
			"IN_PROGRESS",
			"SCAN_IN_PROGRESS",
			"UNPUBLISHED",
			"DELETED",
			"SCAN_SUBMITTED",
			"IN_QUEUE",
			"SCAN_CANCELED",
			"ANALYSIS_ERRORS",
		}, nil).
		SetCurrentOption(0).
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.GetColor(ui.theme.Separator))

	ui.scanStatusFilter.SetSelectedFunc(func(text string, index int) {
		if index == 0 {
			ui.scanStatusFilterValue = ""
		} else {
			ui.scanStatusFilterValue = text
		}
		ui.triggerApplicationsSearch()
	})

	ui.scanStatusFilter.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.app.SetFocus(ui.applicationsTable)
			return nil
		}
		return event
	})

	// Wrap scan status in container with border
	scanStatusContainer := tview.NewFlex().
		AddItem(ui.scanStatusFilter, 0, 1, false)
	scanStatusContainer.SetBorder(true).
		SetTitle(" Scan Status (s) ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border)).
		SetBorderPadding(0, 0, 1, 1)

	ui.scanStatusFilter.SetFocusFunc(func() {
		scanStatusContainer.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.scanStatusFilter.SetBlurFunc(func() {
		scanStatusContainer.SetBorderColor(tcell.GetColor(ui.theme.Border))
		// Trigger search when field loses focus
		ui.triggerApplicationsSearch()
	})

	// Scan Type dropdown - matches scan_type query parameter from Swagger spec
	ui.scanTypeFilter = tview.NewDropDown().
		SetOptions([]string{"All", "STATIC", "DYNAMIC", "MANUAL"}, nil).
		SetCurrentOption(0).
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.GetColor(ui.theme.Separator))

	ui.scanTypeFilter.SetSelectedFunc(func(text string, index int) {
		if index == 0 {
			ui.scanTypeFilterValue = ""
		} else {
			ui.scanTypeFilterValue = text
		}
		ui.triggerApplicationsSearch()
	})

	ui.scanTypeFilter.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.app.SetFocus(ui.applicationsTable)
			return nil
		}
		return event
	})

	// Wrap scan type in container with border
	scanTypeContainer := tview.NewFlex().
		AddItem(ui.scanTypeFilter, 0, 1, false)
	scanTypeContainer.SetBorder(true).
		SetTitle(" Scan Type (t) ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border)).
		SetBorderPadding(0, 0, 1, 1)

	ui.scanTypeFilter.SetFocusFunc(func() {
		scanTypeContainer.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.scanTypeFilter.SetBlurFunc(func() {
		scanTypeContainer.SetBorderColor(tcell.GetColor(ui.theme.Border))
		// Trigger search when field loses focus
		ui.triggerApplicationsSearch()
	})

	// Modified After input field
	ui.modifiedAfterInput = tview.NewInputField().
		SetFieldWidth(0).
		SetPlaceholder("yyyy-MM-dd").
		SetFieldBackgroundColor(tcell.GetColor(ui.theme.Separator))

	ui.modifiedAfterInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			dateText := ui.modifiedAfterInput.GetText()

			// Validate date format if not empty
			if dateText != "" && !ui.isValidDate(dateText) {
				ui.statusBar.SetText("[red]Invalid date format. Please use yyyy-MM-dd (e.g., 2025-12-17)[-]")
				// Keep focus on the input field so user can correct the error
				return
			}

			ui.modifiedAfterFilterValue = dateText
			ui.app.SetFocus(ui.applicationsTable)
			ui.triggerApplicationsSearch()
		} else if key == tcell.KeyEscape {
			ui.app.SetFocus(ui.applicationsTable)
		}
	})

	// Wrap modified after in container with border
	modifiedAfterContainer := tview.NewFlex().
		AddItem(ui.modifiedAfterInput, 0, 1, false)
	modifiedAfterContainer.SetBorder(true).
		SetTitle(" Modified After (m) ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border)).
		SetBorderPadding(0, 0, 1, 1)

	ui.modifiedAfterInput.SetFocusFunc(func() {
		modifiedAfterContainer.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.modifiedAfterInput.SetBlurFunc(func() {
		modifiedAfterContainer.SetBorderColor(tcell.GetColor(ui.theme.Border))
		// Validate and trigger search when field loses focus
		dateText := ui.modifiedAfterInput.GetText()
		if dateText != "" && !ui.isValidDate(dateText) {
			ui.statusBar.SetText("[red]Invalid date format. Please use yyyy-MM-dd (e.g., 2025-12-17)[-]")
			return
		}
		ui.modifiedAfterFilterValue = dateText
		ui.triggerApplicationsSearch()
	})

	// Create horizontal flex with all filters on one line
	container := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nameContainer, 0, 1, false).
		AddItem(scanStatusContainer, 0, 1, false).
		AddItem(scanTypeContainer, 0, 1, false).
		AddItem(modifiedAfterContainer, 0, 1, false)

	return container
}

func (ui *UI) createApplicationsTableWidget() *tview.Table {
	ui.applicationsTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	ui.applicationsTable.SetBorder(true).
		SetTitle(" Applications (a) ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border)).
		SetBorderPadding(0, 0, 1, 1)

	ui.applicationsTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.GetColor(ui.theme.SelectionBackground)).
		Foreground(tcell.GetColor(ui.theme.SelectionForeground)))

	ui.applicationsTable.SetFocusFunc(func() {
		ui.applicationsTable.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.applicationsTable.SetBlurFunc(func() {
		ui.applicationsTable.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	return ui.applicationsTable
}

func (ui *UI) createStatusBarWidget() *tview.TextView {
	ui.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	ui.statusBar.SetBorder(false)
	return ui.statusBar
}

// setupApplicationsInputHandlers configures keyboard input for applications view
//
//nolint:gocyclo // Input handling with multiple hotkeys and navigation options, but sequential, not nested
func (ui *UI) setupApplicationsInputHandlers(flex *tview.Flex) {
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle Tab/Shift-Tab navigation
		if event.Key() == tcell.KeyTab {
			return ui.handleTabNavigation(false)
		} else if event.Key() == tcell.KeyBacktab {
			return ui.handleTabNavigation(true)
		}

		currentFocus := ui.app.GetFocus()

		// Handle global hotkeys (but not when typing in input fields)
		if event.Key() == tcell.KeyRune {
			// Don't trigger hotkeys when user is typing in an input field
			if currentFocus == ui.searchInput || currentFocus == ui.modifiedAfterInput {
				return event
			}

			switch event.Rune() {
			case 'a':
				ui.app.SetFocus(ui.applicationsTable)
				return nil
			case 'n':
				ui.app.SetFocus(ui.searchInput)
				return nil
			case 's':
				ui.app.SetFocus(ui.scanStatusFilter)
				return nil
			case 't':
				ui.app.SetFocus(ui.scanTypeFilter)
				return nil
			case 'm':
				ui.app.SetFocus(ui.modifiedAfterInput)
				return nil
			}
		}

		if currentFocus == ui.applicationsTable {
			return ui.handleApplicationsTableInput(event)
		}
		return event
	})

	ui.applicationsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			row, _ := ui.applicationsTable.GetSelection()
			if row > 0 && row-1 < len(ui.applications) {
				ui.selectedApp = &ui.applications[row-1]
				ui.showApplicationDetail()
			}
			return nil
		}
		return event
	})

	// Add double-click support
	ui.applicationsTable.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			row, _ := ui.applicationsTable.GetSelection()
			if row > 0 && row-1 < len(ui.applications) {
				ui.selectedApp = &ui.applications[row-1]
				ui.showApplicationDetail()
			}
			return action, nil
		}
		return action, event
	})

	ui.searchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			ui.searchQuery = ui.searchInput.GetText()
			ui.app.SetFocus(ui.applicationsTable)
			ui.triggerApplicationsSearch()
		case tcell.KeyEscape:
			ui.app.SetFocus(ui.applicationsTable)
		}
	})
}

// handleApplicationsTableInput handles keyboard input when table has focus
func (ui *UI) handleApplicationsTableInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC, tcell.KeyEscape:
		ui.app.Stop()
		return nil
	case tcell.KeyPgDn:
		if ui.currentPage < ui.totalPages-1 {
			ui.currentPage++
			go func() {
				ui.loadApplications()
			}()
		}
		return nil
	case tcell.KeyPgUp:
		if ui.currentPage > 0 {
			ui.currentPage--
			go func() {
				ui.loadApplications()
			}()
		}
		return nil
	case tcell.KeyRune:
		return ui.handleApplicationsTableRune(event.Rune())
	}
	return event
}

// handleApplicationsTableRune handles rune input for applications table
func (ui *UI) handleApplicationsTableRune(r rune) *tcell.EventKey {
	switch r {
	case 'q':
		ui.app.Stop()
		return nil
	case 'n':
		ui.app.SetFocus(ui.searchInput)
		return nil
	case 's':
		ui.app.SetFocus(ui.scanStatusFilter)
		return nil
	case 't':
		ui.app.SetFocus(ui.scanTypeFilter)
		return nil
	case 'm':
		ui.app.SetFocus(ui.modifiedAfterInput)
		return nil
	case 'a':
		ui.app.SetFocus(ui.applicationsTable)
		return nil
	}
	return nil
}

// triggerApplicationsSearch triggers a new search with current filter values
func (ui *UI) triggerApplicationsSearch() {
	ui.currentPage = 0
	go ui.loadApplications()
}

// handleTabNavigation handles Tab and Shift-Tab navigation between fields
func (ui *UI) handleTabNavigation(reverse bool) *tcell.EventKey {
	focusables := []tview.Primitive{
		ui.searchInput,
		ui.scanStatusFilter,
		ui.scanTypeFilter,
		ui.modifiedAfterInput,
		ui.applicationsTable,
	}

	currentFocus := ui.app.GetFocus()
	currentIndex := -1

	// Find current focus index
	for i, f := range focusables {
		if f == currentFocus {
			currentIndex = i
			break
		}
	}

	// Calculate next index
	var nextIndex int
	if reverse {
		// Shift-Tab: go backwards
		nextIndex = currentIndex - 1
		if nextIndex < 0 {
			nextIndex = len(focusables) - 1
		}
	} else {
		// Tab: go forwards
		nextIndex = currentIndex + 1
		if nextIndex >= len(focusables) {
			nextIndex = 0
		}
	}

	// Set focus to next element
	ui.app.SetFocus(focusables[nextIndex])
	return nil
}

// loadApplications fetches applications from the API
func (ui *UI) loadApplications() {
	ui.app.QueueUpdateDraw(func() {
		ui.statusBar.SetText("[yellow]Loading applications...[-]")
	})

	opts := &applications.GetApplicationsOptions{
		Page: ui.currentPage,
		Size: ui.pageSize,
	}

	// Add search query if present
	if ui.searchQuery != "" {
		opts.Name = ui.searchQuery
	}

	// Add scan status filter if present
	if ui.scanStatusFilterValue != "" {
		opts.ScanStatus = []string{ui.scanStatusFilterValue}
	}

	// Add scan type filter if present
	if ui.scanTypeFilterValue != "" {
		opts.ScanType = ui.scanTypeFilterValue
	}

	// Add modified after filter if present
	if ui.modifiedAfterFilterValue != "" {
		opts.ModifiedAfter = ui.modifiedAfterFilterValue
	}

	result, err := ui.appService.GetApplications(opts)

	if err != nil {
		ui.app.QueueUpdateDraw(func() {
			ui.statusBar.SetText(fmt.Sprintf("[red]Error: %v[-]", err))
		})
		return
	}

	if result.Embedded == nil || result.Embedded.Applications == nil {
		ui.applications = []applications.Application{}
		ui.totalPages = 0
		ui.totalApps = 0
	} else {
		ui.applications = result.Embedded.Applications

		// Sort by Modified date descending (most recent first)
		sort.Slice(ui.applications, func(i, j int) bool {
			if ui.applications[i].Modified == nil {
				return false
			}
			if ui.applications[j].Modified == nil {
				return true
			}
			return ui.applications[i].Modified.After(*ui.applications[j].Modified)
		})

		if result.Page != nil {
			ui.totalPages = int(result.Page.TotalPages)
			ui.totalApps = int(result.Page.TotalElements)
		}
	}

	ui.app.QueueUpdateDraw(func() {
		ui.renderApplicationsTable()
		ui.updateStatusBar()
	})
}

func (ui *UI) renderApplicationsTable() {
	ui.applicationsTable.Clear()

	// Add header row
	headers := []string{"Application Name", "Created", "Last Modified", "Last Scan", "Policy Status", "Scan Status"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.GetColor(ui.theme.ColumnHeader)).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false)
		ui.applicationsTable.SetCell(0, col, cell)
	}

	// Use filtered apps if search is active
	appsToShow := ui.filteredApps
	if appsToShow == nil {
		appsToShow = ui.applications
	}

	// Add application rows
	for row, app := range appsToShow {
		rowNum := row + 1

		// Application name
		appName := "Unknown"
		if app.Profile != nil {
			appName = app.Profile.Name
		}
		if len(appName) > 40 {
			appName = appName[:40] + "..."
		}
		ui.applicationsTable.SetCell(rowNum, 0, tview.NewTableCell(appName))

		// Created date
		created := TextNotAvailable
		if app.Created != nil {
			created = app.Created.Format("2006-01-02")
		}
		ui.applicationsTable.SetCell(rowNum, 1, tview.NewTableCell(created))

		// Last modified
		lastModified := TextNotAvailable
		if app.Modified != nil {
			lastModified = app.Modified.Format("2006-01-02")
		}
		ui.applicationsTable.SetCell(rowNum, 2, tview.NewTableCell(lastModified))

		// Last completed scan date
		lastScan := TextNotAvailable
		if app.LastCompletedScanDate != nil {
			lastScan = app.LastCompletedScanDate.Format("2006-01-02")
		}
		ui.applicationsTable.SetCell(rowNum, 3, tview.NewTableCell(lastScan))

		// Policy status
		policyStatus := TextNotAvailable
		if app.Profile != nil && len(app.Profile.Policies) > 0 {
			policyStatus = app.Profile.Policies[0].PolicyComplianceStatus
		}
		ui.applicationsTable.SetCell(rowNum, 4, tview.NewTableCell(policyStatus))

		// Scan status
		scanStatus := TextNotAvailable
		if len(app.Scans) > 0 {
			scanStatus = app.Scans[0].Status
		}
		ui.applicationsTable.SetCell(rowNum, 5, tview.NewTableCell(scanStatus))
	}

	// Select first data row and scroll to top
	if len(appsToShow) > 0 {
		ui.applicationsTable.Select(1, 0)
		ui.applicationsTable.ScrollToBeginning()
	}
}

func (ui *UI) updateStatusBar() {
	appsToShow := ui.filteredApps
	if appsToShow == nil {
		appsToShow = ui.applications
	}

	statusText := fmt.Sprintf(" Showing %d applications", len(appsToShow))
	if ui.totalPages > 1 {
		statusText += fmt.Sprintf(" • Page %d/%d (Total: %d)", ui.currentPage+1, ui.totalPages, ui.totalApps)
	}
	ui.statusBar.SetText(statusText)
}

// isValidDate validates that the date string matches yyyy-MM-dd format
func (ui *UI) isValidDate(dateStr string) bool {
	// Check basic format with regex
	if len(dateStr) != 10 {
		return false
	}

	// Try to parse the date
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}
