import std.sync.*
import std.time.*
import std.collection.*

let count = AtomicInt64(0)

main(): Int64 {
    let list = ArrayList<Future<Int64>>()

    // create 1000 threads.
    for (i in 0..1000) {
        let fut = spawn {
            sleep(Duration.millisecond) // sleep for 1ms.
            count.fetchAdd(1)
        }
        list.append(fut)
    }

    // Wait for all threads finished.
    for (f in list) {
        f.get()
    }

    let val = count.load()
    println("count = ${val}")
    return 0
}
