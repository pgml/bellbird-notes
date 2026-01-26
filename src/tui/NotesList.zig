const NotesList = @This();

const std = @import("std");
const vx = @import("vaxis");

const App = @import("../App.zig");
const Buffer = @import("widgets/TextArea/TextArea.zig").Buffer;
const Cell = @import("layout/Cell.zig");
const fs = @import("../fs.zig");
const ListItem = @import("ListItem.zig");
const theme = @import("layout/theme.zig");
const Icon = theme.Icon;
const utils = @import("../utils.zig");

alloc: std.mem.Allocator,

app: *App,

/// The layout cell/column
cell: *Cell,

/// default width of the directory tree column.
default_width: u16 = 30,

/// default heigh of the directory tree column.
default_height: u16 = 0,

/// default height of a single directory item
default_item_height: u16 = 1,

/// The list index of the selected tree item/directory.
selected_index: isize = 0,

/// A flat list of all visible directories.
note_items: std.ArrayList(*NoteItem) = .empty,

/// The directory path of the currently displayed notes.
/// This path might not match the directory that is selected in the
/// directory tree since we don't automatically display a directory's
/// content on a selection change
current_path: []const u8 = "",

/// Contains dirty buffers of the current notes list
DirtyBuffers: std.ArrayList(Buffer) = .empty,

/// Buffers holds all the open buffers
Buffers: std.ArrayList(*Buffer) = .empty,

scroll_view: vx.widgets.ScrollView,

win: ?vx.Window = null,

pub const NoteItem = struct {
    /// General list data
    data: ListItem,

    cell: *Cell,

    /// Stores the rendered toggle arrow icon
    icon: []const u8 = "",

    // Stores the rendered toggle arrow icon
    toggle_arrow: []const u8 = "",

    pub fn deinit(self: *NoteItem, alloc: std.mem.Allocator) void {
        alloc.free(self.data.name);
        alloc.free(self.data.path);
        alloc.destroy(self.cell);
    }
};

pub fn init(alloc: std.mem.Allocator, title: []const u8, app: *App) !*NotesList {
    const self = try alloc.create(NotesList);

    self.* = .{
        .alloc = alloc,
        .app = app,
        .cell = try .init(alloc),
        .scroll_view = .{},
    };

    self.cell.setWidth(self.default_width);
    self.cell.title = title;

    return self;
}

pub fn update(self: *NotesList, event: App.Event) !void {
    switch (event) {
        .key_press => |key| {
            if (!self.cell.isFocused()) {
                return;
            }

            // this is all temporary
            switch (key.codepoint) {
                'j' => self.selected_index += 1,
                'k' => self.selected_index -= 1,
                vx.Key.enter => {
                    if (self.selectedNote()) |note| {
                        try self.app.editor.openBuf(note.data.path);
                        try self.app.config.meta_infos.setValue(
                            .last_open_note,
                            note.data.path,
                        );
                        try self.app.config.meta_infos.write();
                    }
                },
                else => {},
            }
        },
        else => {},
    }

    var list_len = self.note_items.items.len;
    if (list_len > 0) {
        list_len -= 1;
    }
    self.selected_index = std.math.clamp(self.selected_index, 0, list_len);
}

pub fn draw(self: *NotesList, win: vx.Window) void {
    const opts = self.cell.getChild();
    const child_win = win.child(opts);

    if (self.win == null) {
        self.win = child_win;
    }

    var index: isize = 0;
    for (self.note_items.items) |item| {
        item.cell.setHeight(self.default_item_height);
        var child_opts = item.cell.getChild();
        // reset border for each tree item
        child_opts.border = .{};

        _ = child_win.child(child_opts);

        var style: vx.Cell.Style = .{};
        if (index == self.selected_index) {
            style.bg = .{ .rgb = .{ 66, 75, 93 } };
        }

        const row: u16 = @intCast(index + item.cell.height - 1);
        self.writeLine(item, row, self.cell.width, style);
        index += 1;
        item.data.index = @intCast(index);
    }
}

pub fn drawHeader(self: NotesList, win: vx.Window, col: u16) void {
    Cell.drawHeader(win, self.cell.title, col, self.cell.isFocused());
}

fn writeLine(self: NotesList, item: *NoteItem, row: u16, width: u16, style: vx.Cell.Style) void {
    var col: u16 = 0;

    if (self.win) |win| {
        Cell.write(win, &col, row, " ", style);
        Cell.write(win, &col, row, Icon.getNerd(.note), style);
        Cell.write(win, &col, row, " ", style);
        Cell.write(win, &col, row, item.data.name, style);

        // pad the rest of the line to make the selection expand to the whole row
        while (col < width) {
            Cell.write(win, &col, row, " ", style);
        }
    }
}

pub fn restore(self: *NotesList) !void {
    const meta = self.app.config.meta_infos;
    try self.getNotes(meta.last_directory);
}

pub fn getNotes(self: *NotesList, path: []const u8) !void {
    if (utils.strEql(path, "")) {
        return;
    }

    var arena = std.heap.ArenaAllocator.init(self.alloc);
    defer arena.deinit();

    self.alloc.free(self.current_path);
    self.current_path = try self.alloc.dupe(u8, path);

    const tmp_note_entries = try fs.Notes.list(
        arena.allocator(),
        self.current_path,
    );

    self.freeNotes();
    self.note_items = .empty;

    for (tmp_note_entries) |entry| {
        const note_item = try self.createNoteItem(entry);
        try self.note_items.append(self.alloc, note_item);
    }
}

fn createNoteItem(self: *NotesList, item: fs.Notes.Entry) !*NoteItem {
    const note_item = try self.alloc.create(NoteItem);

    const cell: *Cell = try .init(self.alloc);
    cell.setHeight(self.default_item_height);

    note_item.* = .{
        .data = .{
            .index = 0,
            .name = try self.alloc.dupe(u8, item.name),
            .path = try self.alloc.dupe(u8, item.path),
        },
        .cell = cell,
    };

    return note_item;
}

fn getListItem(self: NotesList, index: usize) ?*NoteItem {
    if (index >= self.note_items.items.len) {
        return null;
    }
    return self.note_items.items[index];
}

fn selectedNote(self: NotesList) ?*NoteItem {
    if (self.getListItem(@intCast(self.selected_index))) |note| {
        return note;
    }
    return null;
}

pub fn focus(self: *NotesList) void {
    self.cell.focus();
}

pub fn blur(self: *NotesList) void {
    self.cell.blur();
}

pub fn setFocus(self: *NotesList, f: bool) void {
    self.cell.setFocus(f);
}

fn freeNotes(self: *NotesList) void {
    for (self.note_items.items) |notes| {
        notes.deinit(self.alloc);
        self.alloc.destroy(notes);
    }
    self.note_items.deinit(self.alloc);
}

pub fn deinit(self: *NotesList) void {
    self.freeNotes();
    self.alloc.free(self.current_path);
}
