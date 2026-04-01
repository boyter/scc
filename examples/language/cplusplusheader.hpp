#pragma once
// compile with C++23
#include <array>
#include <print>

template <std::size_t N>
constexpr std::array<std::array<std::size_t, N>, N> generate_table() noexcept
{
    if constexpr (N == 0) {
        return {};
    }

    std::array<std::array<std::size_t, N>, N> table;

    for (auto i = std::size_t{0}; i < N; ++i) {
        auto row = std::array<std::size_t, N>{};
        for (auto j = std::size_t{0}; j <= i; ++j) {
            row[j] = (i+1) * (j+1);
        }
        table[i] = std::move(row);
    }

    return table;
}

template <typename T, std::size_t N>
void print_table(const std::array<std::array<T, N>, N> &table) noexcept
{
    if constexpr (N == 0) {
        return;
    }

    for (auto i = N; i > 0; --i) {
        for (auto j = std::size_t{0}; j <= i-1; ++j) {
            std::print("{}x{}={} ", (j+1), i, table[i-1][j]);
        }
        std::print("\n");
    }
}
