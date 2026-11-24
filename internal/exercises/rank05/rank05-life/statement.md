# life

- **Assignment name:** `life`
- **Expected files:** `*.c`, `*.h`
- **Allowed functions:** `atoi`, `read`, `putchar`, `malloc`, `calloc`, `realloc`, `free`

You must write a program that simulates a game of life.

The prototype of this program is:

```sh
./life width height iterations
```

`width` and `height` are the dimensions of the board, and `iterations` is the number
of iterations of the game of life.

The initial configuration of the board is drawn by a sequence of commands on the
standard input. Imagine a pen starting in the top left corner of the board. Each
command is one of these characters:

- `w`, `a`, `s`, `d` move the pen up, left, down and right.
- `x` lifts or lowers the pen, in order to start or stop drawing.

Once end of file is reached, your program must simulate a game of life on this
board and print the result on stdout (each living cell is represented by the
character `'O'` and each dead cell by a space).

Each cell outside of the array is considered dead.

### Examples

`cat -e` shows line ends as `$`.

```text

$> echo 'sdxddssaaww' | ./life 5 5 0 | cat -e
     $
 OOO $
 O O $
 OOO $
     $

$> echo 'sdxssdswdxddddsxaadwxwdxwaa' | ./life 10 6 0 | cat -e
          $
 O   OOO  $
 O     O  $
 OOO  O   $
  O  OOO  $
          $

$> echo 'dxss' | ./life 3 3 0 | cat -e
 O $
 O $
 O $

$> echo 'dxss' | ./life 3 3 1 | cat -e
   $
OOO$
   $

$> echo 'dxss' | ./life 3 3 2 | cat -e
 O $
 O $
 O $
```

### Rules of the simulation

- A living cell with 2 or 3 living neighbours stays alive, otherwise it dies.
- A dead cell with exactly 3 living neighbours becomes alive.
- Neighbours are the 8 surrounding cells; anything off the board counts as dead.
- The pen draws the cell it rests on after each command while the pen is down,
  including the moment `x` lowers it. Movement is clamped inside the board.
