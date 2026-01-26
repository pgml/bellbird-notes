const Config = @This();

const std = @import("std");
const known_folders = @import("known-folders");
const microwave = @import("microwave");

const DiretoryTree = @import("tui/DirectoryTree.zig");
const NotesLIst = @import("tui/NotesList.zig");
const TextArea = @import("tui/widgets/TextArea/TextArea.zig");
const theme = @import("tui/layout/theme.zig");
const utils = @import("utils.zig");

pub const application_name = "bellbird-notes-dev";
pub const config_file_name = "config";
pub const meta_file_name = ".metainfos";

alloc: std.mem.Allocator,

meta_infos: *MetaInfos,

pub const MetaInfos = struct {
    alloc: std.mem.Allocator,

    /// The path of the meta info file.
    file_path: []const u8 = "",

    /// A list of all open notes
    last_notes: std.ArrayList(u8) = .empty,

    /// The last open note
    last_open_note: []const u8 = "",

    /// The last opened directory
    last_directory: []const u8 = "",

    /// Currently selected column
    current_column: u16 = 1,

    /// Map of files and directories that holds information about
    /// of a file's cursor position and pinned state and a directorie's
    /// expanded and pinned state.
    files_info: std.StringArrayHashMap(FileInfo) = undefined,

    const FileInfo = struct {
        is_expanded: bool = false,
        is_pinned: bool = false,
        cursor_pos: TextArea.Buffer.CursorPos = .{},
    };

    /// All available meta options
    pub const Options = enum {
        last_directory,
        last_open_note,
        current_column,
        file_info,
        last_notes,
        is_expanded,
        is_pinned,
        cursor_pos,

        const tbl = [@typeInfo(Options).@"enum".fields.len][:0]const u8{
            "last_directory",
            "last_open_note",
            "current_column",
            "file_info",
            "last_notes",
            "expanded",
            "pinned",
            "cursor_position",
        };

        fn str(self: Options) [:0]const u8 {
            return tbl[@intFromEnum(self)];
        }
    };

    pub fn init(alloc: std.mem.Allocator) !*MetaInfos {
        const self = try alloc.create(MetaInfos);
        self.* = .{ .alloc = alloc };
        return self;
    }

    /// Loads the meta info file and populate the meta info struct with the
    /// data from the file.
    pub fn loadAndPopulate(self: *MetaInfos, file_path: []const u8) !void {
        self.files_info = .init(self.alloc);
        self.file_path = try self.alloc.dupe(u8, file_path);

        const file = std.fs.createFileAbsolute(self.file_path, .{
            .truncate = false,
            .read = true,
        }) catch |e| {
            std.log.debug("{}", .{e});
            return;
        };
        defer file.close();
        const stat = try file.stat();

        const buf_size = @max(stat.size, 128);
        const read_buf = try self.alloc.alloc(u8, buf_size);
        defer self.alloc.free(read_buf);

        var reader = file.reader(read_buf);
        const doc = try microwave.parseFromReader(self.alloc, &reader.interface);
        defer doc.deinit();

        var iter = doc.table.iterator();
        // iterate through the parse meta infos file and populate the struct
        while (iter.next()) |entry| {
            const key = entry.key_ptr.*;
            const val = entry.value_ptr;
            const Opts = Options;

            if (utils.strEql(key, Opts.last_directory.str())) {
                self.last_directory = try self.alloc.dupe(u8, val.string);
            }

            if (utils.strEql(key, Opts.last_open_note.str())) {
                self.last_open_note = try self.alloc.dupe(u8, val.string);
            }

            if (utils.strEql(key, Opts.current_column.str())) {
                self.current_column = @intCast(val.integer);
            }

            // popule files info map
            if (val.anyTableOrNull()) |table| {
                var is_expanded = false;
                var is_pinned = false;
                var cursor_pos: TextArea.Buffer.CursorPos = .{};

                if (table.getEntry(Opts.is_expanded.str())) |expanded| {
                    is_expanded = expanded.value_ptr.bool;
                }

                if (table.getEntry(Opts.is_pinned.str())) |pinned| {
                    is_pinned = pinned.value_ptr.bool;
                }

                // create a cursor object
                if (table.getEntry(Opts.cursor_pos.str())) |cp| {
                    if (cp.value_ptr.anyArrayOrNull()) |array| {
                        cursor_pos.row = @intCast(array.items[0].integer);
                        cursor_pos.row_offset = @intCast(array.items[1].integer);
                        cursor_pos.col = @intCast(array.items[2].integer);
                    }
                }

                const section = try self.alloc.dupe(u8, key);
                try self.files_info.put(section, .{
                    .is_expanded = is_expanded,
                    .is_pinned = is_pinned,
                    .cursor_pos = cursor_pos,
                });
            }
        }
    }

    /// Updates a meta info value
    pub fn setValue(self: *MetaInfos, opt: Options, val: anytype) !void {
        const ValType = @TypeOf(val);
        const val_type_info = @typeInfo(ValType);

        switch (val_type_info) {
            .int,
            .float,
            .bool,
            => {
                if (opt == .current_column) {
                    self.current_column = val;
                }
            },
            .pointer => |ptr| {
                if (ptr.child == u8) {
                    if (opt == .last_directory) {
                        self.last_directory = val;
                    } else if (opt == .last_open_note) {
                        self.last_open_note = val;
                    }
                }
            },
            else => {},
        }
    }

    pub fn addFileInfo(self: *MetaInfos, file: anytype) !void {
        try self.files_info.put(file.data.path, .{
            .is_expanded = file.is_expanded,
            .is_pinned = file.data.is_pinned,
        });

        try self.write();
    }

    pub fn updateFileInfo(
        self: *MetaInfos,
        filepath: []const u8,
        opt: Options,
        val: anytype,
    ) !void {
        // this is a cheap check but it's okay for now.
        const is_file = !utils.strEql(std.fs.path.extension(filepath), "");
        const file = try self.files_info.getOrPut(filepath);
        const ValType = @TypeOf(val);

        if (ValType == bool) {
            if (opt == .is_expanded) file.value_ptr.is_expanded = val;
            if (opt == .is_pinned) file.value_ptr.is_pinned = val;
        }

        if (is_file) {
            if (ValType == TextArea.Buffer.CursorPos) {
                if (opt == .cursor_pos) file.value_ptr.cursor_pos = val;
            }
        }

        try self.write();
    }

    /// Writes the info struct to the meta info file.
    pub fn write(self: *MetaInfos) !void {
        const file = std.fs.createFileAbsolute(
            self.file_path,
            .{ .truncate = true },
        ) catch |e| {
            std.log.debug("{}", .{e});
            return;
        };
        defer file.close();

        var buf: [4096]u8 = undefined;
        var writer = file.writer(&buf);

        var write_stream: microwave.WriteStream = .{
            .allocator = self.alloc,
            .writer = &writer.interface,
        };
        defer write_stream.deinit();

        try write_stream.beginKeyPair(Options.last_directory.str());
        try write_stream.writeString(self.last_directory);

        try write_stream.beginKeyPair(Options.last_open_note.str());
        try write_stream.writeString(self.last_open_note);

        try write_stream.beginKeyPair(Options.current_column.str());
        try write_stream.writeInteger(self.current_column);

        var file_iter = self.files_info.iterator();
        while (file_iter.next()) |entry| {
            const section = entry.key_ptr.*;
            const val = entry.value_ptr;
            // this is a cheap check but it's okay for now.
            const is_file = !utils.strEql(std.fs.path.extension(section), "");

            try write_stream.writeTable(section);
            try write_stream.beginKeyPair(Options.is_pinned.str());
            try write_stream.writeBoolean(val.is_pinned);

            if (!is_file) {
                // Store the expanded state for directorie's only
                try write_stream.beginKeyPair(Options.is_expanded.str());
                try write_stream.writeBoolean(val.is_expanded);
            } else {
                // For files we only need cursor position
                try write_stream.beginKeyPair(Options.cursor_pos.str());
                try write_stream.beginArray();
                {
                    try write_stream.writeInteger(val.cursor_pos.row);
                    try write_stream.writeInteger(val.cursor_pos.row_offset);
                    try write_stream.writeInteger(val.cursor_pos.col);
                }
                try write_stream.endArray();
            }
        }

        try write_stream.writer.writeByte('\n');
        try write_stream.writer.flush();
    }

    pub fn deinit(self: *MetaInfos) void {
        self.alloc.free(self.last_directory);
        self.alloc.free(self.last_open_note);
        var iter = self.files_info.iterator();
        while (iter.next()) |entry| {
            self.alloc.free(entry.key_ptr.*);
        }
        self.files_info.deinit();
        self.alloc.free(self.file_path);
    }
};

pub fn init(alloc: std.mem.Allocator) !*Config {
    const self = try alloc.create(Config);

    self.* = .{
        .alloc = alloc,
        .meta_infos = try .init(alloc),
    };

    return self;
}

/// Loads the meta info file.
pub fn loadMetaInfo(self: Config) !void {
    var arena: std.heap.ArenaAllocator = .init(self.alloc);
    defer arena.deinit();
    const arena_alloc = arena.allocator();
    const file_path = try std.mem.concat(arena_alloc, u8, &[_][]const u8{
        try self.getNotesRootDir(),
        "/",
        meta_file_name,
    });
    defer arena_alloc.free(file_path);
    try self.meta_infos.loadAndPopulate(file_path);
}

/// Returns the path to the config directory
pub fn getConfDirPath(alloc: std.mem.Allocator) ![]const u8 {
    var arena: std.heap.ArenaAllocator = .init(alloc);
    defer arena.deinit();
    const arena_alloc = arena.allocator();

    if (try known_folders.getPath(arena_alloc, .local_configuration)) |conf_home_path| {
        const app_dir_path: []const u8 = try std.mem.concat(
            arena_alloc,
            u8,
            &[_][]const u8{ conf_home_path, "/", application_name },
        );

        try getCreateDir(app_dir_path, conf_home_path);

        return alloc.dupe(u8, app_dir_path);
    }

    return "";
}

/// Returns the path to the root directory of the notes.
pub fn getNotesRootDir(self: Config) ![]const u8 {
    var arena: std.heap.ArenaAllocator = .init(self.alloc);
    defer arena.deinit();
    const arena_alloc = arena.allocator();

    if (try known_folders.getPath(arena_alloc, .home)) |home_path| {
        const notes_dir_path: []const u8 = try std.mem.concat(
            arena_alloc,
            u8,
            &[_][]const u8{ home_path, "/.", application_name },
        );

        try getCreateDir(notes_dir_path, home_path);

        return notes_dir_path;
    }

    return "";
}

fn getCreateDir(dir_path: []const u8, parent: []const u8) !void {
    var parent_dir: std.fs.Dir = try std.fs.openDirAbsolute(parent, .{});

    parent_dir.makePath(dir_path) catch |err| {
        std.log.err("Failed to create config directory: {}", .{err});
        return err;
    };
}

pub fn deinit(self: Config) void {
    self.meta_infos.deinit();
    self.alloc.destroy(self.meta_infos);
}
