package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const prefixPattern = "OlPrEfIx"

type SymbolCollection struct {
	FileSymbols []BinaryFileSymbols `json:"file_symbols"`
}

type BinaryFileSymbols struct {
	Name    string   `json:"name"`
	Hash    string   `json:"hash"`
	Symbols []string `json:"symbols"`
}

type Flags struct {
	help       bool
	clobber    bool
	outputPath string
}

var flags = Flags{}

func init() {
	flag.BoolVar(&flags.help, "help", false, "Show usage")
	flag.BoolVar(&flags.clobber, "clobber", false, "Overwrite existing symbol collections")
	flag.StringVar(&flags.outputPath, "output", "", "Process symbols into a single file")
	flag.Parse()
}

func main() {
	// Parse command-line flags
	// add a --help flag to show the usage "Usage ./evrfindsymbols [FILE]..."

	if flags.help || len(os.Args) == 1 {
		fmt.Println("Usage: ./evrfindsymbols [FILE]...")
		flag.PrintDefaults()
		return
	}
	var jsonPath string
	var symbolCollection SymbolCollection
	// If using single file then open/create the json file
	if flags.outputPath != "" {
		jsonPath = flags.outputPath
		// If the file exists, read it in
		if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
			// Read the file
			jsonFile, err := os.Open(jsonPath)
			if err != nil {
				fmt.Println("Error opening JSON file:", err)
				return
			}
			defer jsonFile.Close()
			decoder := json.NewDecoder(jsonFile)

			err = decoder.Decode(&symbolCollection)
			if err != nil {
				fmt.Println("Error reading JSON file:", err)
				return
			}
		}
		if symbolCollection.FileSymbols == nil {
			symbolCollection.FileSymbols = make([]BinaryFileSymbols, 0)
		}
	}

	// Process each binary file
	for _, binaryPath := range flag.Args() {
		// Create a json file to store the symbols that is named after the binary file but has .symbols.json on the end

		if flags.outputPath == "" {
			jsonPath = binaryPath + ".symbols.json"
			if _, err := os.Stat(jsonPath); !os.IsNotExist(err) && !flags.clobber {
				fmt.Println("Skipping existing file:", jsonPath)
				continue
			}

			symbolCollection = SymbolCollection{
				FileSymbols: make([]BinaryFileSymbols, 0),
			}
		}

		symbols, hash, err := processFile(binaryPath)
		if err != nil {
			fmt.Println("Error processing file:", err)
			return
		}
		// Write the JSON file
		jsonFile, err := os.Create(jsonPath)
		if err != nil {
			fmt.Println("Error creating JSON file:", err)
			return
		}
		defer jsonFile.Close()

		// just the filename of the  of the binary path

		binaryFileSymbols := BinaryFileSymbols{
			Name:    filepath.Base(binaryPath),
			Hash:    hash,
			Symbols: symbols,
		}
		// Check if this hash already exists in the collection, replace it
		for i, fileSymbols := range symbolCollection.FileSymbols {
			if fileSymbols.Hash == hash {
				// Remove it from the sliceq
				symbolCollection.FileSymbols = append(symbolCollection.FileSymbols[:i], symbolCollection.FileSymbols[i+1:]...)
			}
		}
		symbolCollection.FileSymbols = append(symbolCollection.FileSymbols, binaryFileSymbols)

		// Write the JSON data to the file
		encoder := json.NewEncoder(jsonFile)
		if err != nil {
			fmt.Println("Error writing JSON data:", err)
			return
		}
		encoder.SetIndent("", "  ")
		encoder.Encode(symbolCollection)

		// Print a count of how many symbols
		fmt.Printf("%s: %d symbols\n", binaryPath, len(symbols))

	}
}

func processFile(binaryPath string) (symbols []string, hash string, err error) {
	symbolScanner := NewSymbolScanner()

	// Open the binary file
	file, err := os.Open(binaryPath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	defer file.Close()

	hasher := sha256.New()

	symbolmap := make(map[string]bool)
	// Read the binary file 100MB at a time as to not load the entire file into memory
	const chunkSize = 100 * 1024 * 1024
	buffer := make([]byte, chunkSize)
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil {
			break
		}
		// If the last byte isn't null, we need to read more to ensure we don't miss any symbols
		if buffer[bytesRead-1] != 0 {
			// Read until we find a null character
			for buffer[bytesRead-1] != 0 {
				n, err := file.Read(buffer[bytesRead : bytesRead+1])
				if err != nil {
					break
				}
				bytesRead += n
			}
		}
		// Write the bytes to the hasher
		hasher.Write(buffer[:bytesRead])

		// Scan the bytes for symbols
		symbolmap, err = symbolScanner.ScanBytes(buffer[:bytesRead], symbolmap)
		if err != nil {
			fmt.Println("Error scanning file:", err)
			return nil, "", err
		}
	}

	// Convert the map to a slice
	symbols = make([]string, 0, len(symbolmap))
	for k := range symbolmap {
		symbols = append(symbols, k)
	}

	return symbols, fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

type SymbolScanner struct{}

func NewSymbolScanner() *SymbolScanner {
	return &SymbolScanner{}
}

func (s *SymbolScanner) ScanBytes(data []byte, symbols map[string]bool) (map[string]bool, error) {
	if len(data) == 0 {
		return symbols, nil
	}
	if symbols == nil {
		symbols = make(map[string]bool)
	}

	// Search the binary for the prefix pattern
	prefix := []byte(prefixPattern)

	bytes.Split(data, prefix)
	for _, b := range bytes.Split(data, prefix) {
		// Search for the first null byte
		for i, c := range b {
			if c == 0 {
				// If the byte is null, we have found the end of the symbol
				symbols[string(b[:i])] = true
				break
			}
		}
	}

	return symbols, nil
}
