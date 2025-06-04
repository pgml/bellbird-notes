package statusbarcolumn

type Column int

const (
	General Column = iota
	Info
	CursorPos
	Progress
)

//var StatusBarColumn = struct {
//	General, Info, CursorPos, Progress Column
//}{
//	General:   ColumnGeneral,
//	Info:      ColumnInfo,
//	CursorPos: ColumnCursorPos,
//	Progress:  ColumnProgress,
//}
