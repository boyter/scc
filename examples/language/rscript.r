#!/usr/bin/env Rscript

print("===== Start =====")

for (i in 1:9) {
    for (j in 1:i) {
        cat(i, 'x', j, '=', i*j, ' ')
    }
    cat('\n')
}

print("===== End =====")
