package ui

import (
	"fmt"
	"sort"

	"github.com/dipsylala/veracode-tui/services/applications"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// setupApplicationsView creates the applications list view
func (ui *UI) setupApplicationsView() {
	// Create all widgets
	headerWidget := ui.createHeaderWidget()
	searchWidget := ui.createSearchWidget()
	applicationsWidget := ui.createApplicationsTableWidget()
	statusWidget := ui.createStatusBarWidget()

	// Create keyboard shortcuts bar
	shortcutsBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("[%s]Enter/Double-click[-] Details  [%s]/[-] Search  [%s]n/p[-] Next/Prev Page  [%s]q/ESC[-] Quit",
			ui.theme.Info, ui.theme.Info, ui.theme.Info, ui.theme.Info))
	shortcutsBar.SetBorder(false)

	// Layout: header, search, status bar, table, shortcuts
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(headerWidget, 3, 0, false).
		AddItem(searchWidget, 3, 0, false).
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

func (ui *UI) createSearchWidget() *tview.Flex {
	ui.searchInput = tview.NewInputField().
		SetFieldWidth(50).
		SetFieldBackgroundColor(tcell.GetColor(ui.theme.Separator))

	container := tview.NewFlex().
		AddItem(ui.searchInput, 0, 1, true)

	container.SetBorder(true).
		SetTitle(" Search ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border)).
		SetBorderPadding(0, 0, 1, 1)

	// Set focus handlers on the input field to change the container's border
	ui.searchInput.SetFocusFunc(func() {
		container.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	ui.searchInput.SetBlurFunc(func() {
		container.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	return container
}

func (ui *UI) createApplicationsTableWidget() *tview.Table {
	ui.applicationsTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	ui.applicationsTable.SetBorder(true).
		SetTitle(" Applications ").
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
func (ui *UI) setupApplicationsInputHandlers(flex *tview.Flex) {
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.app.GetFocus() == ui.applicationsTable {
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
			ui.currentPage = 0
			ui.app.SetFocus(ui.applicationsTable)
			go func() {
				ui.loadApplications()
			}()
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
	case '/':
		ui.app.SetFocus(ui.searchInput)
		return nil
	case 'n':
		if ui.currentPage < ui.totalPages-1 {
			ui.currentPage++
			go func() {
				ui.loadApplications()
			}()
		}
		return nil
	case 'p':
		if ui.currentPage > 0 {
			ui.currentPage--
			go func() {
				ui.loadApplications()
			}()
		}
		return nil
	}
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

	// Select first data row if available
	if len(appsToShow) > 0 {
		ui.applicationsTable.Select(1, 0)
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
