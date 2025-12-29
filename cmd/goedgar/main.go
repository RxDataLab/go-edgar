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

		// Batch mode
		cik              string
		formType         string
		dateFrom         string
		dateTo           string
		includePaginated bool
	)

	flag.BoolVar(&saveOriginal, "save-original", false, "Save the original XML file")
	flag.BoolVar(&saveOriginal, "s", false, "Save the original XML file (shorthand)")
	flag.StringVar(&outputPath, "output", "", "Output JSON file path (default: stdout)")
	flag.StringVar(&outputPath, "o", "", "Output JSON file path (shorthand)")
	flag.StringVar(&email, "email", "", "Email for SEC User-Agent header (or use SEC_EMAIL env var)")
	flag.StringVar(&email, "e", "", "Email for SEC User-Agent (shorthand)")

	// Batch mode flags
	flag.StringVar(&cik, "cik", "", "CIK to fetch filings for (batch mode)")
	flag.StringVar(&formType, "form", "4", "Form type to fetch (default: 4)")
	flag.StringVar(&dateFrom, "from", "", "Start date for filtering (YYYY-MM-DD)")
	flag.StringVar(&dateTo, "to", "", "End date for filtering (YYYY-MM-DD)")
	flag.BoolVar(&includePaginated, "all", false, "Include all paginated filings (can be slow)")

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
		fmt.Fprintf(os.Stderr, "  # Batch mode\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 0000078003 --form 4 --from 2025-01-01 --to 2025-06-30\n")
		fmt.Fprintf(os.Stderr, "  goedgar --cik 1631574 --form 4  # All recent Form 4s\n\n")
		fmt.Fprintf(os.Stderr, "Environment:\n")
		fmt.Fprintf(os.Stderr, "  SEC_EMAIL    Email for SEC User-Agent header (required for URL fetching)\n")
	}

	flag.Parse()

	// Determine mode: batch (CIK) or single file
	if cik != "" {
		// Batch mode
		if err := runBatch(cik, formType, dateFrom, dateTo, includePaginated, email, outputPath); err != nil {
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

		if err := run(source, email, saveOriginal, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func run(source, email string, saveOriginal bool, outputPath string) error {
	// Determine if source is URL or file path
	isURL := strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")

	var xmlData []byte
	var urlMeta *edgar.FilingMetadata
	var err error

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
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		// Fetch from SEC
		fmt.Fprintf(os.Stderr, "Fetching from SEC: %s\n", source)
		xmlData, err = edgar.FetchForm(source, email)
		if err != nil {
			return fmt.Errorf("failed to fetch form: %w", err)
		}
	} else {
		// Read from file
		fmt.Fprintf(os.Stderr, "Reading from file: %s\n", source)
		xmlData, err = os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	// Parse the form (auto-detect type)
	fmt.Fprintf(os.Stderr, "Parsing form...\n")
	form, err := edgar.ParseAny(bytes.NewReader(xmlData))
	if err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Detected form type: %s\n", form.FormType)

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

		if result.OriginalPath != "" {
			fmt.Fprintf(os.Stderr, "Saved original XML: %s\n", result.OriginalPath)
		}
		if result.OutputPath != "" {
			fmt.Fprintf(os.Stderr, "Saved JSON output: %s\n", result.OutputPath)
		}
	}

	// If no output file specified, print JSON to stdout
	if outputPath == "" && !saveOriginal {
		jsonData, err := edgar.FormatJSON(form)
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	}

	return nil
}

func runBatch(cik, formType, dateFrom, dateTo string, includePaginated bool, email, outputPath string) error {
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

	// Output results as JSON array
	jsonData, err := edgar.FormatJSONBatch(result.Filings)
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
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
