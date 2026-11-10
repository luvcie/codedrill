#include <limits.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

static int number(const char *text)
{
    int value = 0;
    if (!*text) return -1;
    for (; *text; ++text) {
        if (*text < '0' || *text > '9' || value > (INT_MAX - (*text - '0')) / 10)
            return -1;
        value = value * 10 + *text - '0';
    }
    return value;
}

int main(int argc, char **argv)
{
    if (argc != 4) return 1;
    int width = number(argv[1]), height = number(argv[2]), steps = number(argv[3]);
    if (width <= 0 || height <= 0 || steps < 0 || (size_t)height > SIZE_MAX / width)
        return 1;
    size_t cells = (size_t)width * height;
    unsigned char *board = calloc(cells, 1), *next = calloc(cells, 1);
    if (!board || !next) { free(board); free(next); return 1; }
    int column = 0, row = 0, drawing = 0;
    char command;
    ssize_t received;
    while ((received = read(STDIN_FILENO, &command, 1)) > 0) {
        switch (command) {
        case 'x': drawing = !drawing; break;
        case 'a': if (column > 0) --column; break;
        case 'd': if (column + 1 < width) ++column; break;
        case 'w': if (row > 0) --row; break;
        case 's': if (row + 1 < height) ++row; break;
        }
        if (drawing) board[(size_t)row * width + column] = 1;
    }
    if (received < 0) { free(board); free(next); return 1; }
    for (int generation = 0; generation < steps; ++generation) {
        for (int y = 0; y < height; ++y) {
            for (int x = 0; x < width; ++x) {
                int neighbors = 0;
                for (int dy = -1; dy <= 1; ++dy) {
                    for (int dx = -1; dx <= 1; ++dx) {
                        if ((!dx && !dy) || x + dx < 0 || x + dx >= width ||
                            y + dy < 0 || y + dy >= height) continue;
                        neighbors += board[(size_t)(y + dy) * width + x + dx];
                    }
                }
                size_t position = (size_t)y * width + x;
                next[position] = neighbors == 3 || (board[position] && neighbors == 2);
            }
        }
        unsigned char *swap = board; board = next; next = swap;
    }
    for (int y = 0; y < height; ++y) {
        for (int x = 0; x < width; ++x)
            putchar(board[(size_t)y * width + x] ? 'O' : ' ');
        putchar('\n');
    }
    free(board);
    free(next);
    return 0;
}
