package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/elliotchance/phpserialize"
	"github.com/fatih/color"
	"os"
	"strconv"
	"strings"
	"time"
)

// Banner to print at startup.
const banner = `
  ____  _       _____                _           
 |  _ \(_)     |  __ \              | |          
 | |_) |_ _ __ | |__) |___  __ _  __| | ___ _ __ 
 |  _ <| | '_ \|  _  // _ \/ _` + "`" + ` |/ _` + "`" + ` |/ _ \ '__|
 | |_) | | | | | | \ \  __/ (_| | (_| |  __/ |   
 |____/|_|_| |_|_|  \_\___|\__,_|\__,_|\___|_|   
                                                
`

// extractTopLevelKeys scans a PHP serialized string (top-level only)
// and extracts the keys in the order they appear.
// This is a simple parser that works for basic serialized arrays.
func extractTopLevelKeys(serialized string) ([]string, error) {
	// Find the opening '{' and the matching closing '}'
	start := strings.Index(serialized, "{")
	end := strings.LastIndex(serialized, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("invalid serialized string")
	}
	content := serialized[start+1 : end]
	keys := []string{}
	i := 0
	for i < len(content) {
		// Look for a key starting with 's:'
		if content[i] == 's' && i+1 < len(content) && content[i+1] == ':' {
			// Find the next colon after the length
			j := i + 2
			for j < len(content) && content[j] != ':' {
				j++
			}
			if j >= len(content) {
				break
			}
			// Extract the length as an integer
			numStr := content[i+2 : j]
			n, err := strconv.Atoi(numStr)
			if err != nil {
				return nil, fmt.Errorf("invalid string length: %v", err)
			}
			// Expect the next character to be a double quote
			if j+1 >= len(content) || content[j+1] != '"' {
				return nil, fmt.Errorf("expected '\"' after length")
			}
			// Extract the key (n characters starting from j+2)
			if j+2+n > len(content) {
				return nil, fmt.Errorf("string length exceeds content")
			}
			key := content[j+2 : j+2+n]
			keys = append(keys, key)
			// Advance i past the key and its closing '";'
			i = j + 2 + n + 2
		} else {
			i++
		}
	}
	return keys, nil
}

// printMapToBuffer writes a formatted version of the map to the provided buffer.
// For the top-level map, we supply the keys in the original order.
func printMapToBuffer(buf *bytes.Buffer, data map[interface{}]interface{}, indent int, keys []interface{}) {
	indentation := strings.Repeat("  ", indent)
	for _, key := range keys {
		// Look up the key in the map. (It may be a string.)
		var value interface{}
		// Try both the key as given and as a string.
		if v, ok := data[key]; ok {
			value = v
		} else if v, ok := data[fmt.Sprintf("%v", key)]; ok {
			value = v
		} else {
			continue
		}
		switch v := value.(type) {
		case map[interface{}]interface{}:
			buf.WriteString(fmt.Sprintf("%s- %v:\n", indentation, key))
			// For nested maps, we don't have the original order, so iterate in arbitrary order.
			nestedKeys := []interface{}{}
			for nk := range v {
				nestedKeys = append(nestedKeys, nk)
			}
			printMapToBuffer(buf, v, indent+1, nestedKeys)
		default:
			buf.WriteString(fmt.Sprintf("%s- %v: %v\n", indentation, key, v))
		}
	}
}

func main() {
	fmt.Print(banner)

	// Define CLI flags
	filePath := flag.String("file", "", "Path to the .bin file")
	flag.StringVar(filePath, "f", "", "Path to the .bin file (shorthand)")
	verbose := flag.Bool("verbose", false, "Show detailed output")
	flag.BoolVar(verbose, "v", false, "Show detailed output (shorthand)")
	outputFile := flag.String("output", "", "Save output to a file")
	flag.StringVar(outputFile, "o", "", "Save output to a file (shorthand)")
	maxSize := flag.Int64("max-size", 10*1024*1024, "Maximum file size in bytes (default: 10MB)")
	flag.Int64Var(maxSize, "m", 10*1024*1024, "Maximum file size in bytes (shorthand, default: 10MB)")
	hexDump := flag.Bool("hex", false, "Display file content as a hex dump")
	flag.BoolVar(hexDump, "x", false, "Display file content as a hex dump (shorthand)")
	showMetadata := flag.Bool("metadata", false, "Show file metadata")
	flag.BoolVar(showMetadata, "d", false, "Show file metadata (shorthand)")
	searchPattern := flag.String("search", "", "Search for a string in the file")
	flag.StringVar(searchPattern, "s", "", "Search for a string in the file (shorthand)")
	compareFile := flag.String("compare", "", "Compare with another .bin file")
	flag.StringVar(compareFile, "c", "", "Compare with another .bin file (shorthand)")
	phpParse := flag.Bool("php", false, "Parse content as PHP serialized data")
	flag.BoolVar(phpParse, "p", false, "Parse content as PHP serialized data (shorthand)")

	flag.Parse()

	// Check if file path is provided
	if *filePath == "" {
		fmt.Println("Error: Please specify a file with --file/-f")
		fmt.Println("Usage: bin-reader --file path/to/file.bin [--verbose/-v] [--output/-o output.txt] [--max-size/-m bytes] [--hex/-x] [--metadata/-d] [--search/-s pattern] [--compare/-c other.bin] [--php/-p]")
		os.Exit(1)
	}

	// Get file info for validation and metadata
	fileInfo, err := os.Stat(*filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Error: File '%s' does not exist\n", *filePath)
		} else {
			fmt.Printf("Error: Unable to access file '%s': %v\n", *filePath, err)
		}
		os.Exit(1)
	}

	// Check file size against maxSize
	fileSize := fileInfo.Size()
	if fileSize > *maxSize {
		fmt.Printf("Error: File '%s' size (%d bytes) exceeds maximum allowed size (%d bytes)\n", *filePath, fileSize, *maxSize)
		os.Exit(1)
	}

	// Read the file content
	fileData, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Printf("Error: Unable to read the file '%s': %v\n", *filePath, err)
		os.Exit(1)
	}

	// Prepare output buffer
	var outputBuffer bytes.Buffer

	// Add metadata if requested
	if *showMetadata {
		outputBuffer.WriteString("File Metadata:\n")
		outputBuffer.WriteString(fmt.Sprintf("  Name: %s\n", fileInfo.Name()))
		outputBuffer.WriteString(fmt.Sprintf("  Size: %d bytes\n", fileSize))
		outputBuffer.WriteString(fmt.Sprintf("  Modified: %s\n", fileInfo.ModTime().Format(time.RFC1123)))
		// Creation time is not directly available on all platforms, using ModTime as fallback
		outputBuffer.WriteString(fmt.Sprintf("  Last Accessed: %s\n", fileInfo.ModTime().Format(time.RFC1123)))
		outputBuffer.WriteString("\n")
	}

	// Only show file content if PHP parsing is not enabled
	if !*phpParse {
		if *hexDump {
			outputBuffer.WriteString("Hex dump of file content:\n")
			outputBuffer.WriteString(hex.Dump(fileData))
		} else {
			outputBuffer.WriteString("File content (raw):\n")
			outputBuffer.WriteString(string(fileData))
			outputBuffer.WriteString("\n")
		}
	}

	// Add verbose details if requested
	if *verbose {
		verboseOutput := fmt.Sprintf("File: %s\nSize: %d bytes\nRaw bytes: %v\n", *filePath, len(fileData), fileData)
		outputBuffer.WriteString(verboseOutput)
	}

	// Search for pattern if provided
	if *searchPattern != "" {
		outputBuffer.WriteString(fmt.Sprintf("Searching for pattern: %q\n", *searchPattern))
		offsets := []int{}
		contentStr := string(fileData)
		for i := 0; ; i++ {
			pos := strings.Index(contentStr[i:], *searchPattern)
			if pos == -1 {
				break
			}
			offsets = append(offsets, i+pos)
			i += pos + len(*searchPattern) - 1
		}
		if len(offsets) > 0 {
			outputBuffer.WriteString(fmt.Sprintf("Found %d matches at offsets: %v\n", len(offsets), offsets))
		} else {
			outputBuffer.WriteString("No matches found\n")
		}
		outputBuffer.WriteString("\n")
	}

	// Compare with another file if requested
	if *compareFile != "" {
		// Validate and read the second file
		compareInfo, err := os.Stat(*compareFile)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Error: Comparison file '%s' does not exist\n", *compareFile)
			} else {
				fmt.Printf("Error: Unable to access comparison file '%s': %v\n", *compareFile, err)
			}
			os.Exit(1)
		}
		if compareInfo.Size() > *maxSize {
			fmt.Printf("Error: Comparison file '%s' size (%d bytes) exceeds maximum allowed size (%d bytes)\n", *compareFile, compareInfo.Size(), *maxSize)
			os.Exit(1)
		}

		compareData, err := os.ReadFile(*compareFile)
		if err != nil {
			fmt.Printf("Error: Unable to read comparison file '%s': %v\n", *compareFile, err)
			os.Exit(1)
		}

		// Compare the files
		outputBuffer.WriteString(fmt.Sprintf("Comparing '%s' with '%s':\n", *filePath, *compareFile))
		if bytes.Equal(fileData, compareData) {
			outputBuffer.WriteString("Files are identical\n")
		} else {
			outputBuffer.WriteString("Files differ\n")
			minLen := len(fileData)
			if len(compareData) < minLen {
				minLen = len(compareData)
			}
			differences := []string{}
			for i := 0; i < minLen; i++ {
				if fileData[i] != compareData[i] {
					differences = append(differences, fmt.Sprintf("Offset %d: %x vs %x", i, fileData[i], compareData[i]))
				}
			}
			if len(fileData) != len(compareData) {
				differences = append(differences, fmt.Sprintf("Length mismatch: %d vs %d bytes", len(fileData), len(compareData)))
			}
			outputBuffer.WriteString(fmt.Sprintf("Differences found: %d\n", len(differences)))
			for _, diff := range differences {
				outputBuffer.WriteString(fmt.Sprintf("  %s\n", diff))
			}
		}
		outputBuffer.WriteString("\n")
	}

	// If PHP parse flag is set, parse and pretty-print the PHP serialized data.
	if *phpParse {
		outputBuffer.WriteString("PHP Serialized Data (parsed):\n")
		// Extract the top-level key order from the serialized string.
		topKeys, err := extractTopLevelKeys(string(fileData))
		if err != nil {
			outputBuffer.WriteString(fmt.Sprintf("Error extracting key order: %v\n", err))
		}

		var parsedData map[interface{}]interface{}
		err = phpserialize.Unmarshal(fileData, &parsedData)
		if err != nil {
			outputBuffer.WriteString(fmt.Sprintf("Failed to parse as PHP serialized data: %v\n", err))
		} else {
			// Convert extracted keys (if any) to a slice of interface{}
			keys := []interface{}{}
			for _, k := range topKeys {
				keys = append(keys, k)
			}
			// Use a buffer to collect pretty output.
			var buf bytes.Buffer
			printMapToBuffer(&buf, parsedData, 0, keys)
			outputBuffer.WriteString(buf.String())
		}
		outputBuffer.WriteString("\n")
	}

	output := outputBuffer.String()

	// Colorize output
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	coloredOutput := output
	coloredOutput = strings.ReplaceAll(coloredOutput, "File content (raw):", green("File content (raw):"))
	coloredOutput = strings.ReplaceAll(coloredOutput, "Hex dump of file content:", green("Hex dump of file content:"))
	coloredOutput = strings.ReplaceAll(coloredOutput, "File Metadata:", yellow("File Metadata:"))
	coloredOutput = strings.ReplaceAll(coloredOutput, "Searching for pattern:", yellow("Searching for pattern:"))
	coloredOutput = strings.ReplaceAll(coloredOutput, "Comparing", yellow("Comparing"))
	coloredOutput = strings.ReplaceAll(coloredOutput, "Error:", red("Error:"))
	fmt.Print(coloredOutput)

	// Save to file if specified
	if *outputFile != "" {
		err = os.WriteFile(*outputFile, []byte(output), 0644)
		if err != nil {
			fmt.Printf("Error: Unable to write to output file '%s': %v\n", *outputFile, err)
			os.Exit(1)
		}
		fmt.Printf("Output saved to '%s'\n", *outputFile)
	}
}
