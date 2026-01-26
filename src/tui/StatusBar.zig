const StatusBar = @This();

const std = @import("std");
const vx = @import("vaxis");

const App = @import("../App.zig");
const Cell = @import("layout/Cell.zig");
const utils = @import("../utils.zig");

alloc: std.mem.Allocator,

app: *App,

cell: *Cell,

default_width: u16 = 0,

default_height: u16 = 1,

//content_cols: [4]*Cell,

info_column: *InfoColumn,

const InfoColumn = struct {
    alloc: std.mem.Allocator,

    cell: *Cell,

    value: std.ArrayList(vx.Cell.Character),

    col: i32 = 0,

    pub fn init(alloc: std.mem.Allocator) !*InfoColumn {
        const self = try alloc.create(InfoColumn);

        self.* = .{
            .alloc = alloc,
            .value = .empty,
            .col = 0,
            .cell = try .init(alloc),
        };

        return self;
    }

    pub fn insertSliceAtCursor(self: *InfoColumn, slice: []const u8) !void {
        try self.value.insert(self.alloc, @intCast(self.col), .{
            .grapheme = slice,
            .width = 1,
        });
        self.col += 1;
    }

    pub fn getValue(self: InfoColumn) []vx.Cell.Character {
        return self.value.items;
    }

    pub fn getValueStr(self: InfoColumn) ![]const u8 {
        var val: std.ArrayList(u8) = .empty;
        for (self.getValue()) |char| {
            try val.appendSlice(self.alloc, char.grapheme);
        }
        return try val.toOwnedSlice(self.alloc);
    }

    pub fn clear(self: *InfoColumn) void {
        self.value.shrinkAndFree(self.alloc, 0);
        self.value = .empty;
        self.col = 0;
    }

    pub fn deinit(self: *InfoColumn) void {
        self.value.deinit(self.alloc);
        self.alloc.destroy(self.cell);
    }
};

pub fn init(alloc: std.mem.Allocator, app: *App) !*StatusBar {
    const self = try alloc.create(StatusBar);

    self.* = .{
        .alloc = alloc,
        .app = app,
        .cell = try .init(alloc),
        .info_column = try .init(alloc),
    };

    self.cell.setHeight(self.default_height);
    self.info_column.cell.setOffsetX(self.cell.offset_x);

    return self;
}

pub fn update(self: *StatusBar, event: App.Event) !void {
    if (!self.cell.isFocused()) {
        return;
    }

    switch (event) {
        .key_press => |key| {
            if (key.matches(vx.Key.enter, .{})) {
                const val = try self.info_column.getValueStr();
                defer self.alloc.free(val);

                if (utils.strEql(val, "q")) {
                    return try self.app.quit();
                }
            }
            if (key.text) |text| {
                try self.info_column.insertSliceAtCursor(text);
            }
        },
        else => {},
    }
}

pub fn draw(self: *StatusBar, win: vx.Window) !void {
    var child_opts: vx.Window.ChildOptions = self.cell.getChild();
    child_opts.border = .{};
    const child_win = win.child(child_opts);

    // Use an arena for string manipulations for each draw
    var arena = std.heap.ArenaAllocator.init(self.alloc);
    defer arena.deinit();

    {
        var content: []const u8 = "";
        if (self.shouldShowMode()) {
            content = try self.getModeStr(arena.allocator());
        }
        try self.drawInfoCol(child_win, content);
    }
}

fn drawInfoCol(self: *StatusBar, win: vx.Window, content: []const u8) !void {
    var col: u16 = 1;
    var col_opts = self.info_column.cell.getChild();
    // Reset border
    col_opts.border = .{};
    const child_win = win.child(col_opts);

    // Display current mode
    Cell.write(child_win, &col, 0, content, .{});

    // Display user input prompt
    for (self.info_column.value.items) |*char| {
        Cell.write(child_win, &col, 0, char.grapheme, .{});
    }

    if (self.cell.isFocused()) {
        child_win.showCursor(col, 0);
    }
}

fn shouldShowMode(self: StatusBar) bool {
    return self.app.mode == .insert or
        self.app.mode == .command or
        //self.app.mode == .search or
        //self.app.mode == .search_prompt or
        self.app.mode == .replace or
        self.app.mode == .visual or
        self.app.mode == .visual_line or
        self.app.mode == .visual_block;
}

fn getModeStr(self: StatusBar, alloc: std.mem.Allocator) ![]const u8 {
    var mode = self.app.mode;
    var mode_str: []const u8 = mode.str();
    const buf: []u8 = try alloc.alloc(u8, mode_str.len);
    defer alloc.free(buf);

    mode_str = std.ascii.upperString(buf, mode_str);
    if (mode != .command and mode != .visual_block and mode != .visual_line and
        mode != .visual)
    {
        mode_str = try std.mem.concat(alloc, u8, &[_][]const u8{
            "-- ", mode_str, " --",
        });
    }
    defer alloc.free(mode_str);

    const s = try alloc.dupe(u8, mode_str);
    return s;
}

pub fn focus(self: *StatusBar) void {
    self.app.mode = .command;
    self.info_column.clear();
    self.cell.focus();
}

pub fn blur(self: *StatusBar) void {
    self.app.mode = .normal;
    self.info_column.clear();
    self.cell.blur();
}

pub fn deinit(self: *StatusBar) void {
    //for (0..self.content_cols.len) |i| {
    //    self.alloc.destroy(self.content_cols[i]);
    //}
    self.info_column.deinit();
    self.alloc.destroy(self.info_column);
    self.alloc.destroy(self.cell);
}
