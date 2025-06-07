package statusbarcolumn

type Column int

const (
	General Column = iota
	DirInfo
	FileInfo
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
