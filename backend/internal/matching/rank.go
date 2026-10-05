package matching

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type Evidence struct {
	TestScore    int
	Skills       []string
	Achievements int
}
type Score struct {
	Total    int      `json:"total"`
	Reasons  []string `json:"reasons"`
	Test     int      `json:"test"`
	Stack    int      `json:"stack"`
	FSP      int      `json:"fsp"`
	TestMax  int      `json:"test_max"`
	StackMax int      `json:"stack_max"`
	FSPMax   int      `json:"fsp_max"`
	Matched  []string `json:"matched"`
	Missing  []string `json:"missing"`
}

func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "golang", "го":
		return "go"
	case "postgres", "postgresql", "постгрес", "постгресql", "постгрескл":
		return "postgresql"
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "react.js", "reactjs":
		return "react"
	case "node", "nodejs":
		return "node.js"
	case "vuejs", "vue.js":
		return "vue"
	case "k8s":
		return "kubernetes"
	case "csharp":
		return "c#"
	case "dotnet":
		return ".net"
	}
	return s
}
func Rank(p Evidence, wanted []string) Score {
	reasons := []string{fmt.Sprintf("Подтверждённый тест: %d из 100", p.TestScore)}
	test := max(0, min(100, p.TestScore)) * 50 / 100
	score := test
	matched, missing := []string{}, []string{}
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
			matched = append(matched, k)
		} else {
			missing = append(missing, k)
		}
	}
	stack := 0
	if len(seen) > 0 {
		stack = hit * 40 / len(seen)
		score += stack
		reasons = append(reasons, fmt.Sprintf("Стек из профиля: %d из %d нужных навыков", hit, len(seen)))
	} else {
		reasons = append(reasons, "Укажите стек, чтобы уточнить соответствие")
	}
	n := max(0, min(p.Achievements, 3))
	fsp := n * 10 / 3
	score += fsp
	if len(seen) == 0 {
		score = score * 100 / 60
	}
	if n > 0 {
		reasons = append(reasons, fmt.Sprintf("Достижения ФСП: %d (демонстрационные данные)", p.Achievements))
	} else {
		reasons = append(reasons, "Без истории ФСП — учитывается результат теста")
	}
	sort.Strings(matched)
	sort.Strings(missing)
	testMax, stackMax, fspMax := 50, 40, 10
	if len(seen) == 0 {
		testMax, stackMax, fspMax = 83, 0, 17
		// Keep the integer sum exact after normalizing away the unspecified stack.
		test = test * 100 / 60
		fsp = score - test
	}
	return Score{Total: score, Reasons: reasons, Test: test, Stack: stack, FSP: fsp, TestMax: testMax, StackMax: stackMax, FSPMax: fspMax, Matched: matched, Missing: missing}
}
func ParseNeed(text string) (spec, grade string, skills []string) {
	text = strings.ReplaceAll(strings.ToLower(text), "ё", "е")
	text = strings.NewReplacer("back-end", "backend", "front-end", "frontend").Replace(text)
	skills = []string{}
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '#' || r == '.')
	})
	for i := range tokens {
		tokens[i] = strings.TrimSuffix(tokens[i], ".")
	}
	matches := func(terms []string) bool {
		for _, term := range terms {
			for _, token := range tokens {
				if strings.HasSuffix(term, "*") {
					if strings.HasPrefix(token, strings.TrimSuffix(term, "*")) {
						return true
					}
				} else if token == term {
					return true
				}
			}
		}
		return false
	}
	roles := map[string]bool{}
	for _, v := range []struct {
		k     string
		terms []string
	}{{"backend", []string{"backend", "back-end", "бэкенд*", "бекенд*", "бэкэнд*", "сервер*"}}, {"frontend", []string{"frontend", "front-end", "фронтенд*", "фронтэнд*"}}, {"analytics", []string{"аналитик*", "analytics", "analyst"}}} {
		if matches(v.terms) {
			roles[v.k] = true
		}
	}
	if len(roles) == 1 {
		for key := range roles {
			spec = key
		}
	}
	levels := map[string]bool{}
	for _, v := range []struct {
		k     string
		terms []string
	}{{"junior", []string{"junior", "джуниор", "джун", "начинающ*"}}, {"middle", []string{"middle", "мидл", "миддл"}}, {"senior", []string{"senior", "сеньор", "синьор", "ведущий"}}} {
		if matches(v.terms) {
			levels[v.k] = true
		}
	}
	if len(levels) == 1 {
		for key := range levels {
			grade = key
		}
	}
	known := map[string]bool{}
	for _, s := range []string{"go", "postgresql", "redis", "python", "sql", "react", "typescript", "javascript", "docker", "kubernetes", "excel", "figma", "css", "html", "node.js", "vue", "angular", "java", "kotlin", "rust", "c#", "c++", ".net", "mongodb", "mysql", "clickhouse", "kafka", "rabbitmq", "graphql", "git", "linux", "pandas", "numpy", "spark", "airflow", "powerbi", "tableau", "dbt", "fastapi", "django", "spring"} {
		known[s] = true
	}
	seen := map[string]bool{}
	for i, w := range tokens {
		w = strings.TrimSuffix(w, ".")
		w = Normalize(w)
		if i > 0 && (tokens[i-1] == "без" || tokens[i-1] == "without") {
			continue
		}
		if known[w] && !seen[w] {
			skills = append(skills, w)
			seen[w] = true
		}
	}
	sort.Strings(skills)
	return
}
