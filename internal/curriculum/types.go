package curriculum

// ChallengeType is the kind of drill the harness runs.
type ChallengeType string

const (
	TypeTransform ChallengeType = "transform"
	TypeNavigate  ChallengeType = "navigate"
	TypeDot       ChallengeType = "dot"
	TypeGolf      ChallengeType = "golf"
	TypeSpeedrun  ChallengeType = "speedrun"
	TypeBoss      ChallengeType = "boss"
)

// Belt is a rank in the dojo curriculum.
type Belt struct {
	ID          string      `yaml:"id" json:"id"`
	Name        string      `yaml:"name" json:"name"`
	Glyph       string      `yaml:"glyph" json:"glyph"`
	Rank        int         `yaml:"rank" json:"rank"`
	Title       string      `yaml:"title" json:"title"`
	Description string      `yaml:"description" json:"description"`
	Color       string      `yaml:"color" json:"color"`
	Principles  []string    `yaml:"principles" json:"principles"`
	Challenges  []Challenge `yaml:"-" json:"challenges"`
}

// Challenge is a single nvim drill.
type Challenge struct {
	ID           string        `yaml:"id" json:"id"`
	Belt         string        `yaml:"belt" json:"belt"`
	Title        string        `yaml:"title" json:"title"`
	Type         ChallengeType `yaml:"type" json:"type"`
	Skills       []string      `yaml:"skills" json:"skills"`
	Principle    string        `yaml:"principle" json:"principle"`
	Brief        string        `yaml:"brief" json:"brief"`
	Teach        string        `yaml:"teach" json:"teach"`
	Hint         string        `yaml:"hint" json:"hint"`
	Start        string        `yaml:"start" json:"start"`
	Target       string        `yaml:"target" json:"target"`
	StartCursor  []int         `yaml:"start_cursor" json:"start_cursor"`
	TargetCursor []int         `yaml:"target_cursor,omitempty" json:"target_cursor,omitempty"`
	Par          int           `yaml:"par" json:"par"`
	Solution     string        `yaml:"solution" json:"solution"`
	// Require and Forbid are regexps checked against the command keys
	// (Normal/Visual mode only). They make a drill about its technique,
	// not just its result.
	Require []string `yaml:"require,omitempty" json:"require,omitempty"`
	Forbid  []string `yaml:"forbid,omitempty" json:"forbid,omitempty"`
	// TimeTargetMS is the speedrun clock target, from the first key.
	TimeTargetMS int `yaml:"time_target_ms,omitempty" json:"time_target_ms,omitempty"`
	// TargetRegister makes a register part of the win condition.
	TargetRegister *Register `yaml:"target_register,omitempty" json:"target_register,omitempty"`
	// Hints is an optional hand-written ladder, gentlest first. Without it
	// the ladder is built from the skills, Hint and Solution (HintLadder).
	Hints []string `yaml:"hints,omitempty" json:"hints,omitempty"`
	// Variants are alternative buffers for the same skill, used by reviews
	// so you practice the skill rather than memorize one puzzle.
	Variants []Variant `yaml:"variants,omitempty" json:"-"`
	// Review hides the lesson: no brief, no teach, only a skill-name hint.
	Review bool `yaml:"-" json:"review,omitempty"`
	// Technique is shown when a require/forbid rule fails.
	Technique string `yaml:"technique,omitempty" json:"technique,omitempty"`
	Language  string `yaml:"language,omitempty" json:"language,omitempty"`
}

// beltFile is the on-disk YAML shape (challenges nested under the belt).
type beltFile struct {
	Belt       `yaml:",inline"`
	Challenges []Challenge `yaml:"challenges"`
}

// Catalog is the loaded curriculum.
type Catalog struct {
	Belts      []Belt
	Challenges []Challenge
	byID       map[string]Challenge
	byBelt     map[string][]Challenge
	bySkill    map[string][]Challenge
}

func (c *Catalog) Challenge(id string) (Challenge, bool) {
	ch, ok := c.byID[id]
	return ch, ok
}

func (c *Catalog) Belt(id string) (Belt, bool) {
	for _, b := range c.Belts {
		if b.ID == id {
			return b, true
		}
	}
	return Belt{}, false
}

func (c *Catalog) ChallengesForBelt(id string) []Challenge {
	return c.byBelt[id]
}

func (c *Catalog) ChallengesForSkill(skill string) []Challenge {
	return c.bySkill[skill]
}

func (c *Catalog) BeltIDs() []string {
	ids := make([]string, len(c.Belts))
	for i, b := range c.Belts {
		ids[i] = b.ID
	}
	return ids
}

func (c *Catalog) ChallengeBelts() map[string]string {
	m := make(map[string]string, len(c.Challenges))
	for _, ch := range c.Challenges {
		m[ch.ID] = ch.Belt
	}
	return m
}

func (c *Catalog) Skills() []string {
	seen := map[string]struct{}{}
	var out []string
	for _, ch := range c.Challenges {
		for _, s := range ch.Skills {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func (c *Catalog) index() {
	c.byID = make(map[string]Challenge, len(c.Challenges))
	c.byBelt = make(map[string][]Challenge)
	c.bySkill = make(map[string][]Challenge)
	for _, ch := range c.Challenges {
		c.byID[ch.ID] = ch
		c.byBelt[ch.Belt] = append(c.byBelt[ch.Belt], ch)
		for _, s := range ch.Skills {
			c.bySkill[s] = append(c.bySkill[s], ch)
		}
	}
}

// Register is a register name and the text it must hold.
type Register struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}
