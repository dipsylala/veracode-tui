package ui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dipsylala/veracode-tui/services/annotations"
	"github.com/dipsylala/veracode-tui/services/findings"
	"github.com/dipsylala/veracode-tui/veracode"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// findingDetailViews holds all the views for the finding detail page
type findingDetailViews struct {
	titleView      *tview.TextView
	leftView       *tview.TextView
	rightView      *tview.TextView
	techView       *tview.TextView
	dataPathsView  *tview.TextView
	annotView      *tview.TextView
	descView       *tview.TextView
	focusableViews []tview.Primitive
}

// showFindingDetail displays the detailed view for a selected finding
func (ui *UI) showFindingDetail() {
	if ui.selectedFinding == nil {
		return
	}

	finding := ui.selectedFinding

	// Determine context name
	contextName := "Policy Scan"
	if finding.ContextType == findings.ContextTypeSandbox && ui.selectionIndex >= 0 && ui.selectionIndex < len(ui.sandboxes) {
		contextName = ui.sandboxes[ui.selectionIndex].Name
	}

	// Get application name
	appName := "Unknown Application"
	if ui.selectedApp != nil && ui.selectedApp.Profile != nil {
		appName = ui.selectedApp.Profile.Name
	}

	// Create all views
	views := ui.createFindingDetailViews(appName, contextName)

	// Show loading indicators initially
	loadingText := fmt.Sprintf("[%s]Loading details...[-]", ui.theme.Pending)
	views.leftView.SetText(loadingText)
	views.rightView.SetText(loadingText)
	views.techView.SetText(loadingText)
	views.annotView.SetText(loadingText)
	views.descView.SetText(loadingText)

	// Set up technical details row (includes data paths for STATIC scans)
	techRow := ui.setupTechnicalDetailsRow(finding, views.techView, views.dataPathsView)

	// Determine focusable views based on scan type - include all views in logical order
	if finding.ScanType == findings.ScanTypeStatic {
		views.focusableViews = []tview.Primitive{
			views.leftView,
			views.rightView,
			views.techView,
			views.dataPathsView,
			views.annotView,
			views.descView,
		}
	} else {
		views.focusableViews = []tview.Primitive{
			views.leftView,
			views.rightView,
			views.techView,
			views.annotView,
			views.descView,
		}
	}

	// Create main layout
	topRow := tview.NewFlex().
		AddItem(views.leftView, 0, 1, false).
		AddItem(views.rightView, 0, 1, false)

	// Create keyboard shortcuts bar
	shortcutsBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	if finding.ScanType == findings.ScanTypeStatic {
		shortcutsBar.SetText(fmt.Sprintf("[%s]ESC[-] Back  [%s]q[-] Quit  [%s]m[-] Mitigations  [%s]Tab[-] Navigate  [%s]←/→[-] Data Paths",
			ui.theme.Info, ui.theme.Info, ui.theme.Info, ui.theme.Info, ui.theme.Info))
	} else {
		shortcutsBar.SetText(fmt.Sprintf("[%s]ESC[-] Back  [%s]q[-] Quit  [%s]m[-] Mitigations  [%s]Tab[-] Navigate",
			ui.theme.Info, ui.theme.Info, ui.theme.Info, ui.theme.Info))
	}
	shortcutsBar.SetBorder(false)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(views.titleView, 1, 0, false).
		AddItem(topRow, 11, 0, false).
		AddItem(techRow, 12, 0, false).
		AddItem(views.annotView, 0, 1, true).
		AddItem(views.descView, 0, 1, false).
		AddItem(shortcutsBar, 1, 0, false)

	// Set up input handling
	mainLayout.SetInputCapture(ui.createFindingDetailInputHandler(finding, views.focusableViews))

	ui.findingDetailView = mainLayout

	// Add or update the page
	if ui.pages.HasPage("finding_detail") {
		ui.pages.RemovePage("finding_detail")
	}
	ui.pages.AddPage("finding_detail", ui.findingDetailView, true, false)
	ui.pages.SwitchToPage("finding_detail")
	ui.app.SetFocus(views.descView)

	// Load content asynchronously
	go ui.loadFindingDetailContent(finding, views)
}

// createFindingDetailViews creates all the views for the finding detail page
func (ui *UI) createFindingDetailViews(appName, contextName string) *findingDetailViews {
	views := &findingDetailViews{}

	// Create title view
	views.titleView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	views.titleView.SetText(fmt.Sprintf("[white::b]Finding Details - %s - %s", appName, contextName))
	views.titleView.SetBorder(false)

	// Create left column (Basic Information & Policy)
	views.leftView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWordWrap(true)
	views.leftView.SetBorder(true).
		SetTitle(" Basic Information & Policy ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	views.leftView.SetFocusFunc(func() {
		views.leftView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	views.leftView.SetBlurFunc(func() {
		views.leftView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create right column (Finding Status)
	views.rightView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWordWrap(true)
	views.rightView.SetBorder(true).
		SetTitle(" Finding Status ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	views.rightView.SetFocusFunc(func() {
		views.rightView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	views.rightView.SetBlurFunc(func() {
		views.rightView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create technical details view
	views.techView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWordWrap(true)
	views.techView.SetBorder(true).
		SetTitle(" Technical Details ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	views.techView.SetFocusFunc(func() {
		views.techView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	views.techView.SetBlurFunc(func() {
		views.techView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create data paths view (scrollable, only for STATIC scans)
	views.dataPathsView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	views.dataPathsView.SetBorder(true).
		SetTitle(" Data Paths ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	views.dataPathsView.SetFocusFunc(func() {
		views.dataPathsView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	views.dataPathsView.SetBlurFunc(func() {
		views.dataPathsView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create annotations view (scrollable)
	views.annotView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	views.annotView.SetBorder(true).
		SetTitle(" Mitigations ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	views.annotView.SetFocusFunc(func() {
		views.annotView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	views.annotView.SetBlurFunc(func() {
		views.annotView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create description view (scrollable)
	views.descView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	views.descView.SetBorder(true).
		SetTitle(" Description ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	views.descView.SetFocusFunc(func() {
		views.descView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	views.descView.SetBlurFunc(func() {
		views.descView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	return views
}

// loadFindingDetailContent loads finding detail content asynchronously
func (ui *UI) loadFindingDetailContent(finding *findings.Finding, views *findingDetailViews) {
	// Build all content (this is fast, just string formatting)
	leftContent := ui.buildBasicInfoContent(finding)
	rightContent := ui.buildFindingStatusContent(finding)
	techContent := ui.buildTechnicalDetailsContent(finding)
	annotContent := ui.buildAnnotationsContent(finding)
	descContent := ui.buildDescriptionContent(finding)

	// Update all views
	ui.app.QueueUpdateDraw(func() {
		views.leftView.SetText(leftContent)
		views.rightView.SetText(rightContent)
		views.techView.SetText(techContent)
		views.annotView.SetText(annotContent)
		views.descView.SetText(descContent)

		// Store reference to annotations view for updates
		ui.findingAnnotationsView = views.annotView
	})

	// For STATIC scans, load data paths (this can be slow due to API call)
	if finding.ScanType == findings.ScanTypeStatic {
		ui.loadAndDisplayStaticFlawInfo(finding, views.dataPathsView)
	}
}

// setupTechnicalDetailsRow sets up the technical details row, including data paths for STATIC scans
func (ui *UI) setupTechnicalDetailsRow(finding *findings.Finding, techView, dataPathsView *tview.TextView) tview.Primitive {
	if finding.ScanType == findings.ScanTypeStatic {
		dataPathsView.SetText(fmt.Sprintf("[%s]Loading data paths...[-]", ui.theme.Pending))
		dataPathsView.SetTitle(" Data Path ")
		ui.currentDataPathsView = dataPathsView
		ui.currentDataPathIndex = 0
		ui.currentStaticFlawInfo = nil

		techFlex := tview.NewFlex().
			AddItem(techView, 0, 1, false).
			AddItem(dataPathsView, 0, 1, false)
		return techFlex
	}

	ui.currentDataPathsView = nil
	return techView
}

// createFindingDetailInputHandler creates the input handler for the finding detail view
func (ui *UI) createFindingDetailInputHandler(finding *findings.Finding, focusableViews []tview.Primitive) func(*tcell.EventKey) *tcell.EventKey {
	// Start at the last index since we set focus to descView (Description) by default
	focusIndex := len(focusableViews) - 1

	return func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.pages.SwitchToPage("findings")
			ui.app.SetFocus(ui.findingsTable)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'm' {
				ui.showMitigationModal(finding)
				return nil
			}
			if event.Rune() == 'q' {
				ui.app.Stop()
				return nil
			}
		case tcell.KeyTab:
			focusIndex = (focusIndex + 1) % len(focusableViews)
			ui.app.SetFocus(focusableViews[focusIndex])
			return nil
		case tcell.KeyBacktab:
			focusIndex = (focusIndex - 1 + len(focusableViews)) % len(focusableViews)
			ui.app.SetFocus(focusableViews[focusIndex])
			return nil
		case tcell.KeyLeft:
			ui.handleDataPathNavigation(-1)
			return nil
		case tcell.KeyRight:
			ui.handleDataPathNavigation(1)
			return nil
		}
		return event
	}
}

// handleDataPathNavigation handles navigation between data paths for STATIC scans
func (ui *UI) handleDataPathNavigation(direction int) {
	if ui.currentStaticFlawInfo == nil || len(ui.currentStaticFlawInfo.DataPaths) <= 1 || ui.currentDataPathsView == nil {
		return
	}

	ui.currentDataPathIndex += direction
	if ui.currentDataPathIndex < 0 {
		ui.currentDataPathIndex = len(ui.currentStaticFlawInfo.DataPaths) - 1
	} else if ui.currentDataPathIndex >= len(ui.currentStaticFlawInfo.DataPaths) {
		ui.currentDataPathIndex = 0
	}

	ui.currentDataPathsView.SetTitle(fmt.Sprintf(" Data Path %d of %d ", ui.currentDataPathIndex+1, len(ui.currentStaticFlawInfo.DataPaths)))
	ui.currentDataPathsView.SetText(ui.buildDataPathsContent(ui.currentStaticFlawInfo))
	ui.currentDataPathsView.ScrollToBeginning()
}

// buildBasicInfoContent builds the content for the basic information section
//
//nolint:gocyclo // Complex display logic for multiple finding types
func (ui *UI) buildBasicInfoContent(finding *findings.Finding) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s]Issue ID:[-] [white]%d[-]\n", ui.theme.Label, finding.IssueID))
	sb.WriteString(fmt.Sprintf("[%s]Scan Type:[-] [white]%s[-]\n", ui.theme.Label, finding.ScanType))

	ui.appendCWEAndSeverity(&sb, finding)

	// Status badge
	status := extractStatus(finding)
	statusColor := ui.getStatusColorHex(finding)
	sb.WriteString(fmt.Sprintf("[%s]Status:[-] [%s]%s[-]\n", ui.theme.Label, statusColor, status))

	// Grace period expiration date
	if finding.GracePeriodExpiresDate != nil {
		sb.WriteString(fmt.Sprintf("[%s]Grace Period Expires:[-] [white]%s[-]\n\n", ui.theme.Label,
			finding.GracePeriodExpiresDate.Format("2006-01-02")))
	} else {
		sb.WriteString(fmt.Sprintf("[%s]Grace Period Expires:[-] [white]%s[-]\n\n", ui.theme.Label, TextNotAvailable))
	}

	ui.appendPolicyInfo(&sb, finding)

	return sb.String()
}

// appendCWEAndSeverity extracts and appends CWE and severity information
func (ui *UI) appendCWEAndSeverity(sb *strings.Builder, finding *findings.Finding) {
	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		return
	}

	// CWE
	if cweData, ok := details["cwe"].(map[string]interface{}); ok {
		cweID := ""
		cweName := ""
		if id, ok := cweData["id"].(float64); ok {
			cweID = fmt.Sprintf("%d", int(id))
		}
		if name, ok := cweData["name"].(string); ok {
			cweName = processCWEDescription(name)
		}
		if cweID != "" && cweName != "" {
			sb.WriteString(fmt.Sprintf("[%s]CWE:[-] [white]%s - %s[-]\n", ui.theme.Label, cweID, cweName))
		} else if cweID != "" {
			sb.WriteString(fmt.Sprintf("[%s]CWE:[-] [white]%s[-]\n", ui.theme.Label, cweID))
		}
	}

	// Severity with color
	if sev, ok := details["severity"].(float64); ok {
		sevInt := int(sev)
		sevColor := ui.getSeverityColorHex(sevInt)
		sb.WriteString(fmt.Sprintf("[%s]Severity:[-] [%s]%d[-]\n", ui.theme.Label, sevColor, sevInt))
	}

	// Exploitability (for static scans)
	if exploitability, ok := details["exploitability"].(float64); ok {
		sb.WriteString(fmt.Sprintf("[%s]Exploitability:[-] [white]%d[-]\n", ui.theme.Label, int(exploitability)))
	}
}

// appendPolicyInfo appends policy compliance information
func (ui *UI) appendPolicyInfo(sb *strings.Builder, finding *findings.Finding) {
	isMitigated := false
	if finding.FindingStatus != nil {
		if finding.FindingStatus.ResolutionStatus == findings.ResolutionApproved ||
			finding.FindingStatus.MitigationReviewStatus == findings.ResolutionApproved {
			isMitigated = true
		}
	}

	if isMitigated {
		sb.WriteString(fmt.Sprintf("[green]%s Mitigated (Approved)[-]\n", EmojiPassesPolicy))
	} else if finding.ViolatesPolicy {
		sb.WriteString(fmt.Sprintf("[red]%s Violates Policy[-]", EmojiViolatesPolicy))
	} else {
		sb.WriteString("[white]Does not affect policy[-]")
	}
}

// buildFindingStatusContent builds the content for the finding status section
func (ui *UI) buildFindingStatusContent(finding *findings.Finding) string {
	var sb strings.Builder

	if finding.FindingStatus == nil {
		sb.WriteString(fmt.Sprintf("[%s]No status information available[-]\n", ui.theme.SecondaryText))
		return sb.String()
	}

	if finding.FindingStatus.FirstFoundDate != nil {
		sb.WriteString(fmt.Sprintf("[%s]First Found:[-] [white]%s[-]\n", ui.theme.Label,
			finding.FindingStatus.FirstFoundDate.Format("2006-01-02 15:04:05")))
	}
	if finding.FindingStatus.LastSeenDate != nil {
		sb.WriteString(fmt.Sprintf("[%s]Last Seen:[-] [white]%s[-]\n", ui.theme.Label,
			finding.FindingStatus.LastSeenDate.Format("2006-01-02 15:04:05")))
	}

	if finding.FindingStatus.Status != "" {
		sb.WriteString(fmt.Sprintf("[%s]Finding Status:[-] [white]%s[-]\n", ui.theme.Label,
			finding.FindingStatus.Status))
	}

	// Mitigation status
	if finding.FindingStatus.ResolutionStatus != "" && finding.FindingStatus.ResolutionStatus != findings.ResolutionNone {
		sb.WriteString(fmt.Sprintf("[%s]Resolution:[-] [white]%s[-]\n", ui.theme.Label,
			finding.FindingStatus.ResolutionStatus))
	}
	if finding.FindingStatus.MitigationReviewStatus != "" && finding.FindingStatus.MitigationReviewStatus != findings.ResolutionNone {
		sb.WriteString(fmt.Sprintf("[%s]Mitigation:[-] [white]%s[-]\n", ui.theme.Label,
			finding.FindingStatus.MitigationReviewStatus))
	}
	if finding.FindingStatus.Resolution != "" {
		sb.WriteString(fmt.Sprintf("[%s]Resolution Type:[-] [white]%s[-]\n", ui.theme.Label,
			finding.FindingStatus.Resolution))
	}

	if finding.FindingStatus.New {
		sb.WriteString(fmt.Sprintf("[%s]New Finding:[-] [yellow]Yes[-]\n", ui.theme.Label))
	}

	return sb.String()
}

// buildDynamicScanDetails builds details for dynamic scan findings
func (ui *UI) buildDynamicScanDetails(details map[string]interface{}) string {
	var sb strings.Builder

	// Attack Vector
	if attackVector, ok := details["attack_vector"].(string); ok && attackVector != "" {
		sb.WriteString(fmt.Sprintf("[%s]Attack Vector:[-] [white]%s[-]\n\n", ui.theme.Label, attackVector))
	}

	// Hostname
	if hostname, ok := details["hostname"].(string); ok && hostname != "" {
		sb.WriteString(fmt.Sprintf("[%s]Hostname:[-] [white]%s[-]\n", ui.theme.Label, hostname))
	}

	// Port
	if port, ok := details["port"].(string); ok && port != "" {
		sb.WriteString(fmt.Sprintf("[%s]Port:[-] [white]%s[-]\n", ui.theme.Label, port))
	}

	// Path
	if path, ok := details["path"].(string); ok && path != "" {
		sb.WriteString(fmt.Sprintf("[%s]Path:[-] [white]%s[-]\n", ui.theme.Label, path))
	}

	// URL
	if url, ok := details["URL"].(string); ok && url != "" {
		sb.WriteString(fmt.Sprintf("[%s]URL:[-]\n[white]%s[-]\n\n", ui.theme.Label, url))
	}

	// Vulnerable Parameter
	if param, ok := details["vulnerable_parameter"].(string); ok && param != "" {
		sb.WriteString(fmt.Sprintf("[%s]Vulnerable Parameter:[-] [white]%s[-]\n", ui.theme.Label, param))
	}

	// Plugin
	if plugin, ok := details["plugin"].(string); ok && plugin != "" {
		sb.WriteString(fmt.Sprintf("[%s]Plugin:[-] [white]%s[-]\n", ui.theme.Label, plugin))
	}

	// Finding Category - can be string or number
	if category, ok := details["finding_category"].(string); ok && category != "" {
		sb.WriteString(fmt.Sprintf("[%s]Finding Category:[-] [white]%s[-]\n", ui.theme.Label, category))
	} else if categoryNum, ok := details["finding_category"].(float64); ok {
		sb.WriteString(fmt.Sprintf("[%s]Finding Category:[-] [white]%d[-]\n", ui.theme.Label, int(categoryNum)))
	}

	// Discovered by VSA
	if vsa, ok := details["discovered_by_vsa"].(string); ok && vsa != "" {
		sb.WriteString(fmt.Sprintf("[%s]Discovered by VSA:[-] [white]%s[-]\n", ui.theme.Label, vsa))
	}

	return sb.String()
}

// buildStaticScanDetails builds details for static scan findings
func (ui *UI) buildStaticScanDetails(details map[string]interface{}) string {
	var sb strings.Builder
	hasLineNumber := false

	// Attack Vector - shown first
	if attackVector, ok := details["attack_vector"].(string); ok && attackVector != "" {
		sb.WriteString(fmt.Sprintf("[%s]Attack Vector:[-] [white]%s[-]\n\n", ui.theme.Label, attackVector))
	}

	if filePath, ok := details["file_path"].(string); ok && filePath != "" {
		sb.WriteString(fmt.Sprintf("[%s]File Path:[-] [white]%s[-]\n", ui.theme.Label, filePath))
	}
	if lineNum, ok := details["file_line_number"].(float64); ok {
		sb.WriteString(fmt.Sprintf("[%s]Line Number:[-] [white]%d[-]\n", ui.theme.Label, int(lineNum)))
		hasLineNumber = true
	}
	if procedure, ok := details["procedure"].(string); ok && procedure != "" {
		sb.WriteString(fmt.Sprintf("[%s]Procedure:[-] [white]%s[-]\n", ui.theme.Label, procedure))
	}
	if module, ok := details["module"].(string); ok && module != "" {
		sb.WriteString(fmt.Sprintf("[%s]Module:[-] [white]%s[-]\n", ui.theme.Label, module))
	}
	// Only show relative location if line number is not provided
	if !hasLineNumber {
		if relLoc, ok := details["relative_location"].(float64); ok {
			sb.WriteString(fmt.Sprintf("[%s]Relative Location:[-] [white]%.0f%%[-]\n", ui.theme.Label, relLoc))
		}
	}

	return sb.String()
}

// buildTechnicalDetailsContent builds the content for the technical details section
func (ui *UI) buildTechnicalDetailsContent(finding *findings.Finding) string {
	var sb strings.Builder

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		sb.WriteString(fmt.Sprintf("[%s]No technical details available[-]\n", ui.theme.SecondaryText))
		return sb.String()
	}

	if finding.ScanType == findings.ScanTypeDynamic {
		sb.WriteString(ui.buildDynamicScanDetails(details))
	} else {
		sb.WriteString(ui.buildStaticScanDetails(details))
	}

	return sb.String()
}

func (ui *UI) buildAnnotationsContent(finding *findings.Finding) string {
	var sb strings.Builder

	if len(finding.Annotations) == 0 {
		sb.WriteString(fmt.Sprintf("[%s]No mitigations[-]\n", ui.theme.SecondaryText))
		return sb.String()
	}

	// Sort annotations by date descending (newest first)
	annotations := make([]findings.Annotation, len(finding.Annotations))
	copy(annotations, finding.Annotations)
	sort.Slice(annotations, func(i, j int) bool {
		// Use Created field (from API) with fallback to Date (legacy)
		dateI := annotations[i].Created
		if dateI == nil {
			dateI = annotations[i].Date
		}
		dateJ := annotations[j].Created
		if dateJ == nil {
			dateJ = annotations[j].Date
		}

		if dateI == nil {
			return false
		}
		if dateJ == nil {
			return true
		}
		return dateI.After(*dateJ)
	})

	for i, annotation := range annotations {
		if i > 0 {
			sb.WriteString(fmt.Sprintf("\n[%s]────────────────────────────────────────[-]\n\n", ui.theme.Separator))
		}

		if annotation.Action != "" {
			sb.WriteString(fmt.Sprintf("[%s]Action:[-] [white]%s[-]\n", ui.theme.Label, annotation.Action))
		}

		// Display user (prefer UserName from API, fallback to User)
		user := annotation.UserName
		if user == "" {
			user = annotation.User
		}
		if user != "" {
			sb.WriteString(fmt.Sprintf("[%s]User:[-] [white]%s[-]\n", ui.theme.Label, user))
		}

		// Display date (prefer Created from API, fallback to Date)
		date := annotation.Created
		if date == nil {
			date = annotation.Date
		}
		if date != nil {
			sb.WriteString(fmt.Sprintf("[%s]Date:[-] [white]%s[-]\n", ui.theme.Label,
				date.Format("2006-01-02 15:04:05")))
		}

		if annotation.Description != "" {
			sb.WriteString(fmt.Sprintf("[%s]Description:[-] [white]%s[-]\n", ui.theme.Label, annotation.Description))
		}
		if annotation.Comment != "" {
			sb.WriteString(fmt.Sprintf("[%s]Comment:[-]\n[white]%s[-]\n", ui.theme.Label, annotation.Comment))
		}
	}

	return sb.String()
}

func (ui *UI) buildDescriptionContent(finding *findings.Finding) string {
	var sb strings.Builder

	if finding.Description == "" {
		sb.WriteString(fmt.Sprintf("[%s]No description available[-]\n", ui.theme.SecondaryText))
		return sb.String()
	}

	description := finding.Description

	// For dynamic flaws, check if description is base64 encoded
	if finding.ScanType == findings.ScanTypeDynamic {
		if decoded, err := base64.StdEncoding.DecodeString(description); err == nil {
			// Successfully decoded, use the decoded version
			description = string(decoded)
		}
		// If decode fails, use original (it wasn't base64)
	}

	cleanDesc := cleanHTMLDescription(description)
	sb.WriteString(fmt.Sprintf("[white]%s[-]\n", cleanDesc))

	return sb.String()
}

// cleanHTMLDescription removes HTML tags and cleans up the description text
func cleanHTMLDescription(desc string) string {
	// Remove opening span tags
	spanOpenRe := regexp.MustCompile(`<span>`)
	result := spanOpenRe.ReplaceAllString(desc, "")

	// Replace closing span tags with double newlines for paragraph separation
	spanCloseRe := regexp.MustCompile(`</span>\s*`)
	result = spanCloseRe.ReplaceAllString(result, "\n\n")

	// Format References section with proper spacing
	referencesRe := regexp.MustCompile(`\s*References:`)
	result = referencesRe.ReplaceAllString(result, "\n\nReferences:\n")

	// Convert <a href="URL">TEXT</a> to "TEXT: URL\n" format
	linkRe := regexp.MustCompile(`<a href="([^"]+)">([^<]+)</a>`)
	result = linkRe.ReplaceAllStringFunc(result, func(match string) string {
		submatches := linkRe.FindStringSubmatch(match)
		if len(submatches) == 3 {
			url := submatches[1]
			linkText := submatches[2]
			return fmt.Sprintf("%s: %s\n", linkText, url)
		}
		return match
	})

	// First decode HTML entities
	decoded := html.UnescapeString(result)

	// Remove any remaining HTML tags
	decoded = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(decoded, "")

	// Clean up multiple spaces (but preserve newlines)
	decoded = regexp.MustCompile(` +`).ReplaceAllString(decoded, " ")
	decoded = strings.TrimSpace(decoded)

	return decoded
}

func (ui *UI) getSeverityColorHex(severity int) string {
	switch severity {
	case 5:
		return ui.theme.SeverityVeryHigh // Very High severity
	case 4:
		return ui.theme.SeverityHigh // High severity
	case 3:
		return ui.theme.SeverityMedium // Medium severity
	case 2:
		return ui.theme.SeverityLow // Low severity
	case 1:
		return ui.theme.SeverityVeryLow // Very Low severity
	default:
		return ui.theme.DefaultText // Default
	}
}

func (ui *UI) getResolutionColor(status findings.ResolutionStatus) string {
	switch status {
	case findings.ResolutionApproved:
		return ui.theme.Approved
	case findings.ResolutionRejected:
		return ui.theme.Rejected
	case findings.ResolutionPending:
		return ui.theme.Pending
	default:
		return ""
	}
}

func (ui *UI) getStatusColorHex(finding *findings.Finding) string {
	if finding.FindingStatus == nil {
		return ui.theme.DefaultText
	}

	// Check if it's new
	if finding.FindingStatus.New {
		return ui.theme.New // New finding
	}

	// Priority: Resolution Status > Mitigation Review Status > Status
	if finding.FindingStatus.ResolutionStatus != "" && finding.FindingStatus.ResolutionStatus != findings.ResolutionNone {
		if color := ui.getResolutionColor(finding.FindingStatus.ResolutionStatus); color != "" {
			return color
		}
	}

	if finding.FindingStatus.MitigationReviewStatus != "" && finding.FindingStatus.MitigationReviewStatus != findings.ResolutionNone {
		if color := ui.getResolutionColor(finding.FindingStatus.MitigationReviewStatus); color != "" {
			return color
		}
	}

	if finding.FindingStatus.Status != "" {
		switch finding.FindingStatus.Status {
		case findings.StatusOpen:
			return ui.theme.Error // Open
		case findings.StatusClosed:
			return ui.theme.Success // Closed
		case findings.StatusReopened:
			return ui.theme.Warning // Reopened
		}
	}

	return ui.theme.DefaultText // Default
}

// processCWEDescription extracts the short description from CWE name if it exists in brackets
// Example: "Improper Neutralization of Special Elements used in an OS Command ('OS Command Injection') "
// Returns: "OS Command Injection"
// Example: "Improper Neutralization of Script-Related HTML Tags in a Web Page (Basic XSS)"
// Returns: "Basic XSS"
func processCWEDescription(name string) string {
	// Look for pattern: ('...')
	re := regexp.MustCompile(`\('([^']+)'\)`)
	matches := re.FindStringSubmatch(name)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Look for pattern: (...)
	re2 := regexp.MustCompile(`\(([^)]+)\)`)
	matches2 := re2.FindStringSubmatch(name)
	if len(matches2) > 1 {
		return strings.TrimSpace(matches2[1])
	}

	return name
}

// loadAndDisplayStaticFlawInfo fetches data paths and conditionally displays them
func (ui *UI) loadAndDisplayStaticFlawInfo(finding *findings.Finding, dataPathsView *tview.TextView) {
	// NOTE: API bug - the static_flaw_info endpoint returns 404 when context parameter is provided
	// for sandbox findings ("Build does not have static flaws"), even though the Swaggerhub API
	// definition documents the context parameter as supported. Workaround: always omit context.
	// The finding data is the same regardless of context (same flaw ID).
	// See: TestGetSandboxFindingStaticFlawInfo integration test

	// Fetch static flaw info without context parameter (API bug workaround)
	staticFlawInfo, err := ui.findingsService.GetStaticFlawInfo(ui.selectedApp.GUID, finding.IssueID, "")
	if err != nil {
		ui.app.QueueUpdateDraw(func() {
			dataPathsView.SetText(fmt.Sprintf("[red]Error loading data paths: %v[-]", err))
			dataPathsView.ScrollToBeginning()
		})
		return
	}

	// Store the static flaw info for navigation
	ui.currentStaticFlawInfo = staticFlawInfo
	ui.currentDataPathIndex = 0

	ui.app.QueueUpdateDraw(func() {
		// Update title and content based on whether data paths exist
		if staticFlawInfo != nil && len(staticFlawInfo.DataPaths) > 0 {
			// Update title based on count
			if len(staticFlawInfo.DataPaths) == 1 {
				dataPathsView.SetTitle(" Data Path ")
			} else {
				dataPathsView.SetTitle(fmt.Sprintf(" Data Path %d of %d ", ui.currentDataPathIndex+1, len(staticFlawInfo.DataPaths)))
			}
		} else {
			// No data paths available
			dataPathsView.SetTitle(" Data Path ")
		}

		// Build and display content
		content := ui.buildDataPathsContent(staticFlawInfo)
		dataPathsView.SetText(content)
		dataPathsView.ScrollToBeginning()
	})
}

// buildDataPathsContent formats static flaw data paths for display
func (ui *UI) buildDataPathsContent(staticFlawInfo *findings.StaticFlawInfo) string {
	if staticFlawInfo == nil || len(staticFlawInfo.DataPaths) == 0 {
		return fmt.Sprintf("[%s]No data paths available[-]", ui.theme.SecondaryText)
	}

	var sb strings.Builder

	// Show navigation hint if multiple paths
	if len(staticFlawInfo.DataPaths) > 1 {
		sb.WriteString(fmt.Sprintf("[%s]Use ← → to navigate[-]\n\n", ui.theme.DimmedText))
	}

	// Display only the current data path
	dataPath := staticFlawInfo.DataPaths[ui.currentDataPathIndex]

	// Module and entry point
	sb.WriteString(fmt.Sprintf("[yellow]Module:[-] [white]%s[-]\n", dataPath.ModuleName))
	sb.WriteString(fmt.Sprintf("[yellow]Steps:[-] [white]%d[-]\n", dataPath.Steps))

	if dataPath.LocalPath != "" {
		sb.WriteString(fmt.Sprintf("[yellow]Path:[-] [white]%s[-]\n", dataPath.LocalPath))
	}

	if dataPath.FunctionName != "" {
		sb.WriteString(fmt.Sprintf("[yellow]Function:[-] [white]%s[-] [%s]at line %d[-]\n", dataPath.FunctionName, ui.theme.SecondaryText,
			dataPath.LineNumber))
	}

	// Call stack
	if len(dataPath.Calls) > 0 {
		sb.WriteString(fmt.Sprintf("\n[%s]Call Stack:[-]\n", ui.theme.Label))

		// Sort calls by data_path in descending order (most recent first)
		calls := make([]findings.Call, len(dataPath.Calls))
		copy(calls, dataPath.Calls)
		sort.Slice(calls, func(i, j int) bool {
			return calls[i].DataPath > calls[j].DataPath
		})

		for _, call := range calls {
			sb.WriteString(fmt.Sprintf("  [%s]→ Step %d:[-] [white]%s[-]\n", ui.theme.SecondaryText,
				call.DataPath, call.FunctionName))

			filePath := call.FilePath
			if filePath == "" {
				filePath = call.FileName
			}

			if filePath != "" {
				sb.WriteString(fmt.Sprintf("    [%s]%s:%d[-]\n", ui.theme.DimmedText, filePath, call.LineNumber))
			}
		}
	}

	return sb.String()
}

// modal creates a centered modal primitive
// Use width/height of 0 for default centered size, or positive values for custom proportions
func modal(p tview.Primitive, widthProportion, heightProportion int) tview.Primitive {
	if widthProportion <= 0 {
		widthProportion = 3 // Default: modal takes 3 parts, sides take 1 each (3:1:1 = 60%)
	}
	if heightProportion <= 0 {
		heightProportion = 3 // Default: modal takes 3 parts, top/bottom take 1 each
	}

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, 0, heightProportion, true).
			AddItem(nil, 0, 1, false), 0, widthProportion, true).
		AddItem(nil, 0, 1, false)
}

// getAvailableAnnotationActions determines which annotation actions are available
func (ui *UI) getAvailableAnnotationActions(finding *findings.Finding) []string {
	baseActions := []string{"COMMENT", "FP", "APPDESIGN", "OSENV", "NETENV"}

	// Check if user has approveMitigations permission
	principal, err := ui.identityService.GetPrincipal(context.Background())
	if err != nil || principal == nil {
		return baseActions
	}

	hasApproveMitigations := false
	for _, perm := range principal.Permissions {
		if perm == "approveMitigations" {
			hasApproveMitigations = true
			break
		}
	}

	if !hasApproveMitigations {
		return baseActions
	}

	// Find the last non-COMMENT mitigation action
	lastMitigationAction := ui.getLastNonCommentAction(finding)
	if lastMitigationAction == "" {
		return baseActions
	}

	// Check if last mitigation action qualifies for approval actions
	if ui.isApprovableAction(lastMitigationAction) {
		return append(baseActions, "REJECTED", "ACCEPTED")
	}

	return baseActions
}

// getLastNonCommentAction finds the most recent non-comment annotation action
func (ui *UI) getLastNonCommentAction(finding *findings.Finding) string {
	for i := len(finding.Annotations) - 1; i >= 0; i-- {
		action := finding.Annotations[i].Action
		if action != "COMMENT" && action != "" {
			return action
		}
	}
	return ""
}

// isApprovableAction checks if an action qualifies for approval
func (ui *UI) isApprovableAction(action string) bool {
	return action == "APPDESIGN" || action == "NETENV" ||
		action == "OSENV" || action == "FP" ||
		action == "LIBRARY" || action == "ACCEPTRISK"
}

func (ui *UI) setupMitigationModalInputCapture(
	finding *findings.Finding,
	commentTextArea *tview.TextArea,
	actionDropdown *tview.DropDown,
	statusText *tview.TextView,
	mitigationView *tview.TextView,
	focusables []tview.Primitive,
	currentFocus *int,
) func(*tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.pages.RemovePage("mitigation-modal")
			ui.pages.SwitchToPage("finding_detail")
			return nil
		case tcell.KeyTab:
			*currentFocus = (*currentFocus + 1) % len(focusables)
			ui.app.SetFocus(focusables[*currentFocus])
			return nil
		case tcell.KeyBacktab:
			*currentFocus = (*currentFocus - 1 + len(focusables)) % len(focusables)
			ui.app.SetFocus(focusables[*currentFocus])
			return nil
		case tcell.KeyCtrlS:
			commentText := commentTextArea.GetText()
			if strings.TrimSpace(commentText) == "" {
				statusText.SetText(fmt.Sprintf("[%s]Error: Comment cannot be empty[-]  [%s]ESC/q[-] Close", ui.theme.Error, ui.theme.Info))
				return nil
			}

			_, actionText := actionDropdown.GetCurrentOption()
			statusText.SetText(fmt.Sprintf("[%s]Submitting...[-]", ui.theme.Pending))
			commentTextArea.SetDisabled(true)

			go ui.submitAnnotationCommentInModal(finding, commentText, actionText, statusText, commentTextArea, mitigationView)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				ui.pages.RemovePage("mitigation-modal")
				ui.pages.SwitchToPage("finding_detail")
				return nil
			}
		}
		return event
	}
}

// showMitigationModal displays mitigations in a modal dialog with comment input
func (ui *UI) showMitigationModal(finding *findings.Finding) {
	if finding == nil {
		return
	}

	// Determine available actions
	actionOptions := ui.getAvailableAnnotationActions(finding)

	// Create dropdown for annotation action type
	actionDropdown := tview.NewDropDown().
		SetLabel("Action: ").
		SetOptions(actionOptions, nil).
		SetCurrentOption(0).
		SetLabelColor(tcell.GetColor(ui.theme.Label)).
		SetFieldTextColor(tcell.GetColor(ui.theme.DropDownText)).
		SetFieldBackgroundColor(tcell.GetColor(ui.theme.DropDownBackground))
	actionDropdown.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.GetColor(ui.theme.DropDownText)).Background(tcell.GetColor(ui.theme.DropDownBackground)),
		tcell.StyleDefault.Foreground(tcell.GetColor(ui.theme.DropDownSelectedForeground)).Background(tcell.GetColor(ui.theme.DropDownSelectedBackground)))
	actionDropdown.SetBorder(true).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	actionDropdown.SetFocusFunc(func() {
		actionDropdown.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	actionDropdown.SetBlurFunc(func() {
		actionDropdown.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create multi-line text area for comment
	commentTextArea := tview.NewTextArea().
		SetPlaceholder("Enter your comment here...")
	commentTextArea.SetBorder(true).
		SetBorderColor(tcell.GetColor(ui.theme.Border)).
		SetBorderPadding(0, 0, 1, 1).
		SetTitle(" Comment Text ").
		SetTitleAlign(tview.AlignLeft)

	commentTextArea.SetFocusFunc(func() {
		commentTextArea.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	commentTextArea.SetBlurFunc(func() {
		commentTextArea.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create a text view for the mitigations
	mitigationView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	mitigationView.SetBorder(true).
		SetTitle(" Existing Mitigations ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border)).
		SetBorderPadding(1, 1, 2, 2)

	mitigationView.SetFocusFunc(func() {
		mitigationView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	mitigationView.SetBlurFunc(func() {
		mitigationView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	mitigationView.SetText(ui.buildAnnotationsContent(finding))

	// Create status text
	statusText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("[%s]Ctrl+S[-] Submit Annotation  [%s]Tab[-] Navigate  [%s]ESC/q[-] Close", ui.theme.Info, ui.theme.Info, ui.theme.Info))
	statusText.SetBorder(false)

	// Create layout
	modalContent := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(actionDropdown, 3, 0, false).
		AddItem(commentTextArea, 6, 0, true).
		AddItem(mitigationView, 0, 1, false).
		AddItem(statusText, 1, 0, false)

	modalContent.SetBorder(true).
		SetTitle(" Mitigations ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))

	modalPrimitive := modal(modalContent, 6, 6)

	// Setup navigation
	focusables := []tview.Primitive{actionDropdown, commentTextArea, mitigationView}
	currentFocus := 0

	modalContent.SetInputCapture(ui.setupMitigationModalInputCapture(
		finding, commentTextArea, actionDropdown, statusText, mitigationView, focusables, &currentFocus,
	))

	ui.pages.AddPage("mitigation-modal", modalPrimitive, true, true)
	ui.app.SetFocus(commentTextArea)
}

// submitAnnotationCommentInModal submits the annotation and refreshes the modal
func (ui *UI) submitAnnotationCommentInModal(finding *findings.Finding, comment, action string, statusText *tview.TextView, textArea *tview.TextArea, mitigationView *tview.TextView) {
	// Determine context (sandbox GUID or empty for policy)
	contextGUID := ""
	if finding.ContextType == findings.ContextTypeSandbox && ui.selectionIndex >= 0 && ui.selectionIndex < len(ui.sandboxes) {
		contextGUID = ui.sandboxes[ui.selectionIndex].GUID
	}

	// Create the annotation
	annotation := &annotations.AnnotationData{
		IssueList: fmt.Sprintf("%d", finding.IssueID),
		Comment:   comment,
		Action:    action,
	}

	opts := &annotations.CreateAnnotationOptions{
		Context: contextGUID,
	}

	_, err := ui.annotationsService.CreateAnnotation(ui.selectedApp.GUID, annotation, opts)

	ui.app.QueueUpdateDraw(func() {
		if err != nil {
			// Format error message - check if it's an HTTPError
			errorMsg := err.Error()

			// Unwrap to find HTTPError in the error chain
			var httpErr *veracode.HTTPError
			if errors.As(err, &httpErr) {
				// Parse the JSON response to get title and detail
				var errorResp annotations.AnnotationErrorResponse
				if parseErr := json.Unmarshal(httpErr.Body, &errorResp); parseErr == nil {
					if len(errorResp.Embedded.APIErrors) > 0 {
						apiErr := errorResp.Embedded.APIErrors[0]
						// Format: {HTTP Code}:{Detail}
						errorMsg = fmt.Sprintf("%d:%s", httpErr.StatusCode, apiErr.Detail)
					}
				}
			}

			statusText.SetText(fmt.Sprintf("[%s]Error: %s  [%s]Press ESC to close[-]", ui.theme.Error, errorMsg, ui.theme.Info))
			textArea.SetDisabled(false)
		} else {
			// Success - update in-memory data
			now := time.Now()

			// Get current user name if available
			userName := "Current User"
			if ui.identityService != nil {
				principal, err := ui.identityService.GetPrincipal(context.Background())
				if err == nil && principal != nil {
					userName = principal.Username
				}
			}

			// Create new annotation object
			newAnnotation := findings.Annotation{
				Action:   action,
				Comment:  comment,
				Created:  &now,
				UserName: userName,
			}

			// Update the selected finding (which is a pointer to an element in findings)
			// Updating selectedFinding updates the master findings list automatically
			if ui.selectedFinding != nil {
				ui.selectedFinding.Annotations = append(ui.selectedFinding.Annotations, newAnnotation)
			}

			// Refresh both the mitigation view and the main finding annotations view
			mitigationView.SetText(ui.buildAnnotationsContent(finding))
			mitigationView.ScrollToBeginning()

			// Update the main finding detail annotations view if it exists
			if ui.findingAnnotationsView != nil {
				ui.findingAnnotationsView.SetText(ui.buildAnnotationsContent(finding))
			}

			// Update the finding row in the table to reflect the new status
			ui.updateFindingRowInTable(finding)

			// Show success message
			statusText.SetText(fmt.Sprintf("[%s]✓ Annotation submitted!  [%s]Ctrl+S[-] Submit Another  [%s]ESC/q[-] Close", ui.theme.Success, ui.theme.Info, ui.theme.Info))
			textArea.SetDisabled(false)
			textArea.SetText("", true) // Clear the text area
		}
	})
}
