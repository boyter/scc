// 4 Complexity
#include <iostream>

int main()
{
    int i = 0;
    while(1) {
        std::cin >> i;
        if(i == 0) {
            return 0;
        }
        switch(i) {
        case 1:
            std::cout << "one\n";
            break;
        case 2:
            std::cout << "two\n";
            break;
        case 3:
            std::cout << "three\n";
        default:
            std::cout << "try again\n";
        }
    }
}
