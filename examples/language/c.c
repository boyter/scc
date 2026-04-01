#include "cheader.h"
#include <stdio.h>
#include <stdlib.h>

// generate table
table_t generate_table(int width)
{
    if (width <= 0) {
        return NULL;
    }

    table_t table = (table_t)malloc(sizeof(int*)*width);
    for (int i = 0; i < width; ++i) {
        int *row = (int*)calloc(width, sizeof(int));
        for (int j = 0; j <= i; ++j) {
            row[j] = (i+1) * (j+1);
        }
        table[i] = row;
    }

    return table;
}

#define WIDTH 9

int main(void)
{
    table_t table = generate_table(WIDTH);
    if (table == NULL) {
        return 1;
    }

    for (int i = WIDTH; i > 0; --i) {
        for (int j = 0; j <= i-1; ++j) {
            printf("%dx%d=%d ", (j+1), i, table[i-1][j]);
        }
        printf("\n");
    }
    free_table(table, WIDTH);

    return 0;
}

void free_table(table_t table, int width)
{
    if (table == NULL || width <= 0) {
        return;
    }

    for (int i = 0; i < width; ++i) {
        free(table[i]);
    }
    free(table);
}
