package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dipsylala/veracode-tui/config"
	"github.com/dipsylala/veracode-tui/services/annotations"
	"github.com/dipsylala/veracode-tui/services/applications"
	"github.com/dipsylala/veracode-tui/services/findings"
	"github.com/dipsylala/veracode-tui/services/identity"
	"github.com/dipsylala/veracode-tui/ui"
	"github.com/dipsylala/veracode-tui/veracode"
)

// Version is the application version, can be set at build time with -ldflags "-X main.Version=x.y.z"
var Version = "dev"

func main() {
	healthcheck := flag.Bool("healthcheck", false, "Perform a healthcheck and exit (does not open TUI)")
	version := flag.Bool("version", false, "Display version information")
	help := flag.Bool("help", false, "Display usage information")
	noColor := flag.Bool("no-color", false, "Disable colors (monochrome mode)")
	theme := flag.String("theme", "default", "Color theme to use (default, bw, hotdog, matrix)")
	debugLog := flag.String("debug-log", "", "Enable debug logging of REST requests/responses to the specified file")
	flag.Parse()

	if *help {
		fmt.Println("Veracode TUI - Terminal User Interface for Veracode API")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  veracode-tui                       Start the interactive TUI")
		fmt.Println("  veracode-tui --healthcheck         Test API connectivity and credentials")
		fmt.Println("  veracode-tui --version             Show version information")
		fmt.Println("  veracode-tui --no-color            Disable colors (monochrome mode)")
		fmt.Println("  veracode-tui --theme <name>        Set color theme: default, bw, hotdog, matrix (default: default)")
		fmt.Println("  veracode-tui --help                Show this help message")
		fmt.Println("  veracode-tui --debug-log <file>    Log all REST requests/responses to file")
		fmt.Println()
		fmt.Println("Configuration:")
		fmt.Println("  Reads credentials from ~/.veracode/veracode.yml")
		fmt.Println()
		fmt.Println("Environment Variables:")
		fmt.Println("  NO_COLOR                           When set, disables colors (overrides --no-color)")
		fmt.Println()
		os.Exit(0)
	}

	if *version {
		fmt.Printf("Veracode TUI v%s\n", Version)
		fmt.Println("Built with Go - A read-only interface to Veracode API")
		os.Exit(0)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please ensure ~/.veracode/veracode.yml exists with valid API credentials\n")
		os.Exit(1)
	}

	keyID, keySecret := cfg.GetAPICredentials()

	client := veracode.NewClient(keyID, keySecret)

	if *debugLog != "" {
		if err := client.EnableDebugLog(*debugLog); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to enable debug logging: %v\n", err)
		} else {
			fmt.Printf("Debug logging enabled: %s\n", *debugLog)
		}
	}

	if *healthcheck {
		fmt.Println("🏥 Performing Veracode API healthcheck...")
		if err := client.HealthCheck(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Healthcheck failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Healthcheck successful - API is operational and credentials are valid")
		os.Exit(0)
	}

	appService := applications.NewService(client)
	findingsService := findings.NewService(client)
	identityService := identity.NewService(client)
	annotationsService := annotations.NewService(client)

	var selectedTheme *ui.Theme
	if os.Getenv("NO_COLOR") != "" || *noColor {
		selectedTheme = ui.MonochromeTheme()
	} else {
		switch *theme {
		case "bw":
			selectedTheme = ui.MonochromeTheme()
		case "hotdog":
			selectedTheme = ui.HotdogTheme()
		case "matrix":
			selectedTheme = ui.MatrixTheme()
		default:
			selectedTheme = ui.DefaultTheme()
		}
	}

	tui := ui.NewUI(appService, findingsService, identityService, annotationsService, selectedTheme)
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
