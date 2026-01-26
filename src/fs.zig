const std = @import("std");
const Walker = std.fs.Dir.Walker;

const Config = @import("Config.zig");

pub const Directories = struct {
    pub const Entry = struct {
        basename: []const u8 = "",
        path: []const u8 = "",
        num_files: usize = 0,
        num_dirs: usize = 0,
    };

    pub fn list(alloc: std.mem.Allocator, path: []const u8) ![]Entry {
        var entries: std.ArrayList(Entry) = .empty;
        errdefer entries.deinit(alloc);

        var dir = try std.fs.openDirAbsolute(path, .{ .iterate = true });
        defer dir.close();

        var iter = dir.iterate();

        while (try iter.next()) |entry| {
            if (entry.kind != .directory and entry.kind != .sym_link) {
                continue;
            }

            const dir_path = try std.fs.path.join(alloc, &.{ path, entry.name });
            defer alloc.free(dir_path);

            try entries.append(alloc, .{
                .basename = try alloc.dupe(u8, entry.name),
                .path = try alloc.dupe(u8, dir_path),
                .num_files = try getChildCount(dir_path, .file),
                .num_dirs = try getChildCount(dir_path, .directory),
            });
        }

        const owned = try entries.toOwnedSlice(alloc);
        return owned;
    }

    pub fn getChildCount(path: []const u8, kind: std.fs.File.Kind) !usize {
        var dir = try std.fs.openDirAbsolute(path, .{ .iterate = true });
        defer dir.close();

        var iter = dir.iterate();
        var count: usize = 0;

        while (try iter.next()) |entry| {
            if (entry.kind != kind) {
                continue;
            }

            count += 1;
        }

        return count;
    }
};

pub const Notes = struct {
    pub const Entry = struct {
        name: []const u8,
        path: []const u8,
    };

    pub fn list(alloc: std.mem.Allocator, path: []const u8) ![]Entry {
        var entries: std.ArrayList(Entry) = .empty;
        errdefer entries.deinit(alloc);

        var dir = try std.fs.openDirAbsolute(path, .{ .iterate = true });
        defer dir.close();

        var iter = dir.iterate();

        while (try iter.next()) |entry| {
            if (entry.kind != .file) {
                continue;
            }

            const dir_path = try std.fs.path.join(alloc, &.{ path, entry.name });
            defer alloc.free(dir_path);

            try entries.append(alloc, .{
                .name = try alloc.dupe(u8, entry.name),
                .path = try alloc.dupe(u8, dir_path),
            });
        }

        const owned = try entries.toOwnedSlice(alloc);
        return owned;
    }
};
