package MyApp

import System

func collatz(num int32) slice[int32] {
    if num <= 0 {
        throw System.InvalidOperationException("num must be >= 0") /* collatz sequence needs num >= 0 */
    }
    var n = num
    var res slice[int32]
    res = res.Append(n)
    for n > 1 {
        if n&1 == 0 {
            n /= 2
        } else {
            n *= 3
            n++
        }
        res = res.Append(n)
    }
    return res
}

func Main() {
    scope {
        for i in -1 ... 11 {
            go func(ii int32) {
                try {
                    Console.WriteLine("${ii} : ${string.Join(",", collatz(ii))}")
                } catch (e System.InvalidOperationException) {
                    Console.WriteLine("${ii} : Error: ${e.Message}")
                }
            }(i)
        }
    }
}
