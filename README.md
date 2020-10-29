# csvsplit

## Description

`csvsplit` splits CSV files. Unlike the unix `split` command, `csvsplit` will not split a single CSV row across two files if that row has a newline embedded within a quoted field.

## Usage

1. Clone this repo.
1. `go build`.
1. `./csvsplit -help`.


