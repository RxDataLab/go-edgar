package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/RxDataLab/go-edgar"
)

func main() {
	// Define flags
	var (
		// Single file mode
		saveOriginal bool
		outputPath   string
		email        string
		pretty       bool

		// Batch mode
		cik              string
		formType         string
		dateFrom         string
		dateTo           string
		includePaginated bool
		listOnly         bool
	)

	flag.BoolVar(&saveOriginal, "save-original", false, "Save the original XML/HTML file")
	flag.BoolVar(&saveOriginal, "s", false, "Save the original XML/HTML file (shorthand)")
	flag.StringVar(&outputPath, "output", "", "Output JSON file path (default: stdout)")
	flag.StringVar(&outputPath, "o", "", "Output JSON file path (shorthand)")
	flag.StringVar(&email, "email", "", "Email for SEC User-Agent header (or use SEC_EMAIL env var)")
	flag.StringVar(&email, "e", "", "Email for SEC User-Agent (shorthand)")
	flag.BoolVar(&pretty, "pretty", false, "Pretty print table output (XBRL only)")

	// Batch mode flags
	flag.StringVar(&cik, "cik", "", "CIK to fetch filings for (batch mode)")
	flag.StringVar(&formType, "form", "4", "Form type to fetch (default: 4)")
	flag.StringVar(&dateFrom, "from", "", "Start date for filtering (YYYY-MM-DD)")
	flag.StringVar(&dateTo, "to", "", "End date for filtering (YYYY-MM-DD)")
	flag.BoolVar(&includePaginated, "all", false, "Include all paginated filings (can be slow)")
	flag.BoolVar(&listOnly, "list-only", false, "List filings without downloading/parsing (batch mode only)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: goedgar [options] [<source>]\n\n")
		fmt.Fprintf(os.Stderr, "Parse SEC forms from URL, file path, or fetch by CIK.\n\n")
		fmt.Fprintf(os.Stderr, "Modes:\n")
		fmt.Fprintf(os.Stderr, "  Single file: goedgar [options] <source>\n")
		fmt.Fprintf(os.Stderr, "  Batch mode:  goedgar --cik <CIK> [--form 4] [--from DATE] [--to DATE]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Single file\n")
		fmt.Fprintf(os.Stderr, "  goedgar https://www.sec.gov/Archives/edgar/data/.../ownership.xml\n")
		fmt.Fprintf(os.Stderr, "  goedgar ./ownership.xml\n\n")
		fmt.Fprintf(os.Stderr, "  # Batch mode (Form 4)\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 0000078003 --form 4 --from 2025-01-01 --to 2025-06-30\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 1631574 --form 4  # All recent Form 4s\n\n")
		fmt.Fprintf(os.Stderr, "  # Schedule 13D/G (includes amendments 13D/A, 13G/A)\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 1496099 --form 13D  # All 13D filings (includes 13D/A)\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 1496099 --form 13G --list-only  # List 13G filings without parsing\n\n")
		fmt.Fprintf(os.Stderr, "  # List mode (just show URLs, don't parse)\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 1631574 --form 4 --list-only\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 1682852 --form 10-K --from 2023-01-01 --list-only\n\n")
		fmt.Fprintf(os.Stderr, "  # 10-K/10-Q (XBRL)\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 1682852 --form 10-K  # Latest 10-K\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 1682852 --form 10-K --from 2023-01-01  # All 10-Ks from 2023\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 1682852 --form 10-Q --pretty  # Latest 10-Q with table\n\n")
		fmt.Fprintf(os.Stderr, "Environment:\n")
		fmt.Fprintf(os.Stderr, "  SEC_EMAIL    Email for SEC User-Agent header (required for URL fetching)\n")
	}

	flag.Parse()

	// Determine mode: batch (CIK) or single file
	if cik != "" {
		// Batch mode
		if err := runBatch(cik, formType, dateFrom, dateTo, includePaginated, listOnly, email, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Single file mode - require source argument
		if flag.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "Error: source URL or file path required (or use --cik for batch mode)\n\n")
			flag.Usage()
			os.Exit(1)
		}

		source := flag.Arg(0)

		if err := run(source, email, saveOriginal, outputPath, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func run(source, email string, saveOriginal bool, outputPath string, pretty bool) error {
	// Determine if source is URL or file path
	isURL := strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")

	var xmlData []byte
	var urlMeta *edgar.FilingMetadata
	var err error

	// Determine if we should show progress messages (not when outputting JSON to stdout)
	showProgress := saveOriginal || outputPath != ""

	if isURL {
		// Get email for SEC requests (fail fast if not provided)
		if email == "" {
			email, err = edgar.GetSecEmail()
			if err != nil {
				return err
			}
		}

		// Extract metadata from URL
		urlMeta, err = edgar.ExtractMetadataFromURL(source)
		if err != nil && showProgress {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		// Fetch from SEC
		if showProgress {
			fmt.Fprintf(os.Stderr, "Fetching from SEC: %s\n", source)
		}
		xmlData, err = edgar.FetchForm(source, email)
		if err != nil {
			return fmt.Errorf("failed to fetch form: %w", err)
		}
	} else {
		// Read from file
		if showProgress {
			fmt.Fprintf(os.Stderr, "Reading from file: %s\n", source)
		}
		xmlData, err = os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	// Parse the form (auto-detect type)
	if showProgress {
		fmt.Fprintf(os.Stderr, "Parsing form...\n")
	}
	form, err := edgar.ParseAny(bytes.NewReader(xmlData))
	if err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if showProgress {
		fmt.Fprintf(os.Stderr, "Detected form type: %s\n", form.FormType)
	}

	// Extract metadata from parsed form
	formMeta := edgar.ExtractMetadataFromForm(form)

	// Merge metadata from URL and form
	meta := edgar.MergeMetadata(urlMeta, formMeta)

	// Populate source and accession metadata in the form output
	if form.FormType == "4" {
		if f4, ok := form.Data.(*edgar.Form4Output); ok {
			// Set source (URL or file path)
			f4.SetSource(source)
			// Set accession number if available from URL
			if meta.Accession != "" {
				f4.SetFilingMetadata(meta.Accession, "", "")
			}
		}
	} else if form.FormType == "SC 13D" || form.FormType == "SC 13G" {
		// For Schedule 13 filings, populate filer CIK from URL
		if sc13, ok := form.Data.(*edgar.Schedule13Filing); ok {
			// The CIK in the URL is the filer's CIK (the investor), not the issuer
			if meta.CIK != "" {
				sc13.FilerCIK = meta.CIK
			}
		}
	}

	// Prepare save options with default output directory
	saveOpts := edgar.SaveOptions{
		SaveOriginal: saveOriginal,
		OutputDir:    "./output",
	}

	// Determine output path
	if outputPath != "" {
		saveOpts.OutputPath = outputPath
	} else if saveOriginal {
		// If saving original, also save JSON with smart naming
		saveOpts.OutputPath = edgar.GenerateFilename(meta, "json")
	}

	// Save files if requested
	if saveOriginal || outputPath != "" {
		result, err := edgar.SaveFiles(xmlData, form, meta, saveOpts)
		if err != nil {
			return fmt.Errorf("failed to save files: %w", err)
		}

		if showProgress {
			if result.OriginalPath != "" {
				fmt.Fprintf(os.Stderr, "Saved original XML: %s\n", result.OriginalPath)
			}
			if result.OutputPath != "" {
				fmt.Fprintf(os.Stderr, "Saved JSON output: %s\n", result.OutputPath)
			}
		}
	}

	// Check for missing required fields (XBRL only)
	if form.FormType == "XBRL" {
		if snapshot, ok := form.Data.(*edgar.FinancialSnapshot); ok {
			if len(snapshot.MissingRequiredFields) > 0 {
				fmt.Fprintf(os.Stderr, "\n⚠️  Warning: Missing %d required GAAP field(s):\n", len(snapshot.MissingRequiredFields))
				for _, field := range snapshot.MissingRequiredFields {
					fmt.Fprintf(os.Stderr, "  - %s\n", field)
				}
				fmt.Fprintf(os.Stderr, "This may indicate incorrect concept mappings in concept_mappings.json\n\n")
			}
		}
	}

	// If no output file specified, print to stdout
	if outputPath == "" && !saveOriginal {
		// For XBRL, optionally print pretty table
		if form.FormType == "XBRL" && pretty {
			if snapshot, ok := form.Data.(*edgar.FinancialSnapshot); ok {
				printXBRLTable(snapshot)
				return nil
			}
		}

		// Default: JSON output
		jsonData, err := edgar.FormatJSON(form)
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	}

	return nil
}

func printXBRLTable(snapshot *edgar.FinancialSnapshot) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	if snapshot.CompanyName != "" {
		fmt.Printf("  %s\n", snapshot.CompanyName)
	}
	fmt.Println("           Financial Snapshot")
	fmt.Println("═══════════════════════════════════════════════════")
	if snapshot.FiscalYearEnd != "" {
		fmt.Printf("Fiscal Year End: %s", snapshot.FiscalYearEnd)
		if snapshot.FiscalPeriod != "" {
			fmt.Printf(" (%s)", snapshot.FiscalPeriod)
		}
		fmt.Println()
	}
	if snapshot.FormType != "" {
		fmt.Printf("Form Type: %s\n", snapshot.FormType)
	}
	fmt.Println()

	fmt.Printf("%-35s %15s\n", "Metric", "Value")
	fmt.Printf("%-35s %15s\n", "─────────────────────────────────", "──────────────")

	printMetric("Cash & Equivalents", snapshot.Cash)
	printMetric("Total Debt", snapshot.TotalDebt)
	printMetric("Revenue", snapshot.Revenue)
	printMetric("Net Income (Loss)", snapshot.NetIncome)
	printMetric("R&D Expense", snapshot.RDExpense)
	printMetric("G&A Expense", snapshot.GAExpense)

	if snapshot.DilutedShares > 0 {
		millions := snapshot.DilutedShares / 1_000_000
		fmt.Printf("%-35s %12.1fM\n", "Diluted Shares", millions)
	}

	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
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

func runBatch(cik, formType, dateFrom, dateTo string, includePaginated, listOnly bool, email, outputPath string) error {
	// Get email for SEC requests
	if email == "" {
		var err error
		email, err = edgar.GetSecEmail()
		if err != nil {
			return err
		}
	}

	// Setup batch options
	opts := edgar.BatchOptions{
		CIK:              cik,
		FormType:         formType,
		DateFrom:         dateFrom,
		DateTo:           dateTo,
		Email:            email,
		IncludePaginated: includePaginated,
		ListOnly:         listOnly,
	}

	// Fetch and parse batch
	result, err := edgar.FetchAndParseBatch(opts)
	if err != nil {
		return err
	}

	// Print errors if any
	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors encountered:\n")
		for i, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %d. %v\n", i+1, err)
			if i >= 4 { // Show max 5 errors
				fmt.Fprintf(os.Stderr, "  ... and %d more errors\n", len(result.Errors)-5)
				break
			}
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Handle list-only output (just filing metadata)
	var jsonData []byte
	if listOnly {
		// Output filing list as JSON
		jsonData, err = edgar.FormatFilingListJSON(result.FilingList)
		if err != nil {
			return fmt.Errorf("failed to format filing list JSON: %w", err)
		}
	} else {
		// Check for missing required fields in XBRL filings
		if formType == "10-K" || formType == "10-Q" {
			filingsWithMissingFields := 0
			allMissingFields := make(map[string]int) // field name -> count

			for _, filing := range result.Filings {
				if filing.FormType == "XBRL" {
					if snapshot, ok := filing.Data.(*edgar.FinancialSnapshot); ok {
						if len(snapshot.MissingRequiredFields) > 0 {
							filingsWithMissingFields++
							for _, field := range snapshot.MissingRequiredFields {
								allMissingFields[field]++
							}
						}
					}
				}
			}

			if filingsWithMissingFields > 0 {
				fmt.Fprintf(os.Stderr, "\n⚠️  Warning: %d filing(s) missing required GAAP fields:\n", filingsWithMissingFields)
				for field, count := range allMissingFields {
					fmt.Fprintf(os.Stderr, "  - %s (missing in %d filing(s))\n", field, count)
				}
				fmt.Fprintf(os.Stderr, "This may indicate incorrect concept mappings in concept_mappings.json\n\n")
			}
		}

		// Output results as JSON array of parsed forms
		jsonData, err = edgar.FormatJSONBatch(result.Filings)
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
	}

	// Determine output path
	// Default: save to file with smart naming (batch results are often large)
	// Use "-o -" to explicitly output to stdout
	if outputPath == "" {
		// Generate filename: {dateFrom}_{dateTo}_form{formType}_{cik}.json
		// or if no dates: form{formType}_{cik}.json
		var filename string
		if dateFrom != "" && dateTo != "" {
			filename = fmt.Sprintf("%s_%s_form%s_%s.json", dateFrom, dateTo, formType, cik)
		} else if dateFrom != "" {
			filename = fmt.Sprintf("%s_onwards_form%s_%s.json", dateFrom, formType, cik)
		} else if dateTo != "" {
			filename = fmt.Sprintf("until_%s_form%s_%s.json", dateTo, formType, cik)
		} else {
			filename = fmt.Sprintf("form%s_%s.json", formType, cik)
		}
		outputPath = fmt.Sprintf("./output/%s", filename)

		// Ensure output directory exists
		if err := os.MkdirAll("./output", 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Write to file or stdout
	if outputPath == "-" {
		// Explicit stdout request
		fmt.Println(string(jsonData))
	} else {
		if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Saved batch output: %s\n", outputPath)
	}

	return nil
}
