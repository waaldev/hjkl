package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/waaldev/hjkl/internal/app"
	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/game"
	"github.com/waaldev/hjkl/internal/progress"
	"github.com/waaldev/hjkl/internal/runner"
	"github.com/waaldev/hjkl/internal/store"
)

type page int

const (
	pageHome page = iota
	pageBelts
	pageChallenges
	pageTeach
	pageResults
	pageStats
	pageDaily
	pageCheat
)

type model struct {
	app       *app.App
	page      page
	width     int
	height    int
	cursor    int
	err       string
	status    string
	stars     map[string]int
	xp        int
	streak    store.Streak
	unlock    map[string]bool
	beltIdx   int
	chIdx     int
	active    curriculum.Challenge
	outcome   progress.Outcome
	result    runner.Result
	runDir    string
	cheatQ    string
	shown     bool // the par solution was demoed for this result
	homeItems []string
}

type demoDoneMsg struct{ dir string }

type nvimDoneMsg struct {
	err        error
	resultPath string
	dir        string
}

func Run(a *app.App) error {
	m, err := newModel(a)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func newModel(a *app.App) (*model, error) {
	ctx := context.Background()
	stars, err := a.Store.Stars(ctx)
	if err != nil {
		return nil, err
	}
	xp, err := a.Store.TotalXP(ctx)
	if err != nil {
		return nil, err
	}
	streak, err := a.Store.Streak(ctx)
	if err != nil {
		return nil, err
	}
	m := &model{
		app:       a,
		page:      pageHome,
		stars:     stars,
		xp:        xp,
		streak:    streak,
		homeItems: []string{"Continue", "Dojo", "Daily", "Stats", "Cheat", "Quit"},
	}
	m.unlock = game.Unlocked(a.Cat.BeltIDs(), a.Cat.ChallengeBelts(), stars)
	return m, nil
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) refresh() {
	ctx := context.Background()
	m.stars, _ = m.app.Store.Stars(ctx)
	m.xp, _ = m.app.Store.TotalXP(ctx)
	m.streak, _ = m.app.Store.Streak(ctx)
	m.unlock = game.Unlocked(m.app.Cat.BeltIDs(), m.app.Cat.ChallengeBelts(), m.stars)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case nvimDoneMsg:
		return m, m.handleNvim(msg)
	case demoDoneMsg:
		_ = os.RemoveAll(msg.dir)
		m.shown = true
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.page {
		case pageHome:
			return m.updateHome(msg)
		case pageBelts:
			return m.updateBelts(msg)
		case pageChallenges:
			return m.updateChallenges(msg)
		case pageTeach:
			return m.updateTeach(msg)
		case pageResults:
			return m.updateResults(msg)
		case pageStats, pageDaily, pageCheat:
			if msg.String() == "q" || msg.String() == "esc" || msg.String() == "enter" {
				m.page = pageHome
				m.cursor = 0
			}
			if m.page == pageCheat {
				switch msg.String() {
				case "backspace":
					if len(m.cheatQ) > 0 {
						m.cheatQ = m.cheatQ[:len(m.cheatQ)-1]
					}
				case "esc", "enter", "q":
				default:
					if len(msg.Runes) == 1 && msg.Type == tea.KeyRunes {
						m.cheatQ += string(msg.Runes)
					}
				}
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *model) updateHome(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q":
		return m, tea.Quit
	case "j", "down":
		m.cursor = (m.cursor + 1) % len(m.homeItems)
	case "k", "up":
		m.cursor = (m.cursor - 1 + len(m.homeItems)) % len(m.homeItems)
	case "enter", "l":
		switch m.homeItems[m.cursor] {
		case "Continue":
			if ch, ok := progress.NextChallenge(m.app.Cat, m.stars); ok {
				m.active = ch
				m.page = pageTeach
			} else {
				m.status = "Every belt challenge is cleared. Daily reviews keep the edge."
			}
		case "Dojo":
			m.page = pageBelts
			m.cursor = 0
		case "Daily":
			m.page = pageDaily
		case "Stats":
			m.page = pageStats
		case "Cheat":
			m.page = pageCheat
			m.cheatQ = ""
		case "Quit":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) updateBelts(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.app.Cat.Belts)
	switch k.String() {
	case "q", "esc", "h":
		m.page = pageHome
		m.cursor = 1
	case "j", "down":
		m.cursor = (m.cursor + 1) % n
	case "k", "up":
		m.cursor = (m.cursor - 1 + n) % n
	case "enter", "l":
		b := m.app.Cat.Belts[m.cursor]
		if !m.unlock[b.ID] {
			m.status = "Clear 80% of the previous belt to unlock " + b.Name + "."
			return m, nil
		}
		m.beltIdx = m.cursor
		m.page = pageChallenges
		m.cursor = 0
	}
	return m, nil
}

func (m *model) updateChallenges(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	b := m.app.Cat.Belts[m.beltIdx]
	chs := b.Challenges
	if len(chs) == 0 {
		m.page = pageBelts
		return m, nil
	}
	switch k.String() {
	case "q", "esc", "h":
		m.page = pageBelts
		m.cursor = m.beltIdx
	case "j", "down":
		m.cursor = (m.cursor + 1) % len(chs)
	case "k", "up":
		m.cursor = (m.cursor - 1 + len(chs)) % len(chs)
	case "enter", "l":
		m.active = chs[m.cursor]
		m.chIdx = m.cursor
		m.page = pageTeach
	}
	return m, nil
}

func (m *model) updateTeach(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "esc", "h":
		m.page = pageChallenges
		m.cursor = m.chIdx
	case "enter", " ":
		return m, m.launchNvim()
	}
	return m, nil
}

func (m *model) updateResults(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "esc":
		m.page = pageChallenges
		m.cursor = m.chIdx
		m.refresh()
	case "s":
		return m, m.launchDemo()
	case "r":
		return m, m.launchNvim()
	case "enter", "n":
		m.refresh()
		if ch, ok := progress.NextInBelt(m.app.Cat, m.active.Belt, m.stars); ok {
			m.active = ch
			m.page = pageTeach
			return m, nil
		}
		m.page = pageChallenges
		m.cursor = m.chIdx
	}
	return m, nil
}

func (m *model) launchNvim() tea.Cmd {
	dir, err := os.MkdirTemp("", "hjkl-play-*")
	if err != nil {
		m.err = err.Error()
		return nil
	}
	cmd, resultPath, err := runner.Command(m.active, m.app.Cfg.Nvim, dir, "play")
	if err != nil {
		m.err = err.Error()
		_ = os.RemoveAll(dir)
		return nil
	}
	m.runDir = dir
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return nvimDoneMsg{err: err, resultPath: resultPath, dir: dir}
	})
}

// launchDemo plays the par solution in nvim so the player can watch it.
func (m *model) launchDemo() tea.Cmd {
	dir, err := os.MkdirTemp("", "hjkl-demo-*")
	if err != nil {
		m.err = err.Error()
		return nil
	}
	cmd, _, err := runner.Command(m.active, m.app.Cfg.Nvim, dir, "demo")
	if err != nil {
		m.err = err.Error()
		_ = os.RemoveAll(dir)
		return nil
	}
	return tea.ExecProcess(cmd, func(error) tea.Msg { return demoDoneMsg{dir: dir} })
}

func (m *model) handleNvim(msg nvimDoneMsg) tea.Cmd {
	defer os.RemoveAll(msg.dir)
	res, err := runner.ReadResult(msg.resultPath)
	if err != nil {
		m.err = "nvim closed without a result (F10 aborts). " + err.Error()
		m.page = pageTeach
		return nil
	}
	out, err := progress.Apply(context.Background(), m.app.Store, m.app.Cat, m.active, res)
	if err != nil {
		m.err = err.Error()
		m.page = pageTeach
		return nil
	}
	m.result = res
	m.outcome = out
	m.shown = false
	m.page = pageResults
	m.refresh()
	return nil
}

func (m *model) View() string {
	if m.width == 0 {
		return "loading…"
	}
	var body string
	switch m.page {
	case pageHome:
		body = m.viewHome()
	case pageBelts:
		body = m.viewBelts()
	case pageChallenges:
		body = m.viewChallenges()
	case pageTeach:
		body = m.viewTeach()
	case pageResults:
		body = m.viewResults()
	case pageStats:
		body = m.viewStats()
	case pageDaily:
		body = m.viewDaily()
	case pageCheat:
		body = m.viewCheat()
	}
	foot := mutedStyle.Render("j/k move  enter select  q back/quit")
	if m.status != "" {
		foot = goldStyle.Render(m.status) + "\n" + foot
		m.status = ""
	}
	if m.err != "" {
		foot = badStyle.Render(m.err) + "\n" + foot
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, "", foot)
}

func (m *model) viewHome() string {
	head := titleStyle.Render(logo) + "\n" + mutedStyle.Render("  from zero to vim hero") + "\n"
	st := fmt.Sprintf("  xp %d    streak %d    best %d", m.xp, m.streak.Current, m.streak.Best)
	var items []string
	for i, it := range m.homeItems {
		if i == m.cursor {
			items = append(items, selStyle.Render("▸ "+it))
		} else {
			items = append(items, itemStyle.Render("  "+it))
		}
	}
	next := "  curriculum complete"
	if ch, ok := progress.NextChallenge(m.app.Cat, m.stars); ok {
		b, _ := m.app.Cat.Belt(ch.Belt)
		next = fmt.Sprintf("  next: %s %s - %s", b.Glyph, b.Name, ch.Title)
	}
	return head + "\n" + goldStyle.Render(st) + "\n" + mutedStyle.Render(next) + "\n\n" + strings.Join(items, "\n")
}

func (m *model) viewBelts() string {
	var bld strings.Builder
	bld.WriteString(titleStyle.Render("  dojo") + "\n" + mutedStyle.Render("  belts unlock at 80% of the previous") + "\n\n")
	for i, b := range m.app.Cat.Belts {
		frac := game.BeltComplete(b.ID, m.app.Cat.ChallengeBelts(), m.stars)
		lock := " "
		if !m.unlock[b.ID] {
			lock = "🔒"
		}
		line := fmt.Sprintf("%s %s %-8s  %-18s  %3.0f%%", lock, b.Glyph, b.Name, b.Title, frac*100)
		if i == m.cursor {
			bld.WriteString(selStyle.Render(line) + "\n")
		} else {
			col := lipgloss.NewStyle().Foreground(beltColors[b.ID])
			bld.WriteString(col.Render("  "+line) + "\n")
		}
	}
	return bld.String()
}

func (m *model) viewChallenges() string {
	b := m.app.Cat.Belts[m.beltIdx]
	var bld strings.Builder
	bld.WriteString(titleStyle.Render(fmt.Sprintf("  %s %s - %s", b.Glyph, b.Name, b.Title)) + "\n\n")
	for i, ch := range b.Challenges {
		st := starsBar(m.stars[ch.ID])
		line := fmt.Sprintf("%s  %-10s  %s", st, ch.Type, ch.Title)
		if i == m.cursor {
			bld.WriteString(selStyle.Render(line) + "\n")
		} else {
			bld.WriteString(itemStyle.Render(line) + "\n")
		}
	}
	return bld.String()
}

func (m *model) viewTeach() string {
	ch := m.active
	b, _ := m.app.Cat.Belt(ch.Belt)
	prin := curriculum.PrincipleText(ch.Principle)
	body := fmt.Sprintf("%s %s  ·  %s\n\n%s\n\n%s\n\n%s\n\npar %d    skills %s\n\n%s",
		b.Glyph, strings.ToUpper(ch.Belt), ch.Title,
		inkStyle.Render(ch.Brief),
		mutedStyle.Render(strings.TrimSpace(ch.Teach)),
		goldStyle.Render(prin),
		ch.Par, strings.Join(ch.Skills, ", "),
		titleStyle.Render("enter  open neovim    esc  back"),
	)
	return boxStyle.Width(min(m.width-4, 72)).Render(body)
}

func (m *model) viewResults() string {
	ch := m.active
	ok := "CLEAR"
	style := titleStyle
	if !m.result.OK {
		ok = "MISS"
		style = badStyle
	}
	// On a miss the answer stays hidden until you ask to see it (s).
	par := "par  " + ch.Solution
	if !m.result.OK && !m.shown {
		par = "par  hidden - r to retry, s to watch it"
	}
	body := fmt.Sprintf("%s  %s\n\n%s    keys %d / par %d    xp +%d\n%s\n",
		style.Render(ok), ch.Title,
		starsBar(m.outcome.Stars), m.result.KeyCount, ch.Par, m.outcome.XP,
		mutedStyle.Render("you  "+m.result.Keys+"\n"+par),
	)
	if m.outcome.HintsUsed > 0 {
		body += "\n" + mutedStyle.Render(fmt.Sprintf("hints used: %d (-%d star)", m.outcome.HintsUsed, m.outcome.HintsUsed)) + "\n"
	}
	if m.outcome.Technique != "" {
		body += "\n" + goldStyle.Render(m.outcome.Technique) + "\n"
	}
	if m.outcome.DotScore != "" {
		body += "\n" + goldStyle.Render(m.outcome.DotScore) + "\n"
	}
	if p := curriculum.PrincipleText(ch.Principle); p != "" {
		body += "\n" + inkStyle.Render(p) + "\n"
	}
	keysHelp := "enter next    q back"
	if m.outcome.Stars < 3 {
		keysHelp = "s show me    r retry    " + keysHelp
	}
	body += "\n" + mutedStyle.Render(keysHelp)
	return boxStyle.Width(min(m.width-4, 72)).Render(body)
}

func (m *model) viewStats() string {
	var bld strings.Builder
	bld.WriteString(titleStyle.Render("  stats") + "\n")
	bld.WriteString(fmt.Sprintf("  xp %d    streak %d (best %d)\n\n", m.xp, m.streak.Current, m.streak.Best))
	for _, b := range m.app.Cat.Belts {
		frac := game.BeltComplete(b.ID, m.app.Cat.ChallengeBelts(), m.stars)
		bar := progressBar(frac, 16)
		bld.WriteString(fmt.Sprintf("  %s %-8s %s %3.0f%%\n", b.Glyph, b.Name, bar, frac*100))
	}
	bld.WriteString("\n" + titleStyle.Render("  grammar grid") + mutedStyle.Render("  (operator × motion)") + "\n")
	grid, _ := m.app.Store.GrammarGrid(context.Background())
	bld.WriteString("     ")
	for _, mo := range game.GrammarMotions {
		bld.WriteString(fmt.Sprintf("%3s", trimMo(mo)))
	}
	bld.WriteString("\n")
	for _, op := range game.GrammarOps {
		bld.WriteString("  " + op + "  ")
		for _, mo := range game.GrammarMotions {
			n := grid[op+"\t"+mo]
			cell := " · "
			if n > 0 {
				cell = goldStyle.Render(" ■ ")
			}
			bld.WriteString(cell)
		}
		bld.WriteString("\n")
	}
	return bld.String()
}

func (m *model) viewDaily() string {
	ctx := context.Background()
	due, _ := m.app.Store.DueSkills(ctx, time.Now(), 8)
	var bld strings.Builder
	bld.WriteString(titleStyle.Render("  daily") + "\n" + mutedStyle.Render("  spaced reviews + a coach quest") + "\n\n")
	if len(due) == 0 {
		bld.WriteString(inkStyle.Render("  nothing due. train a new belt, or come back tomorrow.") + "\n")
	} else {
		for _, c := range due {
			bld.WriteString(fmt.Sprintf("  • %s  (ef %.2f, interval %.0fd)\n", c.Skill, c.Easiness, c.IntervalDays))
		}
		bld.WriteString("\n" + mutedStyle.Render("  run  hjkl daily  to fight the due skills in nvim") + "\n")
	}
	return bld.String()
}

func (m *model) viewCheat() string {
	unlocked := map[string]struct{}{}
	for _, s := range progress.UnlockedSkills(m.app.Cat, m.stars) {
		unlocked[s] = struct{}{}
	}
	q := strings.ToLower(m.cheatQ)
	var bld strings.Builder
	bld.WriteString(titleStyle.Render("  cheat") + "  " + goldStyle.Render(m.cheatQ+"▌") + "\n\n")
	for _, d := range curriculum.SkillDocs {
		if _, ok := unlocked[d.ID]; !ok && d.Belt != "white" {
			continue
		}
		blob := strings.ToLower(d.ID + " " + d.Keys + " " + d.Summary)
		if q != "" && !strings.Contains(blob, q) {
			continue
		}
		bld.WriteString(goldStyle.Render("  "+d.Keys) + "\n")
		bld.WriteString(inkStyle.Render("    "+d.Summary) + "\n\n")
	}
	return bld.String()
}

func progressBar(frac float64, w int) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	n := int(frac * float64(w))
	return goldStyle.Render(strings.Repeat("█", n)) + mutedStyle.Render(strings.Repeat("░", w-n))
}

func trimMo(s string) string {
	if len(s) > 3 {
		return s[:3]
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
