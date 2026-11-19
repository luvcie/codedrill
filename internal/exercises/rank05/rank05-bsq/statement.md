# bsq

Write `bsq.c` and any additional `.c` or `.h` files. Read maps from filename arguments in
order, or from standard input when no filenames are given. Replace the largest
obstacle-free square with the map's fill character and print the resulting map.
On equal-sized squares, select the topmost, then leftmost one.

A map starts with its positive row count and three distinct printable ASCII
characters: empty, obstacle, fill. Accept packed headers such as `3.ox` and
space-separated headers such as `3 . o x`. Packed headers use the last three
characters as markers, which also permits digit and space markers. Every header
and row ends with a newline. Rows have a nonzero, identical width and contain
only the empty or obstacle marker. Reject extra rows, missing rows, malformed
headers, repeated markers, and unexpected row characters.

For an invalid or unreadable map, write `map error\n` to stderr and continue.
Separate consecutive filename arguments with one extra newline on stdout,
including when either argument is invalid. A map error does not change the
program's zero exit status. These output details define this practice version.

The school-oriented function list is `malloc`, `calloc`, `realloc`, `free`,
`fopen`, `fclose`, `getline`, `fscanf`, `fputs`, and `fprintf`, with the standard
streams and `errno` available. The reference follows this list; tests check
behavior, not allowed-function compliance. `getline` requires POSIX declarations
(for example `_POSIX_C_SOURCE 200809L` before includes).

All supplied material is independently written for Codedrill. Check your actual
school subject for its precise header and output conventions.
