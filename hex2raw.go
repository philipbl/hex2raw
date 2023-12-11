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

	var builder strings.Builder

	scanner := bufio.NewScanner(inputReader)
	for scanner.Scan() {
		line := scanner.Text()

		// Check to see if there is a comment. If there is we will collect the
		// whole thing and remove it.
		if strings.Contains(line, "/*") {
			builder.WriteString(line)

			// Keep looking for the end of the comment
			for !strings.Contains(line, "*/") {
				scanner.Scan()
				line = scanner.Text()
				builder.WriteString(line)
			}
			line = builder.String()
		}

		line = removeMultiLineComments(line)
		line = removeSingleLineComments(line)
		line = strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		line = strings.ReplaceAll(line, " ", "")

		decoded, err := hex.DecodeString(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding hex values: %s\n", err)
			os.Exit(1)
		}

		decoded_string := string(decoded)

		if strings.Contains(decoded_string, "\n") {
			fmt.Fprintln(os.Stderr, "Warning: Output contains newline byte `0a`. This will cause the `Gets` function to return early and not read in your whole input.")
		}

		_, err = fmt.Fprint(outputWriter, decoded_string)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to output: %s\n", err)
			os.Exit(1)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
		os.Exit(1)
	}

	// The old hex2raw always adds a newline to the end of the file
	fmt.Fprint(outputWriter, "\n")
}

func removeMultiLineComments(line string) string {
	re := regexp.MustCompile(`/\*.*?\*/`)
	return re.ReplaceAllString(line, "")
}

func removeSingleLineComments(line string) string {
	re := regexp.MustCompile(`\/\/.*$`)
	return re.ReplaceAllString(line, "")
}
