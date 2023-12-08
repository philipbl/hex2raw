package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

func main() {
	var inputFile, outputFile string

	flag.StringVar(&inputFile, "i", "", "Input file path")
	flag.StringVar(&outputFile, "o", "", "Output file path")
	flag.Parse()

	var inputReader io.Reader

	if inputFile != "" {
		file, err := os.Open(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %s\n", err)
			os.Exit(1)
		}
		defer file.Close()
		inputReader = file
	} else {
		inputReader = os.Stdin
	}

	var outputWriter io.Writer

	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %s\n", err)
			os.Exit(1)
		}
		defer file.Close()
		outputWriter = file
	} else {
		outputWriter = os.Stdout
	}

	scanner := bufio.NewScanner(inputReader)
	for scanner.Scan() {
		line := scanner.Text()
		line = removeComments(line)
		line = strings.ReplaceAll(line, " ", "")

		if len(line) == 0 {
			continue
		}

		decoded, err := hex.DecodeString(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding hex values: %s\n", err)
			os.Exit(1)
		}

		_, err = fmt.Fprint(outputWriter, string(decoded))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to output: %s\n", err)
			os.Exit(1)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
		os.Exit(1)
	}
}

func removeComments(line string) string {
	re := regexp.MustCompile(`/\*.*?\*/`)
	return re.ReplaceAllString(line, "")
}
