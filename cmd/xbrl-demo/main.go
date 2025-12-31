package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/RxDataLab/go-edgar"
)

func main() {
	// Usage
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path-to-10k.htm>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  %s testdata/xbrl/moderna_10k/input.htm\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Extracts key financial metrics from SEC 10-K/10-Q filings (inline XBRL).\n")
		os.Exit(1)
	}

	filePath := os.Args[1]

	// Load the filing
	fmt.Fprintf(os.Stderr, "Loading: %s\n", filePath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "File size: %.2f MB\n", float64(len(data))/1024/1024)

	// Detect XBRL type
	xbrlType := edgar.DetectXBRLType(data)
	fmt.Fprintf(os.Stderr, "XBRL format: %s\n", xbrlType)

	if xbrlType == "unknown" {
		fmt.Fprintf(os.Stderr, "Error: Not a recognized XBRL file\n")
		os.Exit(1)
	}

	// Parse
	fmt.Fprintf(os.Stderr, "Parsing...\n")
	xbrl, err := edgar.ParseXBRLAuto(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing XBRL: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✓ Parsed successfully\n")
	fmt.Fprintf(os.Stderr, "  Contexts: %d\n", len(xbrl.Contexts))
	fmt.Fprintf(os.Stderr, "  Units: %d\n", len(xbrl.Units))
	fmt.Fprintf(os.Stderr, "  Facts: %d\n", len(xbrl.Facts))
	fmt.Fprintf(os.Stderr, "\n")

	// Extract snapshot
	fmt.Fprintf(os.Stderr, "Extracting financial metrics...\n")
	snapshot, err := xbrl.GetSnapshot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting snapshot: %v\n", err)
		os.Exit(1)
	}

	// Display table
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("           Financial Snapshot")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("Period: %s\n\n", snapshot.PeriodEnd.Format("2006-01-02"))

	fmt.Printf("%-35s %15s\n", "Metric", "Value")
	fmt.Printf("%-35s %15s\n", "─────────────────────────────────", "──────────────")

	printMetric("Cash & Equivalents", snapshot.Cash)
	printMetric("Total Debt", snapshot.TotalDebt)
	printMetric("Revenue", snapshot.Revenue)
	printMetric("R&D Expense", snapshot.RDExpense)
	printMetric("G&A Expense", snapshot.GAExpense)
	printMetric("Burn Rate (R&D + G&A)", snapshot.Burn)

	if snapshot.DilutedShares > 0 {
		millions := snapshot.DilutedShares / 1_000_000
		fmt.Printf("%-35s %12.1fM\n", "Diluted Shares", millions)
	}

	if snapshot.RunwayQuarters > 0 {
		fmt.Printf("%-35s %11.1f qtrs\n", "Runway (Cash/Quarterly Burn)", snapshot.RunwayQuarters)
	}

	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()

	// JSON output
	fmt.Fprintf(os.Stderr, "\nJSON Output:\n")
	fmt.Fprintf(os.Stderr, "────────────────────────────────────────────────\n")

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func printMetric(label string, value float64) {
	if value == 0 {
		fmt.Printf("%-35s %15s\n", label, "$0")
		return
	}

	billions := value / 1_000_000_000
	millions := value / 1_000_000

	if billions >= 1 {
		fmt.Printf("%-35s %12.2fB\n", label, billions)
	} else if millions >= 1 {
		fmt.Printf("%-35s %12.1fM\n", label, millions)
	} else {
		fmt.Printf("%-35s %15.0f\n", label, value)
	}
}
