package curriculum

// Principles are Practical Vim ideas tagged onto challenges.
var Principles = map[string]string{
	"modes":                 "Keep Insert and Normal distinct. Type in Insert; speak in Normal.",
	"dot":                   "Don't count, repeat. Prefer dw.. over d3w when the result should undo in chunks.",
	"grammar":               "Operator + motion = action. One new operator multiplies with every motion you know.",
	"chunk-undo":            "Leave Insert mode at natural pauses so u undoes a meaningful thought, not a whole paragraph.",
	"operators-over-visual": "Prefer operators to Visual mode. viwd works; diw repeats with .",
	"text-objects":          "Text objects grab semantic units: words, quotes, paragraphs, tags.",
	"gn":                    "cgn plus . is search-and-replace, one match at a time - a daily-work favorite.",
	"ex-ranges":             "Ex commands shine on ranges: :t, :m, :g, :s, :normal, and @: to replay.",
	"macros":                "Normalize at a known position, strike with repeatable motions, abort on a failed motion.",
	"registers":             "\"0 keeps the last yank after a delete. \"_ is the black hole. \"A appends.",
	"jumps":                 "The jumplist (<C-o>/<C-i>) and changelist (g;/g,) are breadcrumb trails. gi returns to last insert.",
	"files":                 ":argdo, :cdo, :vimgrep and quickfix scale an edit across a project.",
}

func PrincipleText(id string) string {
	if t, ok := Principles[id]; ok {
		return t
	}
	return ""
}
