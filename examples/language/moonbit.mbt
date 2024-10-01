fn iter(data : Bytes) -> Iter[Byte] {
  Iter::new(
    fn(yield) {
      // The code that actually does the iteration
      /////////////////////////////////////////////
      for i = 0, len = data.length(); i < len; i = i + 1 {
        if yield(data[i]) == IterEnd {
          break IterEnd
        }
      /////////////////////////////////////////////
      } else {
        IterContinue
      }
    },
  )
}