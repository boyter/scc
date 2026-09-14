// SPDX-License-Identifier: MIT

/* A block comment
   over more than one line. */

using System;
using System.Collections.Generic;

namespace Example
{
    /// <summary>A documentation comment, which is a line comment with a third slash.</summary>
    public sealed class Holder
    {
        private readonly Dictionary<string, int> counts = new();

        public Holder(string name) => Name = name;

        public string Name { get; }

        public int Tally(IEnumerable<string> words)
        {
            var total = 0;
            foreach (var word in words)
            {
                if (string.IsNullOrEmpty(word))
                {
                    continue;
                }

                counts[word] = counts.GetValueOrDefault(word) + 1;
                total++;
            }

            return total;
        }

        public string Describe()
        {
            var quoted = "a string holding a \" quote";
            var verbatim = @"a verbatim string holding a "" quote and a \ backslash";

            return $"{Name} {quoted} {verbatim}";
        }
    }

    public static class Program
    {
        public static void Main()
        {
            var holder = new Holder("sample");
            var total = holder.Tally(new[] { "one", "two", "", "two" });
            switch (total)
            {
                case 0:
                    Console.WriteLine("nothing");
                    break;
                default:
                    Console.WriteLine(total > 2 ? "many" : "some");
                    break;
            }

            try
            {
                Console.WriteLine(holder.Describe());
            }
            catch (Exception)
            {
                Console.WriteLine("failed");
            }
            finally
            {
                Console.WriteLine("done");
            }
        }
    }
}
