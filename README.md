
# Bellbird Notes

#### A Vim inspired note-taking app for your favourite terminal emulator


[bbnotes_basic.webm](https://github.com/user-attachments/assets/33dabdb5-34cd-45da-96b4-676ed7f48898)


---
> [!IMPORTANT]
> This software is still in early development and should be used with caution.
>
> Even though I already use it on a daily basis, bugs and crashes are to be expected.
---

## Current Key Features

* Basic vim-motions including visual, visual line, insert, replace mode.
* Netrw keybinds for creating, renaming, deleting folders and notes and switching between columns
* Pin/Unpin notes for quicker access
* Buffer support - every note is opened in a new buffer with its own history

[bbnotes_buffers.webm](https://github.com/user-attachments/assets/aa74d6fd-9891-4545-b175-1a0ee326b35d)

[bbnotes_history.webm](https://github.com/user-attachments/assets/b1c0790f-7d67-4080-9c49-7c67021b183a)

## Usage

Open it with `bbnotes` in your favourite terminal and use it like you would use vim.

If you don't have NerdFonts install use `bbnotes --no-nerd-fonts`

#### Keybinds

You can find a list of all currently availabe keybinds [here](docs/keybindings.md)

To change any of the keybinds edit keymap.json under `~/.config/bellbird-notes/keymaps.json` or via `:open keymap`.

A list of the default keybinds can be displayed with `:open defaultkeymap`.


## Currently missing/planned features (no particular order)

#### General

* more config options
* move meta infos to notes directory so that it can be synced

#### Notes/Folders

* Preselect text when creating or renaming

#### Editor

* `U` and `u` for lower- & uppercase in visual mode
* visual block mode
* improve performance on large notes
* save history to file system
* automatically create lists if line starts with a dash
* create lists out of selection
* store time and amount of changes in buffer history
* display undo/redo messages in statusbar
* Multipanes
* Marks

...and a lot more

## Known bugs

* visual mode goes crazy on multilines (wrapped lines)
* General status bar column is cut off if window is too small

## Future Plans

- [x] Overlay support to list open buffers or display dialogues
- [ ] Fuzzyfind notes
- [ ] Search in all notes
- [ ] SQLite Support
- [ ] Themes
- [ ] Localisation
