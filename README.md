# hex2raw

This is a re-implementation of the hex2raw C program written for the CS:APP Attack Lab. After seeing many students struggling with the incorrect comments (according to hex2raw), I decided to write my own version of it to package with our Attack Lab. This version is written in Go and has the following features:

- Handles no spaces after the start of a comment `/*`. This was the biggest problem with the old version.

- Adds `//` comments.

- Like the old version, `/* */` comments can be nested, so you can comment out a block that already has comments in it. This means every `/*` needs its own `*/`.

- It handles the hex slightly differently and, in my opinion, more intuitively. My program removes all whitespace (spaces, tabs, etc.) on a line and then reads in the hex values. So `0 0` is the same as `00`. In the old version, it reports an error since it is not a complete hex byte. Because a group with an odd number of digits is usually a typo, it prints a warning when this happens. Each line still needs an even number of hex digits, so a byte can't be split across lines.

- Prints out warning if hex string includes `0a` byte. This causes problems in the attack labs because it causes the `Gets` function to terminate early.

- Error messages include the line number, and an unclosed `/*` comment is reported instead of silently eating the rest of the file.

- If there are any errors, nothing is written. A typo will never leave a partial output file or overwrite a good one, and it refuses to write the output over the input file.

## Usage

```
hex2raw [-i input] [-o output] [input]
```

It reads from standard input if no input file is given and writes to standard output if `-o` is not given. Like the old version, it adds a newline to the end of the output.

```
hex2raw exploit.txt > exploit-raw.txt
hex2raw -i exploit.txt -o exploit-raw.txt
./hex2raw < exploit.txt | ./ctarget
```

It exits with status 1 if the input has errors (the old version exits with 255) and 2 if the arguments are wrong.

## Building and testing

`make hex2raw` builds the Go version. `make test` also builds the original C version (`hex2raw-old`) and uses `pytest` to check that both produce the same output.
