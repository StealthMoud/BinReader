package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
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

	flag.Parse()

	// Check if file path is provided
	if *filePath == "" {
		fmt.Println("Error: Please specify a file with --file/-f")
		fmt.Println("Usage: bin-reader --file path/to/file.bin")
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
