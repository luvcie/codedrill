#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

// 1. Allocate a blank board that is empty by default
char** makeNewBoard(int width, int height) {
    char **board = calloc(height, sizeof *board);
    for (int row = 0; row < height; ++row) {
        board[row] = calloc(width, 1);
    }
    return board;
}

// 2. Prepare the initial board by reading from stdin
char** initFirstBoard(int width, int height) {

    char **board = makeNewBoard(width, height);

    // Set up variables
    int row = 0;
    int col = 0;
    int drawing = 0;
    char cmd;

    while (read(0, &cmd, 1) == 1) {
        // Move cursor, WASD controls
        if (cmd == 'w' && row > 0) {
            --row;
        }
        if (cmd == 'a' && col > 0) {
            --col;
        }
        if (cmd == 's' && row < height - 1) {
            ++row;
        }
        if (cmd == 'd' && col < width - 1) {
            ++col;
        }
        // Toggle drawing if desired
        if (cmd == 'x') {
            drawing = !drawing;
        }
        // Plot 'O' if drawing
        if (drawing) {
            board[row][col] = 'O';
        }
    }
    return board;
}

// 3. Count how many tiles surround the given cell.
int countNeighbors(char **board, int width, int height, int row, int col) {

    int neighbors = 0;
    for (int drow = -1; drow <= 1; ++drow) {
        for (int dcol = -1; dcol <= 1; ++dcol) {
            if (drow == 0 && dcol == 0) {
                continue;
            }

            int neighborRow = row + drow;
            int neighborCol = col + dcol;

            if (neighborRow < 0 || neighborRow >= height) {
                continue;
            }
            if (neighborCol < 0 || neighborCol >= width) {
                continue;
            }
            if (board[neighborRow][neighborCol] == 'O') {
                ++neighbors;
            }
        }
    }
    return neighbors;
}

// 4. Perform the next step of the game of life and generate the next board state
char** generateNextBoard(char **board, int width, int height) {
    char **nextBoard = makeNewBoard(width, height);

    for (int row = 0; row < height; ++row) {
        for (int col = 0; col < width; ++col) {
            // Count live neighbors around this cell.
            int neighbors = countNeighbors(board, width, height, row, col);

            // Three neighbors gives life. Two keeps a live cell alive.
            if (neighbors == 3 || (neighbors == 2 && board[row][col] == 'O')) {
                nextBoard[row][col] = 'O';
            }
        }
    }
    return nextBoard;
}

// 5. Print the board
void printBoard(char **board, int width, int height) {
    for (int row = 0; row < height; ++row) {
        for (int col = 0; col < width; ++col) {
            putchar(board[row][col] == 'O' ? 'O' : ' ');
        }
        putchar('\n');
    }
}

// 6. Final main - read arguments, initialise board
// play the game of life, and print
int main(int argc, char **argv)
{
    if (argc != 4) {
        return 1;
    }
    int width = atoi(argv[1]);
    int height = atoi(argv[2]);
    int steps = atoi(argv[3]);

    char **board = initFirstBoard(width, height);

    for (int step = 0; step < steps; ++step) {
        board = generateNextBoard(board, width, height);
    }

    printBoard(board, width, height);

    return 0;
}
