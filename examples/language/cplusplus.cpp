// compile with C++23
#include "cplusplusheader.hpp"

int main()
{
    constexpr auto table = generate_table<9>();
    static_assert(table.size(), "empty table");
    print_table(table);
}
