const std = @import("std");

pub fn strEql(a: []const u8, b: []const u8) bool {
    return std.mem.eql(u8, a, b);
}
