# hex2raw

This is a re-implementation of the hex2raw C program written for the CS:APP Attack Lab. After seeing many students struggling with the incorrect comments (according to hex2raw), I decided to write my own version of it to package with our Attack Lab. This version is written in Go and has the following features:

- Handles no spaces after the start of a comment `/*`. This was the biggest problem with the old version.

- Adds `//` comments.

- It handles the hex slightly different and, in my opinion, more intuitively. My program removes all whitespace and then reads in the hex values. So `0 0` is the same as `00`. In the old version, it reports an error since it is not a complete hex byte.
