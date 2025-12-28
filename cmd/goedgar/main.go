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
		saveOriginal bool
		outputPath   string
		email        string
	)

	flag.BoolVar(&saveOriginal, "save-original", false, "Save the original XML file")
	flag.BoolVar(&saveOriginal, "s", false, "Save the original XML file (shorthand)")
	flag.StringVar(&outputPath, "output", "", "Output JSON file path (default: stdout)")
	flag.StringVar(&outputPath, "o", "", "Output JSON file path (shorthand)")
	flag.StringVar(&email, "email", "", "Email for SEC User-Agent header (or use SEC_EMAIL env var)")
	flag.StringVar(&email, "e", "", "Email for SEC User-Agent (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: goedgar [options] <source>\n\n")
		fmt.Fprintf(os.Stderr, "Parse SEC forms from URL or file path.\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  <source>    URL or file path to SEC form XML\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  goedgar https://www.sec.gov/Archives/edgar/data/.../ownership.xml\n")
		fmt.Fprintf(os.Stderr, "  goedgar ./ownership.xml\n")
		fmt.Fprintf(os.Stderr, "  goedgar -s -o output.json https://www.sec.gov/.../ownership.xml\n")
		fmt.Fprintf(os.Stderr, "\nEnvironment:\n")
		fmt.Fprintf(os.Stderr, "  SEC_EMAIL    Email for SEC User-Agent header (required for URL fetching)\n")
	}

	flag.Parse()

	// Check for source argument
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: source URL or file path required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	source := flag.Arg(0)

	// Run the main logic
	if err := run(source, email, saveOriginal, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
