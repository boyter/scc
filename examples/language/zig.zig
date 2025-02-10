const std = @import("std");
const print = std.debug.print;

pub fn main() void {
    const arr = [_]u32{1, 2, 3, 4, 5, 6, 7, 8, 9};
    for (arr) |m| {
        var n: u32 = 1;
        while (n <= m) : (n += 1) {
            print("{} x {} = {}, ", .{n, m, n*m});
        }
        print("\n", .{});
    }
}
