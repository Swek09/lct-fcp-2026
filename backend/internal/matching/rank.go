package matching

import (
	"fmt"
	"sort"
	"strings"
)

type Evidence struct {
	TestScore    int
	Skills       []string
	Achievements int
}
type Score struct {
	Total   int      `json:"total"`
	Reasons []string `json:"reasons"`
}

func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "golang":
		return "go"
	case "postgres", "postgresql":
		return "postgresql"
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "react.js", "reactjs":
		return "react"
	}
	return s
}
func Rank(p Evidence, wanted []string) Score {
	reasons := []string{fmt.Sprintf("Подтверждённый тест: %d из 100", p.TestScore)}
	score := p.TestScore * 50 / 100
	have := map[string]bool{}
	for _, s := range p.Skills {
		have[Normalize(s)] = true
	}
	hit := 0
	seen := map[string]bool{}
	for _, s := range wanted {
		k := Normalize(s)
		if seen[k] || k == "" {
			continue
		}
		seen[k] = true
		if have[k] {
			hit++
		}
	}
	if len(seen) > 0 {
		score += hit * 40 / len(seen)
		reasons = append(reasons, fmt.Sprintf("Стек из профиля: %d из %d нужных навыков", hit, len(seen)))
	} else {
		reasons = append(reasons, "Укажите стек, чтобы уточнить соответствие")
	}
	n := min(p.Achievements, 3)
	score += n * 10 / 3
	if len(seen) == 0 {
		score = score * 100 / 60
	}
	if n > 0 {
		reasons = append(reasons, fmt.Sprintf("Достижения ФСП: %d (демонстрационные данные)", p.Achievements))
	} else {
		reasons = append(reasons, "Без истории ФСП — учитывается результат теста")
	}
	return Score{score, reasons}
}
func ParseNeed(text string) (spec, grade string, skills []string) {
	text = strings.ToLower(text)
	skills = []string{}
	for _, v := range []struct {
		k     string
		terms []string
	}{{"backend", []string{"backend", "бэкенд", "бекенд", "сервер"}}, {"frontend", []string{"frontend", "фронтенд", "фронт"}}, {"analytics", []string{"аналитик", "analytics", "data analyst"}}} {
		for _, t := range v.terms {
			if strings.Contains(text, t) {
				spec = v.k
				break
			}
		}
	}
	for _, v := range []struct {
		k     string
		terms []string
	}{{"junior", []string{"junior", "джуниор"}}, {"middle", []string{"middle", "мидл"}}, {"senior", []string{"senior", "сеньор"}}} {
		for _, t := range v.terms {
			if strings.Contains(text, t) {
				grade = v.k
				break
			}
		}
	}
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '+' || r == '#' || r == '.')
	})
	known := map[string]bool{"go": true, "postgresql": true, "redis": true, "python": true, "sql": true, "react": true, "typescript": true, "javascript": true, "docker": true, "kubernetes": true, "excel": true, "figma": true, "css": true, "html": true}
	seen := map[string]bool{}
	for _, w := range words {
		w = Normalize(w)
		if known[w] && !seen[w] {
			skills = append(skills, w)
			seen[w] = true
		}
	}
	sort.Strings(skills)
	return
}
