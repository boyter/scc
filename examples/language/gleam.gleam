import gleam/int
import gleam/io

pub fn main() {
  let x = int.random(2)
  let y = int.random(2)
  io.debug(x)
  io.debug(y)

  let result = case x, y {
    0, 0 -> "Both are zero"
    0, _ -> "First is zero"
    _, 0 -> "Second is zero"
    _, _ -> "Neither are zero"
  }
  io.debug(result)
}