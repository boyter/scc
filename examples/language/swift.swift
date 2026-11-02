// SPDX-License-Identifier: MIT

/* A block comment
   /* that nests, which Swift allows */
   and carries on. */

import Foundation

struct Holder {
    let name: String
    private var counts: [String: Int] = [:]

    init(name: String) {
        self.name = name
    }

    mutating func tally(_ words: [String]) -> Int {
        var total = 0
        for word in words where !word.isEmpty {
            counts[word, default: 0] += 1
            total += 1
        }

        return total
    }
}

func describe(_ holder: Holder?) -> String {
    guard let holder = holder else {
        return "nothing"
    }

    let quoted = "a string holding a \" quote"
    let multiline = """
        a multiline string
        holding // something that is not a comment
        """

    return "\(holder.name) \(quoted) \(multiline)"
}

var holder = Holder(name: "sample")
let total = holder.tally(["one", "two", "", "two"])
switch total {
case 0:
    print("nothing")
case let n where n > 2:
    print("many: \(n)")
default:
    print("some")
}
print(describe(holder))
