const TextArea = @This();

const std = @import("std");
const vx = @import("vaxis");
const Key = vx.Key;

const dmp = @import("diffmatchpatch");

pub const App = @import("../../../App.zig");
pub const Buffer = @import("Buffer.zig");
pub const Vim = @import("Vim.zig");

alloc: std.mem.Allocator,

app: *App,

/// List of available text buffers
buffers: std.ArrayList(*Buffer),

/// Current buffer
buffer: usize,

win: ?vx.Window = null,

is_focucsed: bool = false,

scroll_view: ?*vx.widgets.ScrollView = null,

/// The textarea's width
width: u16 = 0,

/// The textarea's height
height: u16 = 0,

/// Holds the instance for all the vim functionality
vim: *Vim,

use_virtual_cursor: bool = false,

/// Holds the previous text value of the current buffer.
/// Currently used for storing the value before entering any vim mode
/// that alters the text, so that we can create a reliable history.
prev_value: []const u8 = "",

pub const Event = union(enum) {
    key_press: Key,
};

pub const Character = struct {
    grapheme: []const u8 = " ",
    width: u8 = 1,
    is_selected: bool = false,
};

pub fn init(alloc: std.mem.Allocator, app: *App) !TextArea {
    const self: TextArea = .{
        .alloc = alloc,
        .app = app,
        .buffers = .empty,
        .buffer = 0,
        .vim = try .init(alloc),
    };

    return self;
}

pub fn update(self: *TextArea, event: Event) !void {
    if (!self.is_focucsed) {
        return;
    }

    if (self.vim.enabled) {
        try self.vim.update(event, self);
    } else {
        switch (event) {
            .key_press => |key| {
                if (key.matches(Key.enter, .{})) {
                    self.addNewLine();
                } else if (key.matches(Key.left, .{})) {
                    self.characterLeft();
                } else if (key.matches(Key.right, .{})) {
                    self.characterRight();
                } else if (key.matches(Key.up, .{})) {
                    self.cursorUp();
                } else if (key.matches(Key.down, .{})) {
                    self.cursorDown();
                }

                if (key.text) |text| {
                    try self.insertSliceAtCursor(text);
                }
            },
        }
    }
}

pub fn draw(self: *TextArea) !void {
    var style: vx.Cell.Style = .{};

    if (self.win == null or
        self.scroll_view == null or
        !self.hasBuffers())
    {
        return;
    }

    const win: vx.Window = self.win.?;
    const view: *vx.widgets.ScrollView = self.scroll_view.?;

    self.width = win.width;
    self.height = win.height;

    const buf: *Buffer = self.curBuf();
    buf.updateCursorPos();
    var i: usize = 0;

    for (buf.rows.items) |row| {
        defer i += 1;

        const row_index: u16 = @intCast(i);
        var col: u16 = 0;

        for (row.getValue()) |*char| {
            var iter = vx.unicode.graphemeIterator(char.grapheme);
            while (iter.next()) |grapheme| {
                const g = grapheme.bytes(char.grapheme);
                char.width = @intCast(win.gwidth(g));

                if (self.use_virtual_cursor and col == buf.col and
                    (buf.row == row_index or self.getTermRow() == row_index))
                {
                    style.bg = .{ .rgb = self.vim.mode.bgColor() };
                    style.fg = .{ .rgb = self.vim.mode.fgColor() };
                }

                view.writeCell(win, col, row_index, .{
                    .char = .{ .grapheme = g, .width = char.width },
                    .style = style,
                });
                style = .{};
            }
            col += 1;
        }

        // Do cursor stuff only on current row
        if (buf.row == row_index or self.getTermRow() == row_index) {
            if (self.use_virtual_cursor or !self.is_focucsed) {
                win.hideCursor();
            } else {
                win.showCursor(
                    @intCast(buf.col),
                    @intCast(self.getTermRow()),
                );
            }
        }
    }
}

pub fn newBuf(self: *TextArea, path: []const u8) !void {
    try self.buffers.append(self.alloc, try .init(self.alloc));
    var buf: *Buffer = self.buffers.getLast();
    // use the arena from the buffer since the history is tied to it.
    buf.history = try .init(buf.arena_alloc);
    buf.setPath(path);
    buf.setIndex(self.numBufs() - 1);

    try buf.setContentFromFile(path);

    self.buffer = self.numBufs() + 1;
    self.goToTop();
}

pub fn newScratchBuf(self: *TextArea, content: ?[]const u8) !void {
    try self.buffers.append(self.alloc, try .init(self.alloc));
    const buf: *Buffer = self.buffers.getLast();
    const value = if (content != null) content.? else "";
    try buf.curRow().insertSliceAtCursor(value);

    self.buffer = self.numBufs() + 1;
}

/// Opens a buffer with the given `path`.
/// If no buffer is found it attempts to create a new buffer with `path`.
pub fn openBuf(self: *TextArea, path: []const u8) !void {
    if (self.findBuf(path)) |buffer| {
        self.buffer = buffer.index;
    } else {
        try self.newBuf(path);
    }
}

/// Attempts to find a buffer with the given `path`.
/// If none could be found it returns `null`.
pub fn findBuf(self: TextArea, path: []const u8) ?*Buffer {
    for (self.buffers.items) |buffer| {
        if (std.mem.eql(u8, buffer.path, path)) {
            return buffer;
        }
    }
    return null;
}

pub fn numBufs(self: TextArea) usize {
    return self.buffers.items.len;
}

pub fn hasBuffers(self: TextArea) bool {
    return self.numBufs() > 0;
}

/// Returns the current buffer.
pub fn curBuf(self: TextArea) *Buffer {
    var buf_index = self.buffer;
    if (buf_index > self.numBufs()) {
        buf_index = self.numBufs() - 1;
    }
    return self.buffers.items[buf_index];
}

/// Enables vim motions.
pub fn enableVimMode(self: *TextArea) !void {
    self.vim.enable();
}

/// Insert text at the cursor position
pub fn insertSliceAtCursor(self: *TextArea, data: []const u8) !void {
    const buf: *Buffer = self.curBuf();
    var cur_row = buf.curRow();
    cur_row.col = buf.col;

    var iter = vx.unicode.graphemeIterator(data);

    // line wrap
    if (cur_row.len() > self.width) {
        try buf.addRowAt(@intCast(buf.row + 1), cur_row.offset + 1);
        self.beginLine(false);
        self.cursorDown();
    }

    while (iter.next()) |text| {
        try buf.curRow().insertSliceAtCursor(text.bytes(data));
        self.characterRight();
    }
}

/// Remove the character at `index`.
/// If vim is enabled the cursor will be moved one character to the left.
pub fn deleteCharAt(self: *TextArea, index: i32) void {
    const buf: *Buffer = self.curBuf();
    var cur_row: *Buffer.Row = buf.curRow();

    // Skip if we're at the start of an empty line or
    // index is out of bound
    if (cur_row.len() <= 0 or index > cur_row.len()) {
        return;
    }

    _ = cur_row.deleteCharAt(@intCast(index));
    cur_row.shrinkAndFree();
    buf.col = index;
}

/// Removes the character at the cursor position.
pub fn deleteCurChar(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const row_len: i16 = @intCast(buf.curRow().len());
    self.deleteCharAt(@intCast(buf.col));
    if (buf.col == row_len - 1) {
        self.characterLeft();
    }
}

/// Removes all remaining characters of the current row
/// starting from the cursor's position.
pub fn deleteAfterCursor(self: *TextArea) !void {
    const buf: *Buffer = self.curBuf();
    var cur_row: *Buffer.Row = buf.curRow();
    const col: u32 = @intCast(buf.col);
    const after_cursor: []Character = cur_row.getValue()[col..];

    // make a copy of the value after the cursor that needs to be moved
    // to the next line.
    const after_cursor_cp = try self.alloc.alloc(Character, after_cursor.len);
    defer self.alloc.free(after_cursor_cp);
    @memmove(after_cursor_cp, after_cursor);

    // Remove the value after the cursor from the current line.
    for (col..cur_row.len()) |_| {
        _ = cur_row.deleteCharAt(col);
    }
}

/// Add a new line.
/// If the cursor is not at the end of a line, the line gets automatically
/// split, moving the text after (including) the cursor onto the new line.
pub fn addNewLine(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const new_row: i32 = buf.row + 1;

    if (new_row > buf.rows.items.len or
        buf.col > buf.curRow().len())
    {
        buf.addRow(0) catch return;
    } else {
        buf.splitRow() catch return;
    }

    self.repositionView();
}

/// Add an empty line below the current one.
pub fn addLineBelow(self: *TextArea) !void {
    const buf: *Buffer = self.curBuf();
    const new_row: u32 = @intCast(buf.row + 1);
    if (new_row <= buf.rows.items.len) {
        try buf.addRowAt(new_row, 0);
        self.cursorDown();
    }
}

/// Add an empty line above the current one.
pub fn addLineAbove(self: *TextArea) !void {
    const buf: *Buffer = self.curBuf();
    try buf.addRowAt(@intCast(buf.row), 0);
    buf.col = 0;
}

/// Deletes the next line and moves its content to the current line.
pub fn joinLine(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const next_index: i32 = buf.row + 1;

    if (next_index >= buf.rows.items.len) {
        return;
    }

    var next_row: *Buffer.Row = buf.rows.items[@intCast(next_index)];
    const next_row_val: []Character = next_row.getValue();
    const line_len: u16 = @intCast(buf.curRow().len());

    if (next_row_val.len == 0) {
        self.deleteLineAt(next_index);
        buf.shrinkAndFree();
        return;
    }

    self.newHistoryEntry();

    // Prepend whitespace when line to join doesn't start with one
    // and is not empty
    if (!std.mem.eql(u8, next_row_val[0].grapheme, " ") and
        line_len > 0)
    {
        buf.curRow().appendChar(.{}) catch return;
    }

    for (next_row_val) |val| {
        buf.curRow().appendChar(val) catch return;
    }

    self.moveCursorTo(buf.row, line_len);
    self.deleteLineAt(next_index);
    buf.shrinkAndFree();
    self.updateHistoryEntry() catch return;
}

/// Move the cursor to the start of the line.
/// If `non_white` is true, the cursor moves to the first non-white
/// character of the line.
pub fn beginLine(self: *TextArea, non_white: bool) void {
    const buf: *Buffer = self.curBuf();
    buf.col = 0;
    buf.last_col = 0;

    if (non_white) {
        var i: u16 = 0;
        for (buf.curRow().getValue()) |char| {
            defer i += 1;

            if (!std.mem.eql(u8, char.grapheme, " ")) {
                buf.col = i;
                return;
            }
        }
    }
}

/// Moves the cursor to the end of the current line.
pub fn lineEnd(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const row_len: u16 = @intCast(buf.curRow().len());
    buf.col = if (self.vim.enabled and row_len > 0)
        row_len - 1
    else
        row_len;

    buf.last_col = 0;
}

/// Deletes the live at the given `index` and frees its data.
pub fn deleteLineAt(self: *TextArea, index: i32) void {
    const buf: *Buffer = self.curBuf();

    if (buf.rows.items.len == 0) {
        return;
    }

    // delete at first and only line, just empty that line.
    if (buf.row == 0 and buf.numRows() == 1) {
        buf.curRow().shrinkAndFree();
        self.beginLine(false);
    } else {
        // remove row and free memory of removed row
        const row: *Buffer.Row = buf.rows.orderedRemove(@intCast(index));
        row.deinit();
        //self.alloc.destroy(row);
    }

    // if we're deleting the last row, move to the last available line.
    if (buf.row >= buf.numRows()) {
        buf.row = @intCast(buf.rows.items.len - 1);
    }

    // if the current row is empty move cursor to the start of the line.
    if (buf.curRow().len() == 0) {
        self.beginLine(false);
    }

    if (buf.col > buf.curRow().len()) {
        self.lineEnd();
    }
}

/// Deletes the current line.
/// If `keep_cur_pos` is false the cursor is moved to the
/// beginning of the line.
pub fn deleteCurLine(self: *TextArea, keep_cur_pos: bool) void {
    const buf: *Buffer = self.curBuf();
    self.deleteLineAt(buf.row);

    if (!keep_cur_pos) {
        self.beginLine(false);
    }
}

/// Deletes `n` lines starting from the current row.
pub fn deleteNLines(self: *TextArea, n: usize) void {
    for (0..n) |_| {
        self.deleteCurLine(true);
    }
}

/// Repositions the view to the cursor position, ensuring it's always
/// in the viewport.
pub fn repositionView(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();

    if (self.scroll_view) |view| {
        const min = view.scroll.y;
        const max = min + self.height - 1;
        const row: u32 = @intCast(buf.row);

        if (buf.row < min) {
            view.scroll.y -= min - row;
        } else if (buf.row > max) {
            view.scroll.y += row - max;
        }
    }
}

/// Moves the cursor one line up.
pub fn cursorUp(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();

    if (buf.row > 0) {
        buf.row -= 1;
    }

    if (buf.row - 1 > 0) {
        const prev_row = buf.rows.items[@intCast(buf.row - 1)];
        // save last column position
        if ((prev_row.len() == 0 and
            buf.col > 0 and
            buf.col > buf.last_col and
            buf.last_col >= prev_row.len()) or
            buf.last_col == 0)
        {
            buf.last_col = buf.col;
        }
    }

    self.setCursorRow(@intCast(buf.row));

    if (buf.last_col > 0) {
        self.setCursorCol(buf.last_col);
    }

    self.repositionView();
}

/// Moves the cursor one line down.
pub fn cursorDown(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();

    if (buf.row + 1 < buf.numRows()) {
        buf.row += 1;
    }

    if (buf.row + 1 < buf.numRows()) {
        const next_row = buf.rows.items[@intCast(buf.row + 1)];
        // save last column position
        if ((next_row.len() == 0 and
            buf.col > 0 and
            buf.col > buf.last_col and
            buf.last_col >= next_row.len()) or buf.last_col == 0)
        {
            buf.last_col = buf.col;
        }
    }

    self.setCursorRow(@intCast(buf.row));

    if (buf.last_col > 0) {
        self.setCursorCol(buf.last_col);
    }

    self.repositionView();
}

/// Moves the cursor one character to the left.
pub fn characterLeft(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();

    if (buf.col <= 0) {
        return;
    }

    const char: Character = buf.curRow().getValue()[@intCast(buf.col - 1)];
    self.setCursorCol(buf.col - char.width);
    buf.last_col = 0;
}

/// Moves the cursor one character to the right.
pub fn characterRight(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const row_val: []Character = buf.curRow().getValue();

    var row_len = row_val.len;
    // In vim mode we don't allow the cursor to move past the last character
    // unless we're in insert mode so the length of the row should always
    // be one character less.
    if (self.vim.enabled and self.vim.mode == .normal and row_len > 0) {
        row_len -= 1;
    }

    if (buf.col >= row_len) {
        return;
    }

    const char: Character = row_val[@intCast(buf.col)];
    self.setCursorCol(buf.col + char.width);
    buf.last_col = 0;
}

/// WordRight moves the cursor to the start of the next word.
/// If the cursor is at the end of the row it moves one row down.
/// Skips any non-letter characters that follow. (<-- this is actually yet
/// to be implemted)
pub fn wordRight(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const row: *Buffer.Row = buf.curRow();

    if (self.tryNextLine()) {
        return;
    }

    for (@intCast(buf.col)..row.getValue().len) |i| {
        if (self.tryNextLine()) {
            return;
        }

        const char: Character = row.getValue()[i];

        self.characterRight();

        if (charIsSpace(char.grapheme)) {
            return;
        }
    }
}

/// WordRightEnd moves the cursor to the end of the next word.
/// If the cursor is at the end of the row it moves one row down.
/// Skips any non-letter characters that follow.
pub fn wordRightEnd(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const row: *Buffer.Row = buf.curRow();
    const row_val: []Character = row.getValue();

    if (self.tryNextLine()) {
        return;
    }

    for (@intCast(buf.col + 1)..row.len()) |_| {
        self.characterRight();

        var char: Character = row_val[@intCast(buf.col)];
        while (buf.col < row.len()) {
            char = row_val[@intCast(buf.col)];
            if (!charIsSpace(char.grapheme)) {
                break;
            }
            self.characterRight();
        }

        if (buf.col + 1 >= row.len()) {
            break;
        }

        const next_char = row_val[@intCast(buf.col + 1)];
        if (charIsSpace(next_char.grapheme)) {
            return;
        }
    }
}

/// Moves the cursor to the beginning of the previous word.
/// If the cursor is at the start of the row it moves one row up.
pub fn wordLeft(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const row: *Buffer.Row = buf.curRow();

    // Move to the row above if we reached the start of the line.
    if (buf.col <= 0 and buf.row > 0) {
        self.cursorUp();
        self.lineEnd();
        self.wordLeft();
        return;
    }

    var i: usize = @intCast(buf.col - 1);
    while (i > 0) {
        i -= 1;
        const char: Character = row.getValue()[i];

        self.characterLeft();
        if (charIsSpace(char.grapheme)) {
            return;
        }
    }

    if (i == 0) {
        self.beginLine(false);
    }
}

/// Moves the cursor to the end of the previous word.
/// If the cursor is at the start of the row it moves one row up.
pub fn wordLeftEnd(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const row: *Buffer.Row = buf.curRow();

    self.wordRight();

    for (0..@intCast(buf.col)) |i| {
        const char: Character = row.getValue()[i];
        if (charIsSpace(char.grapheme)) {
            self.characterRight();
            return;
        }
        self.characterLeft();
    }
}

/// Helper to determine if the cursor is at the end of a line and moves
/// to the beginning of the next line if true.
fn tryNextLine(self: *TextArea) bool {
    const buf: *Buffer = self.curBuf();
    const row_len: isize = @intCast(buf.curRow().len());
    const num_rows: i32 = @intCast(buf.numRows());

    if (buf.col >= row_len - 1 and buf.row < num_rows - 1) {
        self.cursorDown();
        self.beginLine(true);
        return true;
    }
    return false;
}

/// Moves the view up half the viewport size centering the cursor.
/// If the buffer's content is larger than the viewport.
pub fn halfPageUp(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const half_height: i16 = @intCast(self.height / 2);
    const row: i32 = @intCast(buf.row);
    const new_row = std.math.clamp(row - half_height, 0, buf.numRows() - 1);

    if (self.scroll_view) |view| {
        const min: i32 = @intCast(view.scroll.y);

        if (new_row <= min) {
            if (min > half_height) {
                view.scroll.y -= @intCast(half_height);
            } else {
                self.goToTop();
            }
        }

        if (min == 0) {
            self.goToTop();
        } else {
            self.moveCursorTo(@intCast(new_row), buf.col);
        }
    }
}

/// Moves the view down half the viewport size centering the cursor.
/// If the buffer's content is larger than the viewport.
pub fn halfPageDown(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const half_height: u16 = self.height / 2;
    const new_row = std.math.clamp(buf.row + half_height, 0, buf.numRows() - 1);

    if (self.scroll_view) |view| {
        const min = view.scroll.y;
        const max = min + self.height - 1;

        if (new_row >= max) {
            view.scroll.y += half_height;
        }

        if (max == buf.numRows() - 1) {
            self.goToBottom();
        } else {
            self.moveCursorTo(@intCast(new_row), buf.col);
        }
    }
}

/// Moves the cursor to the top of the text and repositions the view.
pub fn goToTop(self: *TextArea) void {
    self.moveCursorTo(0, self.curBuf().col);
    self.repositionView();
}

/// Moves the cursor to the bottom of the text and repositions the view.
pub fn goToBottom(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const num_rows: i32 = @intCast(buf.numRows());
    var last_row: i32 = self.height;

    if (num_rows < self.height) {
        last_row = num_rows;
    }

    self.moveCursorTo(num_rows, buf.col);
    self.repositionView();
}

/// Don't use...still buggy.
pub fn centreView(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const half_height: u16 = self.height / 2;

    if (self.scroll_view == null or buf.cursor_row == half_height) {
        return;
    }

    const view: *vx.widgets.ScrollView = self.scroll_view.?;
    const diff: i32 = self.height - buf.cursor_row - half_height;

    if (buf.cursor_row >= half_height) {
        const d: usize = @intCast(@abs(diff));
        view.scroll.y += d;

        self.setCursorRow(@intCast(buf.row - d));
    } else {
        const cur_y: i32 = @intCast(view.scroll.y);
        const d: usize = @intCast(diff);

        if (cur_y - diff > 0) {
            view.scroll.y -= d;
        }

        self.setCursorRow(@intCast(buf.row + d));
    }

    buf.cursor_row = half_height;
}

/// Moves the cursor to the given `row` and `col`.
pub fn moveCursorTo(self: *TextArea, row: i32, col: i32) void {
    self.setCursorRow(row);
    self.setCursorCol(col);
}

/// Moves the cursor to the given column.
/// If `reset_last_col` is true it resets the last cursor position
/// `Buffer.last_col`.
pub fn setCursorCol(self: *TextArea, col: i32) void {
    const buf: *Buffer = self.curBuf();
    var row_len: i32 = @intCast(buf.curRow().len());

    if (row_len > 0 and self.vim.mode != .insert) {
        row_len -= 1;
    }
    buf.col = std.math.clamp(col, 0, row_len);
}

/// Moves the cursor to the given row.
pub fn setCursorRow(self: *TextArea, row: i32) void {
    const buf: *Buffer = self.curBuf();
    const num_rows = buf.numRows();
    const clamped = std.math.clamp(row, 0, num_rows);

    // ensure the cursor does not go further as it should
    var max_cur_row = buf.numRows() - 1;
    if (buf.numRows() > self.height and self.height > 0) {
        max_cur_row = self.height - 1;
    }

    buf.row = @intCast(std.math.clamp(clamped, 0, num_rows - 1));
}

/// Creates a new history entry for the current Buffer
/// saving the correct undo cursor position and current textarea content.
pub fn newHistoryEntry(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    buf.history.newTmpEntry(buf.cursorPos());
    self.prev_value = buf.getString(null) catch return;
}

/// Updates the history entry saving the undo/redo
/// patch, the current cursor position and the hash of the buffer content
pub fn updateHistoryEntry(self: *TextArea) !void {
    const buf: *Buffer = self.curBuf();
    const cur_buf_val = try buf.getString(buf.rows.items);

    var redo_patch = try buf.history.makePatch(self.prev_value, cur_buf_val);
    defer redo_patch.deinit();

    var undo_patch = try buf.history.makePatch(cur_buf_val, self.prev_value);
    defer undo_patch.deinit();

    try buf.history.updateEntry(
        redo_patch,
        undo_patch,
        buf.cursor_pos,
        try buf.getHash(),
    );
    self.prev_value = buf.getString(null) catch return;
}

/// Sets the buffer content to the previous history entry.
pub fn undo(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const patch = buf.history.undo() catch return;
    const patched = buf.history.dmp.patchApply(
        patch.patch,
        buf.getString(null) catch return,
    ) catch return;

    buf.setContentFromStr(patched.@"0") catch return;
    self.moveCursorTo(patch.cursor_pos.row, patch.cursor_pos.col);
    self.repositionView();
}

// Sets the buffer content to the next history entry.
pub fn redo(self: *TextArea) void {
    const buf: *Buffer = self.curBuf();
    const patch = buf.history.redo() catch return;
    const patched = buf.history.dmp.patchApply(
        patch.patch,
        buf.getString(null) catch return,
    ) catch return;

    buf.setContentFromStr(patched.@"0") catch return;
    self.moveCursorTo(patch.cursor_pos.row, patch.cursor_pos.col);
    self.repositionView();
}

/// Get the terminal row for the current cursor position.
pub fn getTermRow(self: TextArea) usize {
    const buf: *Buffer = self.curBuf();
    var row: usize = 0;
    if (self.scroll_view) |view| {
        row = @intCast(buf.row);
        return @intCast(row - view.scroll.y);
    }
    return row;
}

pub fn deinit(self: *TextArea) void {
    for (self.buffers.items) |buffer| {
        buffer.deinit();
        self.alloc.destroy(buffer);
    }
    self.buffers.deinit(self.alloc);
    self.vim.deinit();
    self.alloc.destroy(self.vim);
}

fn charIsSpace(char: []const u8) bool {
    const utf8 = std.unicode.Utf8View.init(char) catch return false;
    var iter = utf8.iterator();

    while (iter.nextCodepoint()) |cp| {
        if (cp == 32) {
            return true;
        }
    }

    return false;
}
