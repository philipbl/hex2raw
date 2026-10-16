# Changelog

## 1.3.0 (unreleased)

Every error and warning goes to stderr, so students see it even when the output is redirected to a file or piped into `ctarget`. If there are any errors, nothing is written to the output, and the last message says so.

### Fixed

- An unclosed `/*` comment made hex2raw hang forever. It is now an error that gives the line where the comment starts.
- A `/*` inside a `//` comment, like `48 c7 // see /* note`, silently dropped the hex on the following lines.
- Tabs between bytes (`48<tab>c7`) were rejected.
- A comment that ends and a new one that starts on the same line (`/* a */ 48 /* b`) was rejected.
- Nested comments, which the original hex2raw allows, were rejected.
- Lines longer than 64 KiB were rejected.
- An error partway through the input left the lines before it in the output.
- `-o` emptied an existing output file before reading the input, so a typo destroyed the last good output.
- `-i exploit.txt -o exploit.txt`, or `-o exploit.txt < exploit.txt`, erased the input file.
- `hex2raw exploit.txt` ignored the file and waited for input from the keyboard, and `hex2raw exploit.txt -o out.txt` also ignored `-o`.
- Part of the `-h` help text went to stdout instead of stderr.
- v1.2.1 reported its version as 1.2.0.

### Changed

- A group with an odd number of hex digits, like the `0` in `00 0 00`, is still joined with the digits next to it, but now prints a warning. It is usually a typo, and two typos on one line would otherwise silently shift every byte after them.
- Comments nest, so every `/*` needs its own `*/`. A `/*` in the text of a comment (`/* see lib/*.c */`) now leaves the comment open, which is reported as an error.
- Errors give the line number and the text that caused them, such as `Error: line 3: invalid hex value "0x48". Leave off the 0x prefix.`
- Up to 10 errors are reported at once instead of only the first one.
- Nothing is written if there are any errors, and the last message is `hex2raw: no output was written because of the error above.`
- The `0a` warning is printed once and lists the lines with a `0a` byte, instead of once per line.
- Bad command-line arguments exit with status 2. Errors in the input still exit with status 1.
- Inputs larger than 16 MiB are rejected as probably being the wrong file.

### Added

- The input file can be given as an argument: `hex2raw exploit.txt`.
- A warning if the input has no hex bytes, for example because it is all commented out.
- When reading from the keyboard, a message explaining how to finish.
- A byte order mark at the start of the file, which some Windows editors add, is ignored.
- Any Unicode whitespace, such as a non-breaking space copied from a PDF, separates hex digits.

### Development

- Added regression tests for each fix. The random tests are now seeded and include tabs, newlines, and uppercase hex, and every test has a timeout so a hang fails instead of stalling.
- The original C version used by the tests (`hex2raw-old`) always printed its usage message where `char` is unsigned, such as Linux on ARM.
- The release workflow builds only the Go binary.

## 1.2.1 (2024-12-17)

### Fixed

- When more than one line had a `/* */` comment, the bytes from the earlier lines were repeated in the output.

## 1.2.0 (2023-12-14)

### Added

- `--version` flag and a usage message with examples.

## 1.1.0 (2023-12-10)

### Added

- A warning when the output contains a `0a` byte, which makes `Gets` stop reading early.

## 1.0.0 (2023-12-10)

First release. A Go re-implementation of the CS:APP Attack Lab hex2raw that allows comments without a space after `/*`, adds `//` comments, and ignores whitespace between hex digits.
