const Editor = @This();

const std = @import("std");
const vx = @import("vaxis");

const App = @import("../App.zig");
const Cell = @import("layout/Cell.zig");
const Config = @import("../Config.zig");
const log = @import("../log.zig");
pub const TextArea = @import("widgets/TextArea/TextArea.zig");
pub const Buffer = TextArea.Buffer;
const utils = @import("../utils.zig");

const theme = @import("layout/theme.zig");

alloc: std.mem.Allocator,

app: *App,

cell: *Cell,

default_width: u16 = 0,

default_height: u16 = 0,

scroll_view: vx.widgets.ScrollView,

textarea: TextArea,

pub fn init(alloc: std.mem.Allocator, title: []const u8, app: *App) !*Editor {
    const self = try alloc.create(Editor);

    self.* = .{
        .alloc = alloc,
        .app = app,
        .cell = try .init(alloc),
        .textarea = try .init(alloc, app),
        .scroll_view = .{},
    };

    self.cell.setWidth(self.default_width);
    self.cell.title = title;

    return self;
}

pub fn update(self: *Editor, event: App.Event) !void {
    if (!self.cell.isFocused() or self.textarea.numBufs() == 0) {
        return;
    }

    try self.textarea.enableVimMode();

    switch (event) {
        .key_press => |key| {
            try self.textarea.update(.{ .key_press = key });
        },
        else => {},
    }
}

pub fn draw(self: *Editor, win: vx.Window) void {
    var child_win: vx.Window = win.child(self.cell.getChild());
    const gutter_width = 6;
    const top_padding = 0;

    child_win.y_off += top_padding;
    child_win.x_off += gutter_width;
    child_win.width -= gutter_width;
    //child_win.height -= top_padding;

    self.textarea.win = child_win;
    self.textarea.is_focucsed = self.cell.isFocused();

    if (self.textarea.hasBuffers()) {
        const buf: *Buffer = self.textarea.curBuf();
        self.scroll_view.draw(child_win, .{
            .cols = self.textarea.width,
            .rows = buf.numRows(),
        });

        self.textarea.scroll_view = &self.scroll_view;

        const ln: vx.widgets.LineNumbers = .{
            .num_lines = buf.numRows() +| 1,
            .style = .{
                .fg = theme.Color.LineNumber.fg,
            },
        };

        ln.draw(win.child(.{
            .x_off = self.cell.offset_x,
            .y_off = self.cell.offset_y + 1,
            .width = gutter_width,
            .height = self.textarea.height,
        }), self.scroll_view.scroll.y);

        self.textarea.draw() catch return;
    }
}

pub fn drawHeader(self: Editor, win: vx.Window, col: u16) !void {
    var arena = std.heap.ArenaAllocator.init(self.alloc);
    defer arena.deinit();

    if (self.textarea.hasBuffers()) {
        const buf: *Buffer = self.textarea.curBuf();
        if (!std.mem.eql(u8, buf.path, "")) {
            const breadcrumb = try self.getBreadCrumb(arena.allocator());
            if (!std.mem.eql(u8, breadcrumb, "")) {
                self.cell.title = breadcrumb;
            }
        }
    }

    Cell.drawHeader(win, self.cell.title, col, self.cell.isFocused());
}

pub fn restore(self: *Editor) !void {
    const meta = self.app.config.meta_infos;
    if (utils.strEql(meta.last_open_note, "")) {
        return;
    }
    const last_open_note = meta.last_open_note;

    try self.openBuf(last_open_note);

    if (meta.files_info.get(last_open_note)) |file_info| {
        const curpos = file_info.cursor_pos;
        self.textarea.moveCursorTo(@intCast(curpos.row), @intCast(curpos.col));
    }
}

pub fn openBuf(self: *Editor, path: []const u8) !void {
    try self.textarea.openBuf(path);
    self.textarea.repositionView();
}

// Remove the root notes directory from the absolute buffer path,
// Caller owns memory.
pub fn getRelativeBufPath(self: Editor, alloc: std.mem.Allocator) ![]const u8 {
    if (self.textarea.hasBuffers()) {
        const buf: *Buffer = self.textarea.curBuf();
        var rel_path: []u8 = try alloc.alloc(u8, buf.path.len);
        const notes_root = try self.app.config.getNotesRootDir();

        if (std.fs.path.dirname(buf.path)) |dir_name| {
            const len = dir_name.len - notes_root.len;
            _ = std.mem.replace(u8, dir_name, notes_root, "", rel_path[0..]);
            return rel_path[0..len];
        }
    }
    return "";
}

fn getBreadCrumb(self: Editor, alloc: std.mem.Allocator) ![]const u8 {
    if (!self.textarea.hasBuffers()) {
        return "";
    }

    const buf: *Buffer = self.textarea.curBuf();
    const rel_path = try self.getRelativeBufPath(alloc);
    const separator = " › ";

    if (std.mem.eql(u8, rel_path, "")) {
        return "";
    }

    var out_buf: [256]u8 = undefined;
    const replacements = std.mem.replace(u8, rel_path, "/", separator, out_buf[0..]);
    const len = rel_path.len + (replacements * (separator.len - 1));

    return try std.mem.concat(alloc, u8, &[_][]const u8{
        " ",
        theme.Icon.getNerd(.dir_closed),
        out_buf[0..len],
        separator,
        theme.Icon.getNerd(.note),
        " ",
        buf.getName(),
        " ",
    });
}

pub fn focus(self: *Editor) void {
    self.cell.focus();
}

pub fn blur(self: *Editor) void {
    self.cell.blur();
}

pub fn setFocus(self: *Editor, f: bool) void {
    self.cell.setFocus(f);
}

pub fn deinit(self: *Editor) void {
    self.alloc.destroy(self.cell);
    self.textarea.deinit();
}
