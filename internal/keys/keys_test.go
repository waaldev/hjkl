package keys

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"dw", []string{"d", "w"}},
		{"A!<Esc>", []string{"A", "!", "<Esc>"}},
		{"ci\"foo<Esc>", []string{"c", "i", "\"", "f", "o", "o", "<Esc>"}},
		{"<C-d><C-u>", []string{"<C-d>", "<C-u>"}},
		{"j.j.", []string{"j", ".", "j", "."}},
		{">>", []string{">", ">"}},
		{":s/a/b<CR>", []string{":", "s", "/", "a", "/", "b", "<CR>"}},
	}
	for _, tt := range tests {
		got := Parse(tt.in)
		if len(got) != len(tt.want) {
			t.Fatalf("Parse(%q) len=%d want %d (%v)", tt.in, len(got), len(tt.want), got)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("Parse(%q)[%d]=%q want %q", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestCount(t *testing.T) {
	if Count("cwbar<Esc>j.j.") != 10 {
		t.Fatalf("Count = %d", Count("cwbar<Esc>j.j."))
	}
}
