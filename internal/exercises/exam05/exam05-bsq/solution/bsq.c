#define _POSIX_C_SOURCE 200809L
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/types.h>

struct map {
    size_t height, width;
    char empty, obstacle, fill;
    char **rows;
};

static size_t count(const char *text, size_t length)
{
    size_t result = 0;
    if (!length) return 0;
    for (size_t i = 0; i < length; ++i) {
        if (text[i] < '0' || text[i] > '9' || result > (SIZE_MAX - (text[i] - '0')) / 10)
            return 0;
        result = result * 10 + text[i] - '0';
    }
    return result;
}

static int header(struct map *map, const char *text, size_t length)
{
    if (length < 5 || text[length - 1] != '\n') return 0;
    --length;
    char markers[3];
    map->height = count(text, length - 3);
    if (map->height) {
        for (size_t i = 0; i < 3; ++i) markers[i] = text[length - 3 + i];
    } else {
        size_t position = 0;
        while (position < length && text[position] >= '0' && text[position] <= '9') ++position;
        map->height = count(text, position);
        for (size_t i = 0; i < 3; ++i) {
            if (position >= length || text[position] != ' ') return 0;
            while (position < length && text[position] == ' ') ++position;
            if (position == length) return 0;
            markers[i] = text[position++];
        }
        if (position != length) return 0;
    }
    for (size_t i = 0; i < 3; ++i) {
        if (markers[i] < 32 || markers[i] > 126) return 0;
    }
    if (!map->height || markers[0] == markers[1] || markers[0] == markers[2] || markers[1] == markers[2])
        return 0;
    map->empty = markers[0]; map->obstacle = markers[1]; map->fill = markers[2];
    return 1;
}

static int load(FILE *input, struct map *map)
{
    char *line = NULL;
    size_t capacity = 0;
    ssize_t length = getline(&line, &capacity, input);
    int valid = length > 0 && header(map, line, (size_t)length);
    free(line);
    if (!valid || map->height > SIZE_MAX / sizeof(*map->rows)) return 0;
    map->rows = calloc(map->height, sizeof(*map->rows));
    if (!map->rows) return 0;
    for (size_t y = 0; y < map->height; ++y) {
        capacity = 0;
        length = getline(&map->rows[y], &capacity, input);
        if (length < 2 || map->rows[y][length - 1] != '\n') return 0;
        if (!y) map->width = (size_t)length - 1;
        if ((size_t)length - 1 != map->width) return 0;
        for (size_t x = 0; x < map->width; ++x) {
            if (map->rows[y][x] != map->empty && map->rows[y][x] != map->obstacle)
                return 0;
        }
    }
    line = NULL;
    capacity = 0;
    length = getline(&line, &capacity, input);
    free(line);
    return length == -1;
}

static int fill_square(struct map *map)
{
    if (map->width == SIZE_MAX || map->width + 1 > SIZE_MAX / sizeof(size_t)) return 0;
    size_t *lengths = calloc(map->width + 1, sizeof(*lengths));
    if (!lengths) return 0;
    size_t best = 0, bottom = 0, right = 0;
    for (size_t y = 0; y < map->height; ++y) {
        size_t diagonal = 0;
        for (size_t x = 1; x <= map->width; ++x) {
            size_t above = lengths[x];
            size_t minimum = above < lengths[x - 1] ? above : lengths[x - 1];
            if (diagonal < minimum) minimum = diagonal;
            lengths[x] = map->rows[y][x - 1] == map->empty ? minimum + 1 : 0;
            diagonal = above;
            if (lengths[x] > best) { best = lengths[x]; bottom = y + 1; right = x; }
        }
    }
    free(lengths);
    for (size_t y = bottom - best; y < bottom; ++y)
        for (size_t x = right - best; x < right; ++x) map->rows[y][x] = map->fill;
    return 1;
}

static void process(FILE *input)
{
    struct map map = {0, 0, 0, 0, 0, NULL};
    if (!input || !load(input, &map) || !fill_square(&map))
        fprintf(stderr, "map error\n");
    else
        for (size_t y = 0; y < map.height; ++y) fprintf(stdout, "%s", map.rows[y]);
    if (map.rows) {
        for (size_t y = 0; y < map.height; ++y) free(map.rows[y]);
        free(map.rows);
    }
}

int main(int argc, char **argv)
{
    if (argc == 1) process(stdin);
    for (int i = 1; i < argc; ++i) {
        FILE *input = fopen(argv[i], "r");
        process(input);
        if (input) fclose(input);
        if (i + 1 < argc) fprintf(stdout, "\n");
    }
    return 0;
}
