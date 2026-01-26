const std = @import("std");

const Config = @import("Config.zig");

pub const app_log = "app.log";
pub const err_log = "err.log";
pub const application_name = "bellbird-notes";

var arena: std.heap.ArenaAllocator = undefined;
var alloc: std.mem.Allocator = undefined;

pub fn init(allocator: std.mem.Allocator) !void {
    arena = std.heap.ArenaAllocator.init(allocator);
    alloc = arena.allocator();
}

pub fn deinit() void {
    arena.deinit();
}

pub fn toFile(
    comptime message_level: std.log.Level,
    comptime scope: @TypeOf(.enum_literal),
    comptime format: []const u8,
    args: anytype,
) void {
    const path = switch (message_level) {
        .debug => app_log,
        .info => app_log,
        .warn => err_log,
        .err => err_log,
        //.err => return std.log.defaultLog(message_level, scope, format, args),
    };

    if (scope == ._ and message_level == .err) {
        path = err_log;
    } else if (message_level == .err) {
        return std.log.defaultLog(message_level, scope, format, args);
    }

    const conf_dir_path = Config.getConfDirPath(alloc) catch {
        return std.log.defaultLog(message_level, scope, format, args);
    };
    defer alloc.free(conf_dir_path);

    var conf_dir = std.fs.openDirAbsolute(conf_dir_path, .{}) catch {
        return std.log.defaultLog(message_level, scope, format, args);
    };

    const conf_file_path: []const u8 = std.mem.concat(alloc, u8, &[_][]const u8{
        conf_dir_path,
        "/",
        path,
    }) catch {
        return std.log.defaultLog(message_level, scope, format, args);
    };
    defer alloc.free(conf_file_path);

    const log = conf_dir.createFile(path, .{ .truncate = false }) catch {
        return std.log.defaultLog(message_level, scope, format, args);
    };

    var file = std.fs.cwd().openFile(
        conf_file_path,
        .{ .mode = .write_only },
    ) catch return;
    defer file.close();

    // Get a writer.
    // See https://ziglang.org/documentation/0.15.1/std/#std.log.defaultLog.
    var buffer: [64]u8 = undefined;
    var log_writer = log.writer(&buffer);

    // Move the write index to the end of the file.
    const end_pos = log.getEndPos() catch {
        return std.log.defaultLog(message_level, scope, format, args);
    };
    log_writer.seekTo(end_pos) catch {
        return std.log.defaultLog(message_level, scope, format, args);
    };

    var writer = &log_writer.interface;

    const level_txt = comptime message_level.asText();
    const prefix2 = if (scope == .default) ": " else "(" ++ @tagName(scope) ++ "): ";
    //std.debug.print("{s} {}", .{ prefix2, scope });

    nosuspend {
        writer.print(level_txt ++ prefix2 ++ format ++ "\n", args) catch {
            return std.log.defaultLog(message_level, scope, format, args);
        };

        writer.flush() catch {
            return std.log.defaultLog(message_level, scope, format, args);
        };
    }
}

pub fn err(comptime format: []const u8, args: anytype) void {
    const bblog = std.log.scoped(._);
    bblog.err(format, args);
}
