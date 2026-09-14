// SPDX-License-Identifier: MIT

/* A block comment
   /* that nests, which Kotlin allows */
   and carries on. */

package example

class Holder(private val name: String) {
    private val counts = mutableMapOf<String, Int>()

    fun tally(words: List<String>): Int {
        var total = 0
        for (word in words) {
            if (word.isEmpty()) {
                continue
            }
            counts[word] = (counts[word] ?: 0) + 1
            total++
        }

        return total
    }

    fun describe(): String {
        val quoted = "a string holding a \" quote"
        val raw = """
            a raw string
            holding // something that is not a comment
            """.trimIndent()

        return "$name $quoted $raw"
    }
}

fun main() {
    val holder = Holder("sample")
    val total = holder.tally(listOf("one", "two", "", "two"))
    when {
        total == 0 -> println("nothing")
        total > 2 -> println("many: $total")
        else -> println("some")
    }
    try {
        println(holder.describe())
    } catch (e: Exception) {
        println("failed")
    } finally {
        println("done")
    }
}
