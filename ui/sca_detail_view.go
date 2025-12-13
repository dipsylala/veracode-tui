package ui

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dipsylala/veracode-tui/services/findings"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// showSCAFindingDetail displays the detailed view for a selected SCA CVE finding
func (ui *UI) showSCAFindingDetail() {
	if ui.selectedFinding == nil {
		return
	}

	finding := ui.selectedFinding

	// Determine context name
	contextName := DefaultContextName
	if finding.ContextType == findings.ContextTypeSandbox && ui.selectionIndex >= 0 && ui.selectionIndex < len(ui.sandboxes) {
		contextName = ui.sandboxes[ui.selectionIndex].Name
	}

	// Get application name
	appName := DefaultApplicationName
	if ui.selectedApp != nil && ui.selectedApp.Profile != nil {
		appName = ui.selectedApp.Profile.Name
	}

	// Create title view
	titleView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	titleView.SetText(fmt.Sprintf("[white::b]CVE Details - %s - %s", appName, contextName))
	titleView.SetBorder(false)

	// Create left column (Basic Information & Policy)
	leftView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWordWrap(true)
	leftView.SetBorder(true).
		SetTitle(" Basic Information & Policy ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	leftView.SetFocusFunc(func() {
		leftView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	leftView.SetBlurFunc(func() {
		leftView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create right column (Finding Status)
	rightView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWordWrap(true)
	rightView.SetBorder(true).
		SetTitle(" Finding Status ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	rightView.SetFocusFunc(func() {
		rightView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	rightView.SetBlurFunc(func() {
		rightView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create CVE details view
	cveDetailsView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWordWrap(true)
	cveDetailsView.SetBorder(true).
		SetTitle(" CVE Details ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	cveDetailsView.SetFocusFunc(func() {
		cveDetailsView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	cveDetailsView.SetBlurFunc(func() {
		cveDetailsView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create component details view
	componentDetailsView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	componentDetailsView.SetBorder(true).
		SetTitle(" Component Details ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	componentDetailsView.SetFocusFunc(func() {
		componentDetailsView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	componentDetailsView.SetBlurFunc(func() {
		componentDetailsView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Create description view (scrollable)
	descView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	descView.SetBorder(true).
		SetTitle(" Description ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.GetColor(ui.theme.Border))
	descView.SetFocusFunc(func() {
		descView.SetBorderColor(tcell.GetColor(ui.theme.BorderFocused))
	})
	descView.SetBlurFunc(func() {
		descView.SetBorderColor(tcell.GetColor(ui.theme.Border))
	})

	// Build content
	leftContent := ui.buildSCABasicInfoContent(finding)
	rightContent := ui.buildFindingStatusContent(finding)
	cveContent := ui.buildSCACVEDetailsContent(finding)
	componentContent := ui.buildComponentDetailsContent(finding)
	descContent := ui.buildDescriptionContent(finding)

	leftView.SetText(leftContent)
	rightView.SetText(rightContent)
	cveDetailsView.SetText(cveContent)
	componentDetailsView.SetText(componentContent)
	descView.SetText(descContent)

	// Create main layout
	topRow := tview.NewFlex().
		AddItem(leftView, 0, 1, false).
		AddItem(rightView, 0, 1, false)

	// Create middle row with CVE and Component details side-by-side
	middleRow := tview.NewFlex().
		AddItem(cveDetailsView, 0, 1, false).
		AddItem(componentDetailsView, 0, 1, false)

	// Create keyboard shortcuts bar
	shortcutsBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("[%s]ESC[-] Back  [%s]q[-] Quit  [%s]Tab[-] Navigate",
			ui.theme.Info, ui.theme.Info, ui.theme.Info))
	shortcutsBar.SetBorder(false)

	// Focusable views
	focusableViews := []tview.Primitive{
		leftView,
		rightView,
		cveDetailsView,
		componentDetailsView,
		descView,
	}

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(titleView, 1, 0, false).
		AddItem(topRow, 11, 0, false).
		AddItem(middleRow, 12, 0, false).
		AddItem(descView, 0, 1, true).
		AddItem(shortcutsBar, 1, 0, false)

	// Set up input handling
	focusIndex := len(focusableViews) - 1 // Start at description
	mainLayout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.pages.SwitchToPage("findings")
			ui.app.SetFocus(ui.findingsTable)
			return nil
		case tcell.KeyRune:
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
		}
		return event
	})

	ui.findingDetailView = mainLayout

	// Add or update the page
	if ui.pages.HasPage("finding_detail") {
		ui.pages.RemovePage("finding_detail")
	}
	ui.pages.AddPage("finding_detail", ui.findingDetailView, true, false)
	ui.pages.SwitchToPage("finding_detail")
	ui.app.SetFocus(descView)
}

// buildSCABasicInfoContent builds the content for the basic information section for SCA findings
func (ui *UI) buildSCABasicInfoContent(finding *findings.Finding) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s]Scan Type:[-] [white]%s[-]\n", ui.theme.Label, finding.ScanType))

	// Component and version
	details, ok := finding.FindingDetails.(map[string]interface{})
	if ok {
		if component, ok := details["component_filename"].(string); ok && component != "" {
			sb.WriteString(fmt.Sprintf("[%s]Component:[-] [white]%s[-]\n", ui.theme.Label, component))
		}
		if version, ok := details["version"].(string); ok && version != "" {
			sb.WriteString(fmt.Sprintf("[%s]Version:[-] [white]%s[-]\n", ui.theme.Label, version))
		}

		// Severity with color
		if sev, ok := details["severity"].(float64); ok {
			sevInt := int(sev)
			sevColor := ui.getSeverityColorHex(sevInt)
			sb.WriteString(fmt.Sprintf("[%s]Severity:[-] [%s]%d[-]\n", ui.theme.Label, sevColor, sevInt))
		}
	}

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

	// Policy info
	if finding.ViolatesPolicy {
		sb.WriteString(fmt.Sprintf("[red]%s Violates Policy[-]\n", EmojiViolatesPolicy))
	} else {
		sb.WriteString("[white]Does not affect policy[-]\n")
	}

	return sb.String()
}

// buildSCACVEDetailsContent builds the CVE-specific details for SCA findings
func (ui *UI) buildSCACVEDetailsContent(finding *findings.Finding) string {
	var sb strings.Builder

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		sb.WriteString(fmt.Sprintf("[%s]No CVE details available[-]\n", ui.theme.SecondaryText))
		return sb.String()
	}

	// CVE information
	var cveName string
	if cveData, ok := details["cve"].(map[string]interface{}); ok {
		if name, ok := cveData["name"].(string); ok && name != "" {
			cveName = name
			sb.WriteString(fmt.Sprintf("[%s]CVE:[-] [white]%s[-]\n", ui.theme.Label, cveName))
		}
		if cveHref, ok := cveData["href"].(string); ok && cveHref != "" {
			sb.WriteString(fmt.Sprintf("[%s]Link:[-] [:::%s]%s[:::-]\n",
				ui.theme.Label, cveHref, cveHref))
		}

		// Add useful links if CVE name exists
		if cveName != "" {
			sb.WriteString(fmt.Sprintf("\n[%s]Useful Links:[-]\n", ui.theme.Label))
			encodedCVE := url.QueryEscape(cveName)
			veracodeSearchURL := fmt.Sprintf("https://sca.analysiscenter.veracode.com/vulnerability-database/search#query=%s", encodedCVE)
			exploitDBSearchURL := fmt.Sprintf("https://www.exploit-db.com/search?cve=%s", encodedCVE)
			sb.WriteString(fmt.Sprintf("  [:::%s]Veracode SCA[:::-]\n", veracodeSearchURL))
			sb.WriteString(fmt.Sprintf("  [:::%s]Exploit-DB[:::-]\n", exploitDBSearchURL))
		}

	}
	sb.WriteString("\n")

	// CWE information
	if cweData, ok := details["cwe"].(map[string]interface{}); ok {
		if cweID, ok := cweData["id"].(string); ok && cweID != "" {
			sb.WriteString(fmt.Sprintf("[%s]CWE:[-] [white]%s[-]\n", ui.theme.Label, cweID))
		}
		if cweName, ok := cweData["name"].(string); ok && cweName != "" {
			sb.WriteString(fmt.Sprintf("[%s]CWE Name:[-] [white]%s[-]\n", ui.theme.Label, cweName))
		}
	}

	return sb.String()
}

// formatComponentPaths formats component path information for display
func (ui *UI) formatComponentPaths(componentPaths []interface{}) string {
	var sb strings.Builder

	if len(componentPaths) == 0 {
		return ""
	}

	pathLabel := "Component Path"
	if len(componentPaths) > 1 {
		pathLabel = "Component Paths"
	}
	sb.WriteString(fmt.Sprintf("[%s]%s:[-]\n", ui.theme.Label, pathLabel))

	for _, pathItem := range componentPaths {
		pathObj, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}

		if pathStr, ok := pathObj["path"].(string); ok {
			sb.WriteString(fmt.Sprintf("  [white]%s[-]\n", pathStr))
		}
	}
	sb.WriteString("\n")

	return sb.String()
}

// formatLicenses formats license information for display
func (ui *UI) formatLicenses(licenses []interface{}) string {
	var sb strings.Builder

	if len(licenses) == 0 {
		return fmt.Sprintf("[%s]Licenses:[-] [white]None[-]\n", ui.theme.Label)
	}

	licenseLabel := "License"
	if len(licenses) > 1 {
		licenseLabel = "Licenses"
	}
	sb.WriteString(fmt.Sprintf("[%s]%s:[-]\n", ui.theme.Label, licenseLabel))

	for _, licenseItem := range licenses {
		licenseObj, ok := licenseItem.(map[string]interface{})
		if !ok {
			continue
		}

		var licenseID, riskRating string
		if id, ok := licenseObj["license_id"].(string); ok {
			licenseID = id
		}
		if risk, ok := licenseObj["risk_rating"].(string); ok {
			riskRating = risk
		}

		if licenseID == "" {
			continue
		}

		if riskRating != "" {
			sb.WriteString(fmt.Sprintf("  [white]%s[-] [%s](Risk: %s)[-]\n", licenseID, ui.theme.SecondaryText, riskRating))
		} else {
			sb.WriteString(fmt.Sprintf("  [white]%s[-]\n", licenseID))
		}
	}

	return sb.String()
}

// buildComponentDetailsContent builds the component-specific details for SCA findings
func (ui *UI) buildComponentDetailsContent(finding *findings.Finding) string {
	var sb strings.Builder

	details, ok := finding.FindingDetails.(map[string]interface{})
	if !ok {
		sb.WriteString(fmt.Sprintf("[%s]No component details available[-]\n", ui.theme.SecondaryText))
		return sb.String()
	}

	// Component path(s) - array of objects with "path" property
	if componentPaths, ok := details["component_path"].([]interface{}); ok {
		sb.WriteString(ui.formatComponentPaths(componentPaths))
	}

	// Language
	if language, ok := details["language"].(string); ok && language != "" {
		sb.WriteString(fmt.Sprintf("[%s]Language:[-] [white]%s[-]\n", ui.theme.Label, language))
	}

	// Licenses - array of objects with license_id and risk_rating
	if licenses, ok := details["licenses"].([]interface{}); ok {
		sb.WriteString(ui.formatLicenses(licenses))
	}

	// Vulnerable methods if available
	if vulnerableMethods, ok := details["vulnerable_methods"].(string); ok && vulnerableMethods != "" {
		sb.WriteString(fmt.Sprintf("\n[%s]Vulnerable Methods:[-]\n[white]%s[-]\n", ui.theme.Label, vulnerableMethods))
	}

	// Metadata if available
	if metadata, ok := details["metadata"].(map[string]interface{}); ok && len(metadata) > 0 {
		sb.WriteString(fmt.Sprintf("\n[%s]Metadata:[-]\n", ui.theme.Label))
		for key, value := range metadata {
			sb.WriteString(fmt.Sprintf("  [%s]%s:[-] [white]%v[-]\n", ui.theme.Label, key, value))
		}
	}

	return sb.String()
}
