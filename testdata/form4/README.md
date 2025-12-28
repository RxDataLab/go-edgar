# Form 4 Test Cases

This directory contains test cases for Form 4 parsing. Each test case is a subdirectory with two files:

## Directory Structure

```
testdata/form4/
├── README.md           # This file
├── <test_case_name>/
│   ├── input.xml       # The Form 4 XML file to parse
│   └── expected.json   # Expected parsed output with metadata
```

## Adding a New Test Case

1. **Download a Form 4 XML file** from SEC EDGAR
2. **Create a new directory** under `testdata/form4/` with a descriptive name
3. **Save the XML** as `input.xml` in that directory
4. **Create `expected.json`** with the structure:

```json
{
  "metadata": {
    "source_url": "https://www.sec.gov/...",
    "notes": "Description of what this test case validates (e.g., 'Edge case: Multiple footnotes with 10b5-1 plan')"
  },
  "expected": {
    // Full parsed Form4 struct as JSON
  }
}
```

5. **Generate the expected output** by parsing the XML and outputting JSON:

```bash
# Quick way to generate the expected structure:
go run -c '
package main
import (
    "encoding/json"
    "fmt"
    "os"
    "github.com/RxDataLab/go-edgar"
)

type TC struct {
    Metadata struct {
        SourceURL string `json:"source_url"`
        Notes     string `json:"notes"`
    } `json:"metadata"`
    Expected *edgar.Form4 `json:"expected"`
}

func main() {
    data, _ := os.ReadFile("input.xml")
    f4, _ := edgar.Parse(data)
    tc := TC{Expected: f4}
    tc.Metadata.SourceURL = "YOUR_URL"
    tc.Metadata.Notes = "YOUR_NOTES"
    json, _ := json.MarshalIndent(tc, "", "  ")
    fmt.Println(string(json))
}
' > expected.json
```

Or manually/with LLM assistance, fill out the expected Form4 struct.

6. **Run tests** - the test will automatically discover and run your new test case:

```bash
go test -v -run TestForm4Parser
```

## Test Case Types

- **Basic cases**: Standard Form 4s with common transaction types
- **Edge cases**: Forms with unusual features (multiple footnotes, 10b5-1 plans, etc.)
- **Regression cases**: Forms that previously caused parsing issues

## Notes

- The test uses deep equality comparison via `go-cmp`
- Metadata fields (`source_url`, `notes`) are for documentation only
- The test automatically discovers all subdirectories in this folder
- Each test case runs independently as a subtest
