# life

Write `life.c` (and any additional `.c` or `.h` files) for a finite Conway's Game of Life
board. Run it as `./life width height generations`. Tests supply positive board
dimensions and a nonnegative generation count that fit in an int.

Read a drawing script from standard input until EOF. A pen begins at column 0,
row 0, lifted. `x` toggles drawing, `w`/`s` move up/down, and `a`/`d` move
left/right. Clamp moves at the board edges. While lowered, the pen makes its
current cell alive after each command, including `x` when lowering it. Other
input characters have no effect. All cells initially are dead.

Advance all cells simultaneously each generation. A cell is alive next time
when it has exactly three live neighbors, or when it is already alive and has
two live neighbors. Count all eight adjacent cells. Outside the board is dead;
edges do not wrap.

Write one line per board row, using `O` for live cells and a space for dead
cells. Include a final newline and no heading. This version uses the letter
`O`; the known school variant using digit `0` is not the same output contract.

For school-oriented practice, use only `atoi`, `read`, `putchar`, `malloc`,
`calloc`, `realloc`, and `free`. The reference follows that function list, but
the behavioral tests do not police it. This statement, test script, and
reference are Codedrill originals, not official school material.
