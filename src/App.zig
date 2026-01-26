const App = @This();

const std = @import("std");
const vaxis = @import("vaxis");

const Config = @import("Config.zig");
const DirectoryTree = @import("tui/DirectoryTree.zig");
const Editor = @import("tui/Editor.zig");
const NotesList = @import("tui/NotesList.zig");
const StatusBar = @import("tui/StatusBar.zig");
const log = @import("log.zig");

alloc: std.mem.Allocator,

config: *Config,

should_quit: bool,

tty: vaxis.tty.PosixTty,

vx: vaxis.Vaxis,

notes_list: *NotesList,

directory_tree: *DirectoryTree,

editor: *Editor,

status_bar: *StatusBar,

current_column: u16 = 1,

last_column: u16 = 0,

mode: Editor.TextArea.Vim.Mode = .normal,

win: vaxis.Window = undefined,

pub const Event = union(enum) {
    key_press: vaxis.Key,
    key_release: vaxis.Key,
    winsize: vaxis.Winsize,
};

pub fn init(alloc: std.mem.Allocator) !App {
    var buffer: [1024]u8 = undefined;

    return .{
        .alloc = alloc,
        .config = try .init(alloc),
        .tty = try vaxis.Tty.init(&buffer),
        .vx = try vaxis.init(alloc, .{}),
        .should_quit = false,
        .notes_list = undefined,
        .directory_tree = undefined,
        .editor = undefined,
        .status_bar = undefined,
    };
}

pub fn run(self: *App) !void {
    const writer: *std.Io.Writer = self.tty.writer();

    var loop: vaxis.Loop(Event) = .{
        .vaxis = &self.vx,
        .tty = &self.tty,
    };
    try loop.init();

    try loop.start();
    defer loop.stop();

    try self.vx.enterAltScreen(writer);

    self.directory_tree = try .init(self.alloc, "Folders", self);
    self.notes_list = try .init(self.alloc, "Notes", self);
    self.editor = try .init(self.alloc, "Editor", self);
    self.status_bar = try .init(self.alloc, self);

    try writer.flush();
    try self.vx.queryTerminal(self.tty.writer(), 1 * std.time.ns_per_s);

    try self.directory_tree.run();
    try self.restoreState();

    while (!self.should_quit) {
        const event: Event = loop.nextEvent();
        try self.update(event);

        switch (event) {
            .key_press => |key| {
                _ = key;
            },
            .winsize => |ws| {
                try self.vx.resize(self.alloc, self.tty.writer(), ws);
            },
            else => {},
        }

        try self.draw();

        // Render the screen
        try self.vx.render(writer);
        try writer.flush();
    }
}

pub fn update(self: *App, event: Event) !void {
    try self.directory_tree.update(event);
    try self.notes_list.update(event);
    try self.editor.update(event);
    try self.status_bar.update(event);

    switch (event) {
        .key_press => |key| {
            // TEMPORARY .. use input map for this
            if (key.matches('l', .{ .ctrl = true })) {
                self.focusNextColumn(true);
                self.config.meta_infos.current_column = self.current_column;
                try self.config.meta_infos.setValue(
                    .current_column,
                    self.current_column,
                );
                try self.config.meta_infos.write();
            }

            if (key.matches('h', .{ .ctrl = true })) {
                self.focusPrevColumn(true);
                self.config.meta_infos.current_column = self.current_column;
                try self.config.meta_infos.setValue(
                    .current_column,
                    self.current_column,
                );
                try self.config.meta_infos.write();
            }

            if (key.matches(':', .{})) {
                self.last_column = self.current_column;
                // unfocus all colums by setting index to 0
                self.focusColumn(0);
                self.status_bar.focus();
            }

            if (key.matches(vaxis.Key.escape, .{})) {
                if (self.mode == .command) {
                    self.focusColumn(self.last_column);
                    self.status_bar.blur();
                }
            }
        },
        .winsize => |ws| {
            try self.vx.resize(self.alloc, self.tty.writer(), ws);
        },
        else => {},
    }
}

pub fn draw(self: *App) !void {
    var win: vaxis.Window = self.vx.window();
    win.clear();
    try self.initComponents(win);
    self.win = win;
}

fn restoreState(self: *App) !void {
    try self.config.loadMetaInfo();
    try self.directory_tree.restore();
    try self.notes_list.restore();
    try self.editor.restore();
    self.focusColumn(@intCast(self.config.meta_infos.current_column));
}

fn initComponents(self: *App, win: vaxis.Window) !void {
    const sb_height = self.status_bar.cell.height;

    self.directory_tree.cell.setHeight(win.height - sb_height);
    self.directory_tree.cell.setOffsetY(0);
    self.directory_tree.draw(win);
    self.directory_tree.drawHeader(win, 1);

    self.notes_list.cell.setHeight(win.height - sb_height);
    self.notes_list.cell.setOffsetY(0);
    self.notes_list.cell.setOffsetX(self.directory_tree.cell.width);
    self.notes_list.draw(win);
    self.notes_list.drawHeader(win, self.directory_tree.cell.width + 1);

    const editor_xoff = self.notes_list.cell.width + self.directory_tree.cell.width;
    self.editor.cell.setHeight(win.height - sb_height);
    self.editor.cell.setOffsetY(0);
    self.editor.cell.setOffsetX(editor_xoff);
    self.editor.draw(win);
    try self.editor.drawHeader(win, editor_xoff + 1);

    self.status_bar.cell.setOffsetY(self.editor.cell.height);
    try self.status_bar.draw(win);
}

pub fn focusColumn(self: *App, index: u16) void {
    self.directory_tree.setFocus(index == 1);
    self.notes_list.setFocus(index == 2);
    self.editor.setFocus(index == 3);
    self.current_column = index;
}

/// Selects and highlights the respectivley next of the
/// currently selected column.
pub fn focusNextColumn(self: *App, cycle: bool) void {
    const num_cols = 3;
    var current_column = self.current_column;

    if (cycle and current_column == num_cols) {
        current_column = 0;
    }

    const index = @min(current_column + 1, num_cols);
    return self.focusColumn(index);
}

/// Selects and highlights the respectivley previous of the
/// currently selected column.
pub fn focusPrevColumn(self: *App, cycle: bool) void {
    const first_col = 1;
    var current_column = self.current_column;

    if (cycle and current_column == 1) {
        current_column = 4;
    }

    const column = @max(current_column - 1, first_col);
    return self.focusColumn(column);
}

pub fn setMode(self: *App, mode: Editor.TextArea.Vim.Mode) void {
    self.mode = mode;
}

pub fn quit(self: *App) !void {
    try self.config.meta_infos.write();
    self.should_quit = true;
}

pub fn deinit(self: *App) void {
    self.config.deinit();
    self.alloc.destroy(self.config);

    self.directory_tree.deinit();
    self.alloc.destroy(self.directory_tree);

    self.notes_list.deinit();
    self.alloc.destroy(self.notes_list.cell);
    self.alloc.destroy(self.notes_list);

    self.editor.deinit();
    self.alloc.destroy(self.editor);

    self.status_bar.deinit();
    self.alloc.destroy(self.status_bar);

    self.vx.deinit(self.alloc, self.tty.writer());
    self.tty.deinit();
}
