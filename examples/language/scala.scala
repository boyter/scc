// SPDX-License-Identifier: MIT

/* A block comment
   /* that nests, which Scala allows */
   and carries on. */

package example

import scala.collection.mutable

final class Holder(val name: String) {
  private val counts = mutable.Map.empty[String, Int]

  def tally(words: Seq[String]): Int = {
    var total = 0
    for (word <- words if word.nonEmpty) {
      counts.update(word, counts.getOrElse(word, 0) + 1)
      total += 1
    }

    total
  }

  def describe: String = {
    val quoted = "a string holding a \" quote"
    val raw =
      """a raw string
        |holding // something that is not a comment""".stripMargin

    s"$name $quoted $raw"
  }
}

object Main {
  def main(args: Array[String]): Unit = {
    val holder = new Holder("sample")
    val total = holder.tally(Seq("one", "two", "", "two"))
    total match {
      case 0            => println("nothing")
      case n if n > 2   => println("many: " + n)
      case _            => println("some")
    }
    while (total < 0) {
      println("never")
    }
    println(holder.describe)
  }
}
