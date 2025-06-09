package statusbarcolumn

type Column int

const (
	General Column = iota
	FileInfo
	KeyInfo
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
