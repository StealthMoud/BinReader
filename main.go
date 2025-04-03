package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const banner = `
  ____  _       _____                _           
 |  _ \(_)     |  __ \              | |          
 | |_) |_ _ __ | |__) |___  __ _  __| | ___ _ __ 
 |  _ <| | '_ \|  _  // _ \/ _` + "`" + ` |/ _` + "`" + ` |/ _ \ '__|
 | |_) | | | | | | \ \  __/ (_| | (_| |  __/ |   
 |____/|_|_| |_|_|  \_\___|\__,_|\__,_|\___|_|   
                                                
`

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

	flag.Parse()

	// Check if file path is provided
	if *filePath == "" {
		fmt.Println("Error: Please specify a file with --file/-f")
		fmt.Println("Usage: bin-reader --file path/to/file.bin [--verbose/-v] [--output/-o output.txt] [--max-size/-m bytes] [--hex/-x] [--metadata/-d] [--search/-s pattern] [--compare/-c other.bin]")
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

	// Prepare output
	var outputBuffer bytes.Buffer

	// Add metadata if requested
	if *showMetadata {
		outputBuffer.WriteString("File Metadata:\n")
		outputBuffer.WriteString(fmt.Sprintf("  Name: %s\n", fileInfo.Name()))
		outputBuffer.WriteString(fmt.Sprintf("  Size: %d bytes\n", fileSize))
		outputBuffer.WriteString(fmt.Sprintf("  Modified: %s\n", fileInfo.ModTime().Format(time.RFC1123)))
		// Creation time is not directly available on all platforms, use ModTime as a fallback
		outputBuffer.WriteString(fmt.Sprintf("  Last Accessed: %s\n", fileInfo.ModTime().Format(time.RFC1123)))
		outputBuffer.WriteString("\n")
	}

	// Add file content (hex or raw)
	if *hexDump {
		outputBuffer.WriteString("Hex dump of file content:\n")
		outputBuffer.WriteString(hex.Dump(fileData))
	} else {
		outputBuffer.WriteString("File content (raw):\n")
		outputBuffer.WriteString(string(fileData))
		outputBuffer.WriteString("\n")
	}

	// Add verbose details if requested
	if *verbose {
		verboseOutput := fmt.Sprintf("File: %s\nSize: %d bytes\nRaw bytes: %v\n", *filePath, len(fileData), fileData)
		outputBuffer.WriteString(verboseOutput)
	}

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

	output := outputBuffer.String()

	// Display output
	fmt.Print(output)

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
