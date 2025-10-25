## Global keybindings

| Key                   | Action                      | Cmd                |
| --------------------- | --------------------------- | ------------------ |
| `ctrl+w l`            | Focus next column           |                    |
| `ctrl+w h`            | Focus previous column       |                    |
| `:`                   | Enter command mode          |                    |
| `/`                   | Search in current note      |                    |
| `space e`             | Show open notes             | `:b`, `:buffers`   |
| `space q`             | Close currently open note   | `:bd`              |
| `space n`             | Create a new scratch buffer | `:new`             |

## Folders

| Key        | Action            | Info                                                       |
| ---------- | ----------------- | ---------------------------------------------------------- |
| `j`        | Folder down       |                                                            |
| `k`        | Folder up         |                                                            |
| `h`        | Collapse folder   |                                                            |
| `l`        | Expand folder     |                                                            |
| `d` or `n` | Create new folder | Creates a new folder as a sibling of the selected folder   |
| `D`        | Delete folder     | Deletes the selected folder                                |
| `R`        | Rename folder     |                                                            |
| `gg`       | Go to top         |                                                            |
| `G`        | Go to bottom      |                                                            |
| `P`        | Pin/Unpin folder  |                                                            |
| `yy`       | Yank note         | Yanks selected directory                                   |
| `p`        | Paste note        | pastes the yanked directory into the selected directory    |

## Notes

| Key        | Action             | Info                                                  |
| ---------- | ------------------ | ----------------------------------------------------- |
| `j`        | Note down          |                                                       |
| `k`        | Note up            |                                                       |
| `%` or `n` | Create new note    |                                                       |
| `D`        | Delete Note        | Deletes the selected note                             |
| `R`        | Rename folder      |                                                       |
| `gg`       | Go to top          |                                                       |
| `G`        | Go to bottom       |                                                       |
| `P`        | Pin/Unpin note     |                                                       |
| `yy`       | Yank note          | Yanks selected note                                   |
| `dd`       | Cut note           | Cuts selected note                                    |
| `p`        | Paste note         | pastes the yanked/cut note into the current directory |


## Editor

### Movement

| Key        | Mode           | Action                                                 | Info   |
| ---------- | -------------- | ------------------------------------------------------ | ------ |
| `j`        | Normal         | Move cursor down                                       |        |
| `k`        | Normal         | Move cursor up                                         |        |
| `l`        | Normal         | Move cursor right                                      |        |
| `h`        | Normal         | Move cursor left                                       |        |
| `gj`       | Normal, Visual | Move cursor down in multi line text                    |        |
| `gk`       | Normal, Visual | Move cursor up in multi line text                      |        |
| `f`        | Normal, Visual | Jump to the next occurence of a character              |        |
| `F`        | Normal, Visual | Jump to the previous occurence of a character          |        |
| `ctrl+d`   | Normal, Visual | Move half page down                                    |        |
| `ctrl+u`   | Normal, Visual | Move half page up                                      |        |
| `gg`       | Normal         | Move cursor to top                                     |        |
| `G`        | Normal         | Move cursor to bottom                                  |        |
| `w`        | Normal, Visual | Move to the start of the next word                     |        |
| `e`        | Normal, Visual | Move to the end of the next word                       |        |
| `b`        | Normal, Visual | Move to the start of the previous word                 |        |
| `^` or `_` | Normal, Visual | Jump to the first non blank character                  |        |
| `0`        | Normal, Visual | Jump to the start of the line                          |        |
| `$`        | Normal, Visual | Jump to the end of the line                            |        |

### Search

| Key        | Mode           | Action                                                 | Info   |
| ---------- | -------------- | ------------------------------------------------------ | ------ |
| `/`        | Normal         | Find in buffer                                         |        |
| `n`        | Normal         | Move to next match                                     |        |
| `N`        | Normal         | Move to previous match                                 |        |
| `*`        | Normal         | Highlight word under cursor                            |        |

### Editing

| Key        | Mode           | Action                                                      | Info   |
| ---------- | -------------- | ----------------------------------------------------------- | ------ |
| `i`        | Normal         | Insert at cursor                                            |        |
| `I`        | Normal         | Insert at start of line                                     |        |
| `a`        | Normal         | Insert after cursor                                         |        |
| `A`        | Normal         | Insert at end of line                                       |        |
| `o`        | Normal         | Insert line below                                           |        |
| `O`        | Normal         | Insert line above                                           |        |
| `r`        | Normal         | Replace character under cursor                              |        |
| `u`        | Normal         | Undo last change                                            |        |
| `ctrl+r`   | Normal         | Redo last change                                            |        |
| `J`        | Normal         | Join line below                                             |        |
| `dd`       | Normal         | Delete line                                                 |        |
| `dw`       | Normal         | Delete characters from cursor position to end of word       |        |
| `diw`      | Normal         | Delete word under cursor                                    |        |
| `daw`      | Normal         | Delete word under cursor and the space after                |        |
| `dj`       | Normal         | Delete current and line below                               |        |
| `dk`       | Normal         | Delete current and line above                               |        |
| `df`       | Normal         | Delete from cursor to (including) next occurence            |        |
| `dF`       | Normal         | Delete from cursor to (including) previous occurence        |        |
| `dt`       | Normal         | Delete from cursor to next occurence                        |        |
| `dT`       | Normal         | Delete from cursor to previous occurence                    |        |
| `D`        | Normal         | Delete to the end of the line                               |        |
| `C`        | Normal         | Delete to the end of the line and substitute                |        |
| `cc`       | Normal         | Delete line and substitute                                  |        |
| `ciw`      | Normal         | Change word under cursor                                    |        |
| `caw`      | Normal         | Change word under cursor including space after	            |        |
| `cf`       | Normal         | Change from cursor to (including) next occurence            |        |
| `cF`       | Normal         | Change from cursor to (including) previous occurence        |        |
| `ct`       | Normal         | Change from cursor to next occurence                        |        |
| `cT`       | Normal         | Change from cursor to previous occurence                    |        |
| `s`        | Normal         | Delete character and substitute 	                        |        |
| `x`        | Normal         | Delete character                                            |        |

### Selecting

| Key        | Mode           | Action                                                 | Info   |
| ---------- | -------------- | ------------------------------------------------------ | ------ |
| `v`        | Normal         | Visual mode                                            |        |
| `V`        | Normal         | Visual line mode                                       |        |
| `iw`       | Visual         | Select inner word                                      |        |
| `aw`       | Visual         | Select outer word                                      |        |

### Cut, copy, paste

| Key        | Mode           | Action                                                 | Info   |
| ---------- | -------------- | ------------------------------------------------------ | ------ |
| `d`        | Visual         | Delete (cut) selection                                 |        |
| `D`        | Visual         | Delete (cut) line                                      |        |
| `x`        | Visual         | Delete (cut) selection                                 |        |
| `s`        | Visual         | Delete (cut) selection and substitute                  |        |
| `c`        | Visual         | Delete (cut) selection and substitute                  |        |
| `y`        | Visual         | Yank selection	                                       |        |
| `Y`        | normal         | Yank text after cursor                                 |        |
| `yy`       | Normal         | Yank line     	                                       |        |
| `yiw`      | Normal         | Yank word     	                                       |        |
| `yaw`      | Normal         | Yank word and space after                              |        |
| `p`        | Normal         | Paste from clipboard                                   |        |

### Buffer List

| Key        | Mode           | Action                                                 | Info   |
| ---------- | -------------- | ------------------------------------------------------ | ------ |
| `k`        | Normal         | Move cursor up                                         |        |
| `j`        | Normal         | Move cursor down                                       |        |
| `gg`       | Normal         | Move cursor to top                                     |        |
| `G`        | Normal         | Move cursor to bottom                                  |        |
| `D`        | Normal         | Delete selected buffer                                 |        |
