#pragma once

typedef int **table_t;

extern table_t generate_table(int width);
extern void free_table(table_t table, int width);
