package curriculum

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/waaldev/hjkl/embeds"
	"github.com/waaldev/hjkl/internal/keys"
	"gopkg.in/yaml.v3"
)

const curriculumDir = "curriculum"

// Load reads every belt YAML from the embedded filesystem.
func Load() (*Catalog, error) {
	return LoadFS(embeds.FS)
}

// LoadFS reads belt YAML files from fs.
func LoadFS(fsys fs.FS) (*Catalog, error) {
	entries, err := fs.ReadDir(fsys, curriculumDir)
	if err != nil {
		return nil, fmt.Errorf("curriculum: %w", err)
	}
	cat := &Catalog{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join(curriculumDir, e.Name()))
		if err != nil {
			return nil, err
		}
		var file beltFile
		if err := yaml.Unmarshal(raw, &file); err != nil {
			return nil, fmt.Errorf("curriculum %s: %w", e.Name(), err)
		}
		if file.ID == "" {
			return nil, fmt.Errorf("curriculum %s: missing belt id", e.Name())
		}
		challenges := make([]Challenge, 0, len(file.Challenges))
		for i, ch := range file.Challenges {
			if err := normalizeChallenge(&ch, file.Belt); err != nil {
				return nil, fmt.Errorf("curriculum %s challenge %d (%s): %w", e.Name(), i, ch.ID, err)
			}
			challenges = append(challenges, ch)
		}
		b := file.Belt
		b.Challenges = challenges
		cat.Belts = append(cat.Belts, b)
		cat.Challenges = append(cat.Challenges, challenges...)
	}
	sort.Slice(cat.Belts, func(i, j int) bool { return cat.Belts[i].Rank < cat.Belts[j].Rank })
	// Keep Challenges in belt-rank then file order.
	var ordered []Challenge
	for _, b := range cat.Belts {
		ordered = append(ordered, b.Challenges...)
	}
	cat.Challenges = ordered
	cat.index()
	return cat, nil
}

func normalizeChallenge(ch *Challenge, belt Belt) error {
	if ch.ID == "" {
		return fmt.Errorf("missing id")
	}
	if ch.Belt == "" {
		ch.Belt = belt.ID
	}
	if ch.Type == "" {
		ch.Type = TypeTransform
	}
	ch.Start = normalizeText(ch.Start)
	ch.Target = normalizeText(ch.Target)
	if ch.Target == "" {
		ch.Target = ch.Start
	}
	if len(ch.StartCursor) == 0 {
		ch.StartCursor = []int{1, 1}
	}
	if len(ch.StartCursor) != 2 {
		return fmt.Errorf("start_cursor must be [line, col]")
	}
	if ch.TargetCursor != nil && len(ch.TargetCursor) != 2 {
		return fmt.Errorf("target_cursor must be [line, col]")
	}
	if ch.Type == TypeNavigate && len(ch.TargetCursor) != 2 {
		return fmt.Errorf("navigate challenges need target_cursor")
	}
	if ch.Par <= 0 {
		n := keys.Count(ch.Solution)
		if n == 0 {
			return fmt.Errorf("missing par and solution")
		}
		ch.Par = n
	}
	if ch.Solution == "" {
		return fmt.Errorf("missing solution")
	}
	if ch.Type == TypeSpeedrun && ch.TimeTargetMS <= 0 {
		return fmt.Errorf("speedrun challenges need time_target_ms")
	}
	return nil
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSuffix(s, "\n")
}

func keysCount(notation string) int { return keys.Count(notation) }

// LoadPersonal reads personal drills (one JSON challenge per file) from dir.
// A missing dir is not an error; unreadable files are skipped.
func LoadPersonal(dir string) ([]Challenge, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Challenge
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var ch Challenge
		if json.Unmarshal(raw, &ch) != nil || normalizeChallenge(&ch, Belt{ID: PersonalBelt}) != nil {
			continue
		}
		out = append(out, ch)
	}
	return out, nil
}
