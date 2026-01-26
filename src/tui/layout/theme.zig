const std = @import("std");
const vx = @import("vaxis");

pub const Color = struct {
    pub const Border = struct {
        pub const fg: vx.Color = .{ .rgb = .{ 96, 109, 135 } };
        pub const fg_focused: vx.Color = .{ .rgb = .{ 105, 200, 220 } };
    };

    pub const CellHeader = struct {
        pub const fg: vx.Color = Color.Border.fg;
        pub const fg_focused: vx.Color = .{ .rgb = .{ 240, 240, 240 } };
    };

    pub const LineNumber = struct {
        pub const fg: vx.Color = .{ .rgb = .{ 110, 110, 110 } };
    };
};

pub const border_T = [6][]const u8;

pub const Border = struct {
    pub const none: border_T = .{ "", "", "", "", "", "" };
    pub const normal: border_T = .{ "┌", "─", "┐", "│", "┘", "└" };
    pub const rounded: border_T = .{ "╭", "─", "╮", "│", "╯", "╰" };
    pub const double: border_T = .{ "╔", "═", "╗", "║", "╝", "╚" };
    pub const thick: border_T = .{ "┏", "━", "┓", "┃", "┛", "┗" };
};

pub const Icon = enum {
    pen,
    note,
    dir_open,
    dir_closed,
    dot,
    pin,

    pub fn getNerd(self: Icon) []const u8 {
        return switch (self) {
            .pen => "",
            .note => "󰎞",
            .dir_open => "",
            .dir_closed => "󰉋",
            .dot => "",
            .pin => "󰐃",
        };
    }

    pub fn getAlt(self: Icon) []const u8 {
        return switch (self) {
            .pen => ">",
            .note => "",
            .dir_open => "▼",
            .dir_closed => "▶",
            .dot => "*",
            .pin => "#",
        };
    }
};
