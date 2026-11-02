// SPDX-License-Identifier: MIT

//! A small Rust sample, kept here so the fuzzer has a seed and the differential
//! walk has a file to find. Nothing here is run.

use std::collections::HashMap;

/// A lifetime, a character literal and a raw string in one place, which is the
/// shape a Rust counter has to tell apart.
struct Holder<'a> {
    name: &'a str,
    counts: HashMap<String, usize>,
}

impl<'a> Holder<'a> {
    fn new(name: &'a str) -> Self {
        Holder { name, counts: HashMap::new() }
    }

    fn tally(&mut self, words: &[&str]) -> usize {
        let mut total = 0;
        for word in words {
            if word.is_empty() {
                continue;
            }
            *self.counts.entry(word.to_string()).or_insert(0) += 1;
            total += 1;
        }

        total
    }
}

fn quoting() {
    let quote = '"';
    let escaped = '\\';
    let raw = r"a \ backslash and a " ;
    let tagged = r#"holds a " quote"#;
    let bytes = b"plain bytes";
    println!("{quote} {escaped} {raw} {tagged} {:?}", bytes);
}

/* A block comment
   /* that nests, which Rust allows and C does not */
   and carries on after the inner one closes. */
fn main() {
    let mut holder = Holder::new("sample");
    let total = holder.tally(&["one", "two", "", "two"]);
    match total {
        0 => println!("nothing"),
        n if n > 2 => println!("many: {n}"),
        _ => println!("some"),
    }
    quoting();
}
