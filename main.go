package main

import (
	"flag"
	"fmt"
	"os"
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

	flag.Parse()

	// Check if file path is provided
	if *filePath == "" {
		fmt.Println("Error: Please specify a file with --file/-f")
		fmt.Println("Usage: bin-reader --file path/to/file.bin")
		os.Exit(1)
	}

	// Read the file content
	fileData, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Printf("Error: Unable to read the file '%s': %v\n", *filePath, err)
		os.Exit(1)
	}

	// For now, just display the raw data
	fmt.Println("File content (raw):")
	fmt.Println(string(fileData))
}
