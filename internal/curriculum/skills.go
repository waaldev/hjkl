package curriculum

// SkillDoc is the cheat-sheet blurb for a skill id.
type SkillDoc struct {
	ID      string
	Keys    string
	Summary string
	Belt    string
}

// SkillDocs is the unlocked-skill cheatsheet corpus.
var SkillDocs = []SkillDoc{
	{ID: "modes", Keys: "<Esc> i", Summary: "Normal is home. Insert is for typing. Esc always brings you back.", Belt: "white"},
	{ID: "insert", Keys: "i a o I A O", Summary: "i before cursor, a after, o/O open a line, I/A go to the ends.", Belt: "white"},
	{ID: "hjkl", Keys: "h j k l", Summary: "Left down up right. Keep your fingers on the home row.", Belt: "white"},
	{ID: "undo", Keys: "u <C-r>", Summary: "u undoes the last change (or insert chunk). Ctrl-r redoes.", Belt: "white"},
	{ID: "write-quit", Keys: ":w :q :q!", Summary: ":w writes, :q quits, :q! abandons. :wq does both.", Belt: "white"},
	{ID: "x", Keys: "x X", Summary: "x deletes the char under the cursor; X deletes before it.", Belt: "white"},
	{ID: "words", Keys: "w b e ge", Summary: "w next word, b back, e end of word. Capital W/B/E skip punctuation.", Belt: "yellow"},
	{ID: "line-ends", Keys: "0 ^ $", Summary: "0 first column, ^ first non-blank, $ end of line.", Belt: "yellow"},
	{ID: "dd-yy-p", Keys: "dd yy p P", Summary: "dd delete line, yy yank line, p paste after, P paste before.", Belt: "yellow"},
	{ID: "counts", Keys: "2w 3j d2w", Summary: "A count prefixes a command. Prefer . when the edit should undo in pieces.", Belt: "yellow"},
	{ID: "operators", Keys: "d c y", Summary: "Delete, change, yank. They wait for a motion.", Belt: "orange"},
	{ID: "dot", Keys: ".", Summary: "Repeat the last change. The whole point of operator+motion.", Belt: "orange"},
	{ID: "find-char", Keys: "f t F T ; ,", Summary: "f{char} onto, t{char} till. ; repeats, , reverses.", Belt: "green"},
	{ID: "text-objects", Keys: "iw aw i\" a( ip it", Summary: "Inner vs a (with delimiters). Pair with d/c/y/v.", Belt: "green"},
	{ID: "percent", Keys: "%", Summary: "Jump to the matching paren, bracket, or brace.", Belt: "green"},
	{ID: "gn", Keys: "cgn dgn .", Summary: "gn selects the next search match. cgn then . replaces matches one at a time.", Belt: "green"},
	{ID: "search", Keys: "/ ? n N * #", Summary: "/ forward, ? back, n/N next, * / # word under cursor.", Belt: "blue"},
	{ID: "marks", Keys: "m' '' `.", Summary: "ma set mark a; 'a line, `a exact. '' toggles last jump.", Belt: "blue"},
	{ID: "jumps", Keys: "<C-o> <C-i> g; g, gi", Summary: "Jumplist and changelist. gi resumes last insert.", Belt: "blue"},
	{ID: "scroll", Keys: "H M L zz <C-d> <C-u>", Summary: "Screen-relative jumps and half-page scroll.", Belt: "blue"},
	{ID: "visual", Keys: "v V <C-v>", Summary: "Character, line, and block visual. Prefer operators when . should replay.", Belt: "purple"},
	{ID: "registers", Keys: "\"0 \"_ \"ay \"A <C-r>", Summary: "Yank register, black hole, named and appending registers, insert-paste.", Belt: "purple"},
	{ID: "macros", Keys: "qa q @q @@", Summary: "Record into a register, play with @. Normalize, strike, abort.", Belt: "purple"},
	{ID: "substitute", Keys: ":s :%s & g& \\v", Summary: "Substitute, repeat with &, very-magic search with \\v.", Belt: "purple"},
	{ID: "buffers", Keys: ":ls :b :bn :bp", Summary: "Buffers are the files in memory. Switch without losing undo.", Belt: "brown"},
	{ID: "windows", Keys: "<C-w>s <C-w>v <C-w>h/j/k/l", Summary: "Splits and moving between them.", Belt: "brown"},
	{ID: "quickfix", Keys: ":vimgrep :copen :cnext :cdo", Summary: "Project-wide hits live in the quickfix list.", Belt: "brown"},
	{ID: "ex-ranges", Keys: ":t :m :sort .,+1 :1t$", Summary: "Ex commands take ranges: copy, move, sort and delete lines by number or relative position.", Belt: "brown"},
	{ID: "global", Keys: ":g :norm", Summary: ":g/pat/cmd on matching lines; :norm types Normal keys over a range.", Belt: "brown"},
	{ID: "golf", Keys: "combine", Summary: "Shortest repeatable sequence that reaches the target.", Belt: "black"},
	{ID: "config", Keys: ":map vimrc", Summary: "Build mappings that encode your own grammar.", Belt: "black"},
}
