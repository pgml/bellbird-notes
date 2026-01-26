const std = @import("std");
const vaxis = @import("vaxis");

const App = @import("App.zig");
const log = @import("log.zig");

pub const std_options: std.Options = .{
    .logFn = log.toFile,
};

pub const KnownFolderConfig = struct {
    xdg_force_default: bool = false,
    xdg_on_mac: bool = false,
};

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer {
        const deinit_status = gpa.deinit();
        if (deinit_status == .leak) {
            std.log.err("memory leak", .{});
        }
    }
    const alloc = gpa.allocator();

    try log.init(alloc);
    defer log.deinit();

    //const args = try std.process.argsAlloc(alloc);
    //defer std.process.argsFree(alloc, args);

    //for (args) |arg| {
    //    std.debug.print("  {s}\n", .{arg});
    //}

    var app = try App.init(alloc);
    defer app.deinit();
    try app.run();
}
