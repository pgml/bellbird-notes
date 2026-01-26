const ListItem = @This();

// The row's index is primarily used to determine the indentation
// of a directory.
index: usize,

name: []const u8,

path: []const u8,

//inputModel *textinput.Model

//Styles Styles
//
//width: u32,

//nerd_fonts: bool,

is_pinned: bool = false,

is_cut: bool = false,

is_selected: bool = false,
