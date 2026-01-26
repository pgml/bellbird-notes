const Buffer = @This();

const std = @import("std");
const Sha256 = std.crypto.hash.sha2.Sha256;

const vx = @import("vaxis");
const Key = vx.Key;

const Dmp = @import("diffmatchpatch");

const TextArea = @import("TextArea.zig");
const Char = TextArea.Character;
const Vim = @import("Vim.zig");

const HistoryError = error{
    EntryNotFound,
    NoUndoEntry,
};

/// A textarea buffer
arena: std.heap.ArenaAllocator,

arena_alloc: std.mem.Allocator,

index: usize = 0,

/// The cursor column index
col: i32 = 0,

/// Last cursor column, used to maintain state when the cursor is moved
/// vertically such that we can maintain the same navigating position.
last_col: i32 = 0,

/// The cursor row index
row: i32 = 0,

/// The number of rows on a multiline the cursor is offset from the start
/// of the line.
row_offset: u16 = 0,

/// List of all buffer rows.
rows: std.ArrayList(*Row),

/// Whether the buffer has any changes.
is_dirty: bool = false,

history: History,

read_buf: []u8 = std.mem.zeroes([]u8),

cursor_pos: CursorPos = .{},

/// The path of the file that is loaded into the buffer.
path: []const u8 = "",

pub const CursorPos = struct {
    row: i32 = 0,
    row_offset: u16 = 0,
    col: i32 = 0,
};

pub const History = struct {
    alloc: std.mem.Allocator,

    // EntryIndex is the current index in the history
    entry_index: i16 = -1,

    // entries holds all recorded undo/redo history entries.
    entries: std.ArrayList(Entry) = .empty,

    tmp_entry: Entry,

    // maxItems is the maximum number of entries allowed in history
    max_items: usize = 100,

    dmp: Dmp.DiffMatchPatch,

    const Entry = struct {
        alloc: std.mem.Allocator,

        redo_patch: []const u8 = "",

        undo_patch: []const u8 = "",

        undo_cursor_pos: CursorPos = .{},

        redo_cursor_pos: CursorPos = .{},

        hash: u64 = 0,

        pub fn init(alloc: std.mem.Allocator) Entry {
            return .{ .alloc = alloc };
        }

        pub fn deinit(self: Entry) void {
            self.alloc.free(self.undo_patch);
            self.alloc.free(self.redo_patch);
        }
    };

    pub fn init(alloc: std.mem.Allocator) !History {
        return .{
            .alloc = alloc,
            .tmp_entry = .init(alloc),
            .dmp = .init(alloc),
        };
    }

    /// Creates a new temporary entry.
    pub fn newTmpEntry(self: *History, cursor_pos: CursorPos) void {
        self.tmp_entry.undo_cursor_pos = cursor_pos;
    }

    /// Creates a new persistent history entry.
    /// If future entries exist (after undo), they are discarded.
    pub fn newEntry(self: *History, cursor_pos: CursorPos) !void {
        // if the current index is lower than the length of all entries
        // truncate the slice to the current index to get rid of all
        // the old entries so the history doesn't get too confusing
        if (self.numEntries() > 0 and self.entry_index < self.numEntries()) {
            for (@intCast(self.entry_index + 1)..self.numEntries()) |i| {
                if (i >= self.numEntries()) {
                    continue;
                }
                const entry = self.entries.orderedRemove(i);
                if (self.numEntries() > 0) {
                    self.entries.shrinkAndFree(self.alloc, self.numEntries() - 1);
                }
                entry.deinit();
            }
        }

        self.tmp_entry.undo_cursor_pos = cursor_pos;
        try self.entries.append(self.alloc, self.tmp_entry);

        self.entry_index = @intCast(self.numEntries() - 1);
    }

    /// Creates a new persistent history entry from he temporary entry.
    pub fn appendTmpEntry(self: *History) !void {
        try self.newEntry(self.tmp_entry.undo_cursor_pos);
        self.tmp_entry = .init(self.alloc);
    }

    /// Updates the current entry with patches and metadata.
    pub fn updateEntry(
        self: *History,
        redo_patch: Dmp.PatchList,
        undo_patch: Dmp.PatchList,
        cursor_pos: CursorPos,
        hash: u64,
    ) !void {
        try self.appendTmpEntry();

        if (self.entry_index >= self.numEntries() or self.entry_index < 0) {
            std.log.warn("History entry index {} not found.", .{self.entry_index});
            return HistoryError.EntryNotFound;
        }

        var entries = self.entries.items;
        const index: usize = @intCast(self.entry_index);

        entries[index].redo_patch = try self.dmp.patchToText(redo_patch);
        entries[index].undo_patch = try self.dmp.patchToText(undo_patch);
        entries[index].redo_cursor_pos = cursor_pos;
        entries[index].hash = hash;
    }

    /// Returns the entry at the given index or nil if out of bounds.
    pub fn getEntry(self: History, index: usize) !Entry {
        if (index > self.numEntries() - 1) {
            return HistoryError.EntryNotFound;
        }
        return self.entries.items[index];
    }

    /// Generates a diff patch between `old_str` and `new_str`.
    pub fn makePatch(
        self: *History,
        old_str: []const u8,
        new_str: []const u8,
    ) !Dmp.PatchList {
        return try self.dmp.patchMakeStringString(old_str, new_str);
    }

    /// Returns the undo patch, content hash, and cursor position.
    /// Returns HistoryError if no undo patch is available.
    pub fn undo(self: *History) !struct {
        patch: Dmp.PatchList,
        hash: u64,
        cursor_pos: CursorPos,
    } {
        if (self.entry_index < 0 or self.entry_index >= self.numEntries()) {
            return HistoryError.NoUndoEntry;
        }

        const entry: Entry = try self.getEntry(@intCast(self.entry_index));
        const cursor_pos: CursorPos = entry.undo_cursor_pos;
        const patch: Dmp.PatchList = try self.dmp.patchFromText(entry.undo_patch);

        self.entry_index -= 1;

        return .{
            .patch = patch,
            .hash = entry.hash,
            .cursor_pos = cursor_pos,
        };
    }

    /// Returns the redo patch, content hash, and cursor position.
    /// Returns HistoryError if no redo patch is available.
    pub fn redo(self: *History) !struct {
        patch: Dmp.PatchList,
        hash: u64,
        cursor_pos: CursorPos,
    } {
        if (self.entry_index + 1 >= self.numEntries()) {
            return HistoryError.NoUndoEntry;
        }

        self.entry_index += 1;

        const entry: Entry = try self.getEntry(@intCast(self.entry_index));
        const cursor_pos: CursorPos = entry.redo_cursor_pos;
        const patch: Dmp.PatchList = try self.dmp.patchFromText(entry.redo_patch);

        return .{
            .patch = patch,
            .hash = entry.hash,
            .cursor_pos = cursor_pos,
        };
    }

    fn numEntries(self: History) usize {
        return self.entries.items.len;
    }

    pub fn deinit(self: *History) void {
        for (self.entries.items) |entry| {
            entry.deinit();
        }
        self.entries.deinit(self.alloc);
    }
};

/// A single buffer row
pub const Row = struct {
    alloc: std.mem.Allocator,

    /// The buffer's text value
    value: std.ArrayList(Char),

    /// Cursor column
    col: i32,

    /// Offset from the top of the row on multilines
    offset: i16,

    pub fn init(alloc: std.mem.Allocator, offset: i16) !*Row {
        const self = try alloc.create(Row);

        self.* = .{
            .alloc = alloc,
            .value = .empty,
            .col = 0,
            .offset = offset,
        };

        return self;
    }

    pub fn insertSliceAtCursor(self: *Row, slice: []const u8) !void {
        try self.value.insert(self.alloc, @intCast(self.col), .{
            .grapheme = slice,
            .width = 1,
        });
    }

    pub fn len(self: Row) usize {
        return self.value.items.len;
    }

    pub fn getValue(self: Row) []Char {
        return self.value.items;
    }

    pub fn appendChar(self: *Row, char: Char) !void {
        const owned = try self.alloc.dupe(u8, char.grapheme);
        try self.value.append(self.alloc, .{
            .grapheme = owned,
            .width = char.width,
        });
        //try self.value.append(self.alloc, char);
    }

    pub fn deleteCharAt(self: *Row, index: usize) void {
        _ = self.value.orderedRemove(index);
    }

    pub fn shrinkAndFree(self: *Row) void {
        self.value.shrinkAndFree(self.alloc, self.len());
    }

    pub fn deinit(self: *Row) void {
        self.value.deinit(self.alloc);
    }
};

pub fn init(alloc: std.mem.Allocator) !*Buffer {
    const self = try alloc.create(Buffer);

    self.* = .{
        .arena = .init(alloc),
        .arena_alloc = self.arena.allocator(),
        .rows = .empty,
        .history = undefined,
    };

    // add first row
    try self.addRow(0);

    return self;
}

pub fn getName(self: Buffer) []const u8 {
    return std.fs.path.basename(self.path);
}

pub fn setIndex(self: *Buffer, index: usize) void {
    self.index = index;
}

pub fn setPath(self: *Buffer, path: []const u8) void {
    self.path = path;
}

pub fn setContentFromStr(self: *Buffer, content: []const u8) !void {
    self.arena_alloc.free(self.read_buf);
    self.rows.shrinkAndFree(self.arena_alloc, 0);
    try self.addRow(0);
    self.row = 0;
    self.col = 0;

    var iter = std.mem.splitAny(u8, content, "\n");
    var i: usize = 0;

    while (iter.next()) |line| {
        defer i += 1;

        if (i > 0) {
            try self.addRow(0);
        }

        var g_iter = vx.unicode.graphemeIterator(line);

        while (g_iter.next()) |g| {
            try self.curRow().appendChar(.{
                .grapheme = g.bytes(line),
                .width = 1,
            });
        }
    }
}

pub fn setContentFromFile(self: *Buffer, file_path: []const u8) !void {
    const file = try std.fs.openFileAbsolute(file_path, .{ .mode = .read_write });
    defer file.close();
    const stat = try file.stat();
    const size = stat.size;

    // Empty file - nothing to read.
    if (stat.size == 0) {
        try self.curRow().appendChar(.{ .grapheme = "", .width = 1 });
        return;
    }

    self.read_buf = try self.arena_alloc.alloc(u8, size);
    var reader = file.reader(self.read_buf);
    var i: usize = 0;

    while (try reader.interface.takeDelimiter('\n')) |line| {
        defer i += 1;

        if (i > 0) {
            try self.addRow(0);
        }

        var iter = vx.unicode.graphemeIterator(line);
        while (iter.next()) |g| {
            try self.curRow().appendChar(.{
                .grapheme = g.bytes(line),
                .width = 1,
            });
        }
    }
}

/// Appends a new row after the last.
pub fn addRow(self: *Buffer, offset: usize) !void {
    if (self.rows.items.len > 0) {
        self.row += 1;
    }
    self.col = 0;

    try self.rows.append(self.arena_alloc, try .init(
        self.arena_alloc,
        @intCast(offset),
    ));
}

/// Adds a new row at `index`
pub fn addRowAt(self: *Buffer, index: usize, offset: i32) !void {
    try self.rows.insert(
        self.arena_alloc,
        index,
        try .init(self.arena_alloc, @intCast(offset)),
    );
}

/// Splits the current row and moves the value after the cursor to
/// a new line
pub fn splitRow(self: *Buffer) !void {
    const cur_row: *Row = self.curRow();
    const col: u32 = @intCast(self.col);
    const after_cursor: []Char = cur_row.getValue()[col..];

    // make a copy of the value after the cursor that needs to be moved
    // to the next line.
    const after_cursor_cp: []Char = try self.arena_alloc.alloc(
        Char,
        after_cursor.len,
    );
    defer self.arena_alloc.free(after_cursor_cp);
    @memmove(after_cursor_cp, after_cursor);

    // Remove the value after the cursor from the current line.
    for (col..cur_row.len()) |_| {
        _ = cur_row.deleteCharAt(col);
    }

    // Add new below
    self.row += 1;
    self.col = 0;
    try self.addRowAt(@intCast(self.row), 0);

    // Append copied value to the new line.
    for (after_cursor_cp) |char| {
        try self.curRow().appendChar(char);
    }
}

pub fn getString(self: *Buffer, rows: ?[]*Row) ![]u8 {
    var items = self.rows.items;
    if (rows != null) {
        items = rows.?;
    }

    const total = self.totalByteLen(items) - 1;
    var buffer = try self.arena_alloc.alloc(u8, total);
    var index: usize = 0;
    var row_index = index;

    for (items) |row| {
        defer row_index += 1;

        for (row.value.items) |ch| {
            std.mem.copyForwards(
                u8,
                buffer[index .. index + ch.grapheme.len],
                ch.grapheme,
            );
            index += ch.grapheme.len;
        }

        if (row_index + 1 < self.rows.items.len) {
            buffer[index] = '\n';
            index += 1;
        }
    }

    return buffer;
}

fn totalByteLen(self: *Buffer, rows: []const *Row) usize {
    _ = self;
    var total: usize = 0;

    for (rows) |row| {
        for (row.value.items) |ch| {
            total += ch.grapheme.len;
        }
        total += 1;
    }

    return total;
}

pub fn getHash(self: *Buffer) !u64 {
    const str = try self.getString(null);
    return fastHash(str);
}

/// Returns a reference to the current row.
pub fn curRow(self: *Buffer) *Row {
    return self.rows.items[@intCast(self.row)];
}

pub fn numRows(self: Buffer) usize {
    return self.rows.items.len;
}

pub fn cursorPos(self: Buffer) CursorPos {
    return self.cursor_pos;
}

pub fn updateCursorPos(self: *Buffer) void {
    self.cursor_pos.col = self.col;
    self.cursor_pos.row = self.row;
    self.cursor_pos.row_offset = self.row_offset;
}
pub fn shrinkAndFree(self: *Buffer) void {
    self.rows.shrinkAndFree(self.arena_alloc, self.numRows());
}

pub fn deinit(self: *Buffer) void {
    self.history.deinit();
    self.arena_alloc.free(self.read_buf);
    self.arena.deinit();
}

pub fn hashStr(str: []const u8) [Sha256.digest_length]u8 {
    var sha256: Sha256 = .init(.{});
    sha256.update(str);
    return sha256.finalResult();
}

pub fn fastHash(str: []const u8) u64 {
    return std.hash.Wyhash.hash(0, str);
}
