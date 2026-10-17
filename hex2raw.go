package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

const appVersion = "1.3.0"

// Attack Lab inputs are a few hundred bytes. Anything this big is the wrong file
// (or something like /dev/zero), so refuse it instead of reading until we run
// out of memory.
const maxInputSize = 16 << 20

// Stop after this many errors so the wrong file doesn't print pages of errors.
const maxErrors = 10

func main() {
	var inputFile, outputFile string
	var version bool

	flag.StringVar(&inputFile, "i", "", "Input file path. Default is stdin.")
	flag.StringVar(&outputFile, "o", "", "Output file path. Default is stdout.")
	flag.BoolVar(&version, "version", false, "Print the version of the program.")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage: %s [-i input] [-o output] [input]\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(out, "\nExamples:")
		fmt.Fprintln(out, "  hex2raw -i input.txt -o output.txt")
		fmt.Fprintln(out, "  hex2raw input.txt > output.txt")
		fmt.Fprintln(out, "  cat input.txt | hex2raw > output.txt")
		fmt.Fprintln(out, "  hex2raw --version")
	}
	positional := parseFlags(os.Args[1:])

	if version {
		fmt.Println("hex2raw version", appVersion)
		return
	}

	switch {
	case len(positional) > 1:
		usageError("expected at most one input file, got %d: %s", len(positional), strings.Join(positional, " "))
	case len(positional) == 1 && inputFile != "":
		usageError("give the input file with -i or as an argument, not both")
	case len(positional) == 1:
		inputFile = positional[0]
	}

	input := os.Stdin
	if inputFile != "" {
		file, err := os.Open(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %s\n", err)
			os.Exit(1)
		}
		defer file.Close()
		input = file
	} else if isTerminal(os.Stdin) {
		eof := "Ctrl-D"
		if runtime.GOOS == "windows" {
			eof = "Ctrl-Z then Enter"
		}
		fmt.Fprintf(os.Stderr, "Reading hex from the keyboard. Press %s when done (or run: hex2raw input.txt).\n", eof)
	}

	// Writing the output would replace the student's hex file with raw bytes
	if outputFile != "" && sameFile(input, outputFile) {
		fmt.Fprintf(os.Stderr, "Error: the output file %q is the input file. This would erase your input.\n", outputFile)
		os.Exit(1)
	}

	data, err := io.ReadAll(io.LimitReader(input, maxInputSize+1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
		os.Exit(1)
	}
	if len(data) > maxInputSize {
		fmt.Fprintf(os.Stderr, "Error: input is larger than %d MiB. Is it the right file?\n", maxInputSize>>20)
		os.Exit(1)
	}

	output, ok := convert(data, os.Stderr)
	if !ok {
		os.Exit(1)
	}

	if err := writeOutput(outputFile, output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %s\n", err)
		os.Exit(1)
	}
}

// parseFlags parses flags that appear anywhere on the command line and returns
// the other arguments. The flag package stops at the first non-flag argument,
// so on its own "hex2raw input.txt -o output.txt" would silently ignore -o.
func parseFlags(args []string) []string {
	var positional []string
	for {
		flag.CommandLine.Parse(args)
		args = flag.Args()
		if len(args) == 0 {
			return positional
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
}

func usageError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	flag.Usage()
	os.Exit(2)
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func sameFile(input *os.File, path string) bool {
	inInfo, err := input.Stat()
	if err != nil || !inInfo.Mode().IsRegular() {
		return false
	}
	outInfo, err := os.Stat(path)
	return err == nil && os.SameFile(inInfo, outInfo)
}

// writeOutput is only called once the whole input has been converted, so a
// typo never leaves a truncated or partially written output file behind.
func writeOutput(path string, data []byte) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// converter holds the state for turning hex text into bytes, one character at
// a time. Hex digits are collected per line: whitespace between them is
// ignored (so "0 0" is the same as "00"), but each line must have an even
// number of digits.
type converter struct {
	msgs io.Writer
	line int

	commentDepth int // how many /* comments we are inside
	commentLine  int // line where the outermost open /* comment started
	lineComment  bool

	token     []rune   // the whitespace-separated group being read
	digits    []byte   // all hex digits on the current line
	oddTokens []string // groups on the current line with an odd number of digits
	lineBad   bool     // an error was already reported for the current line

	output       []byte
	newlineLines []int // lines that produce a 0a byte
	errors       int
}

// convert turns hex text into raw bytes, writing any warnings and errors to
// msgs. If ok is false there were errors and no output should be written.
func convert(data []byte, msgs io.Writer) (output []byte, ok bool) {
	// Some editors (e.g., Notepad) start the file with a byte order mark
	text := []rune(strings.TrimPrefix(string(data), "\uFEFF"))
	c := &converter{msgs: msgs, line: 1}

	for i := 0; i < len(text) && c.errors < maxErrors; i++ {
		r := text[i]
		var next rune
		if i+1 < len(text) {
			next = text[i+1]
		}

		switch {
		case r == '\n':
			c.endLine()
			c.lineComment = false
			c.line++
		case c.lineComment:
			// Ignore everything to the end of the line, including /* and */
		case c.commentDepth > 0:
			// Comments nest, like in the original hex2raw, so a block that
			// already contains comments can be commented out
			if r == '/' && next == '*' {
				c.commentDepth++
				i++
			} else if r == '*' && next == '/' {
				c.commentDepth--
				i++
			}
		case r == '/' && next == '*':
			c.endToken()
			c.commentDepth = 1
			c.commentLine = c.line
			i++
		case r == '/' && next == '/':
			c.endToken()
			c.lineComment = true
			i++
		case r == '*' && next == '/':
			c.endToken()
			c.errorf(c.line, "found */ without a matching /*")
			i++
		case unicode.IsSpace(r):
			c.endToken()
		default:
			c.token = append(c.token, r)
		}
	}

	// Nothing is written when there are errors, so say so. Otherwise a student
	// piping into ctarget only sees ctarget fail on empty input.
	if c.errors >= maxErrors {
		fmt.Fprintf(msgs, "hex2raw: stopped after %d errors. No output was written.\n", c.errors)
		return nil, false
	}

	c.endLine()
	if c.commentDepth > 0 {
		c.errorf(c.commentLine, "this /* comment is never closed with */ (comments nest, so each /* needs its own */)")
	}
	switch c.errors {
	case 0:
	case 1:
		fmt.Fprintln(msgs, "hex2raw: no output was written because of the error above.")
		return nil, false
	default:
		fmt.Fprintf(msgs, "hex2raw: no output was written because of the %d errors above.\n", c.errors)
		return nil, false
	}

	if len(c.output) == 0 {
		fmt.Fprintln(msgs, "Warning: no hex bytes were found in the input (is it all commented out?), so the output is just a newline.")
	}
	if len(c.newlineLines) > 0 {
		fmt.Fprintf(msgs, "Warning: Output contains newline byte `0a` (%s). This will cause the `Gets` function to return early and not read in your whole input.\n", formatLines(c.newlineLines))
	}

	// The old hex2raw always adds a newline to the end of the file
	return append(c.output, '\n'), true
}

func (c *converter) endToken() {
	if len(c.token) == 0 {
		return
	}
	token := string(c.token)
	c.token = c.token[:0]

	for _, r := range token {
		if !isHexDigit(r) {
			hint := "Use only hex digits (0-9, a-f); comments go inside /* */ or after //."
			if strings.HasPrefix(strings.ToLower(token), "0x") {
				hint = "Leave off the 0x prefix."
			}
			c.errorf(c.line, "invalid hex value %s. %s", quote(token), hint)
			c.lineBad = true
			return
		}
	}

	if len(token)%2 == 1 {
		c.oddTokens = append(c.oddTokens, quote(token))
	}
	c.digits = append(c.digits, token...)
}

func (c *converter) endLine() {
	c.endToken()

	switch {
	case c.lineBad:
		// Already reported
	case len(c.digits)%2 == 1:
		c.errorf(c.line, "odd number of hex digits; check %s. Each byte needs two hex digits.", strings.Join(c.oddTokens, ", "))
	default:
		if len(c.oddTokens) > 0 {
			c.warnf(c.line, "groups with an odd number of hex digits were joined with the digits next to them: %s. Check that no digit is missing.", strings.Join(c.oddTokens, ", "))
		}

		// Can't fail: digits only holds an even number of hex digits
		decoded, _ := hex.DecodeString(string(c.digits))
		for _, b := range decoded {
			if b == '\n' {
				c.newlineLines = append(c.newlineLines, c.line)
				break
			}
		}
		c.output = append(c.output, decoded...)
	}

	c.digits = c.digits[:0]
	c.oddTokens = c.oddTokens[:0]
	c.lineBad = false
}

func (c *converter) errorf(line int, format string, args ...any) {
	c.errors++
	fmt.Fprintf(c.msgs, "Error: line %d: %s\n", line, fmt.Sprintf(format, args...))
}

func (c *converter) warnf(line int, format string, args ...any) {
	fmt.Fprintf(c.msgs, "Warning: line %d: %s\n", line, fmt.Sprintf(format, args...))
}

func isHexDigit(r rune) bool {
	return '0' <= r && r <= '9' || 'a' <= r && r <= 'f' || 'A' <= r && r <= 'F'
}

// quote formats a group of characters for a message, shortening it so the
// wrong file (e.g., a binary) doesn't print one enormous line.
func quote(token string) string {
	if runes := []rune(token); len(runes) > 20 {
		token = string(runes[:20]) + "..."
	}
	return strconv.Quote(token)
}

func formatLines(lines []int) string {
	numbers := make([]string, len(lines))
	for i, line := range lines {
		numbers[i] = strconv.Itoa(line)
	}
	if len(lines) == 1 {
		return "line " + numbers[0]
	}
	return "lines " + strings.Join(numbers, ", ")
}
