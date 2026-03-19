from std.collections import InlineArray

struct StaticMultiplicationTable:
    comptime length = 81
    var data: InlineArray[Int, Self.length]

    fn __init__(out self):
        self.data = InlineArray[Int, Self.length](fill=0)
        
        # unroll the loop
        # all completed during compiling time
        comptime for i in range(1, 10):
            comptime for j in range(1, 10):
                self.data[(i-1)*9 + (j-1)] = i * j

    fn display(self):
        comptime for i in range(1, 10):
            var line: String = ""
            comptime for j in range(1, 10):
                if j <= i:
                    var val = self.data[(i-1)*9 + (j-1)]
                    line += String(j) + "x" + String(i) + "=" + String(val) + "\t"
            print(line)

fn main():
    comptime table_const = StaticMultiplicationTable()
    var table = materialize[table_const]()
    
    # run
    table.display()
