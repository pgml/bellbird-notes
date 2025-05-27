package components

import "testing"

func TestDirectoryTree_getChildren(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		path  string
		level int
		want  []TreeItem
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDirectoryTree()
			got := d.getChildren(tt.path, tt.level)
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("getChildren() = %v, want %v", got, tt.want)
			}
		})
	}
}
