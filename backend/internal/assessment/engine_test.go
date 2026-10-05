package assessment

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestVariantsHaveStableBlueprintAndCorrectScoring(t *testing.T) {
	for _, spec := range []string{"backend", "frontend", "analytics"} {
		for _, grade := range []string{"junior", "middle", "senior"} {
			t.Run(spec+"/"+grade, func(t *testing.T) {
				base := Generate(spec, grade, 1)
				if len(base) != QuestionCount {
					t.Fatal("expected 12 questions")
				}
				unique := map[string]bool{}
				families := map[string]bool{}
				for seed := uint64(1); seed <= 100; seed++ {
					qs := Generate(spec, grade, seed)
					answers := map[string]string{}
					topics, kinds := map[string]int{}, map[string]int{}
					used := map[string]bool{}
					facts := []string{}
					for i, q := range qs {
						if used[q.TemplateID] || q.Prompt == "" || q.Expected == "" || q.ID != fmt.Sprintf("q%d", i+1) {
							t.Fatal("duplicate family, missing content or unstable ID")
						}
						used[q.TemplateID], families[q.TemplateID] = true, true
						topics[q.Topic]++
						kinds[q.Kind]++
						facts = append(facts, q.Prompt)
						if q.Topic != base[i].Topic {
							t.Fatal("topic blueprint drift")
						}
						answers[q.ID] = q.Expected
						if q.Kind == "choice" {
							found := 0
							for _, s := range q.Options {
								if s == q.Expected {
									found++
								}
							}
							if found != 1 {
								t.Fatal("invalid options")
							}
						}
					}
					if len(topics) != 6 || kinds["choice"] != 6 || kinds["number"] != 6 {
						t.Fatal("unbalanced topic/kind blueprint")
					}
					for _, n := range topics {
						if n != 2 {
							t.Fatal("each topic must have two tasks")
						}
					}
					score, _ := Score(qs, answers)
					if score != 100 {
						t.Fatalf("correct answer scored %d", score)
					}
					zero, _ := Score(qs, map[string]string{})
					if zero != 0 {
						t.Fatal("empty answers scored points")
					}
					raw, _ := json.Marshal(facts)
					unique[string(raw)] = true
					public, _ := json.Marshal(Public(qs))
					if strings.Contains(string(public), "\"expected\"") || strings.Contains(string(public), "\"template_id\"") {
						t.Fatal("answer exposed")
					}
				}
				if len(families) != 18 {
					t.Fatalf("reachable task families: %d, want 18", len(families))
				}
				if len(unique) < 90 {
					t.Fatalf("too few unique variants: %d", len(unique))
				}
				if !reflect.DeepEqual(base, Generate(spec, grade, 1)) {
					t.Fatal("seed not reproducible")
				}
			})
		}
	}
}
func TestScoreBoundary(t *testing.T) {
	qs := Generate("backend", "junior", 2)
	answers := map[string]string{}
	for i := 0; i < 8; i++ {
		answers[qs[i].ID] = qs[i].Expected
	}
	s, _ := Score(qs, answers)
	if s >= PassScore {
		t.Fatal("8/12 must fail")
	}
	answers[qs[8].ID] = qs[8].Expected
	s, _ = Score(qs, answers)
	if s < PassScore {
		t.Fatal("9/12 must pass")
	}
}

func TestTopicRequiresBothTasksAndLegacyScoring(t *testing.T) {
	qs := []PrivateQuestion{{Question: Question{ID: "q1", Topic: "SQL", Kind: "number"}, Expected: "1.25"}, {Question: Question{ID: "q2", Topic: "SQL", Kind: "choice"}, Expected: "yes"}}
	s, topics := Score(qs, map[string]string{"q1": " 1,25 ", "q2": "no"})
	if s != 50 || topics["SQL"] {
		t.Fatal("one answer incorrectly confirms the whole topic")
	}
	for _, v := range []string{"NaN", "+Inf", "-Inf", "1.3", ""} {
		if s, _ := Score(qs[:1], map[string]string{"q1": v}); s != 0 {
			t.Fatalf("accepted %q", v)
		}
	}
	// Stored v3 attempts have six private tasks without template IDs.
	legacy := []PrivateQuestion{}
	answers := map[string]string{}
	for i := 0; i < 6; i++ {
		id := fmt.Sprint(i)
		legacy = append(legacy, PrivateQuestion{Question: Question{ID: id, Topic: id, Kind: "choice"}, Expected: "yes"})
		if i < 5 {
			answers[id] = "yes"
		}
	}
	if s, topics := Score(legacy, answers); s != 83 || len(topics) != 6 {
		t.Fatal("legacy stored attempts changed")
	}
	if s, _ := Score(nil, nil); s != 0 {
		t.Fatal("empty exam")
	}
	if len(Generate("unknown", "junior", 1)) != 0 || len(Generate("backend", "lead", 1)) != 0 {
		t.Fatal("invalid category accepted")
	}
}

// Solve tricky new numeric tasks from their public facts, without reading Expected.
func TestExpandedCalculationsFromPublicFacts(t *testing.T) {
	for seed := uint64(1); seed <= 30; seed++ {
		cases := []struct {
			spec, grade  string
			slot, family int
			solve        func(string) float64
		}{
			{"frontend", "middle", 1, 1, func(p string) float64 {
				var width, columns int
				if _, err := fmt.Sscanf(p, "Контейнер имеет ширину %d px, border-box, padding по 20 px с двух сторон, без border. Grid содержит %d", &width, &columns); err != nil {
					t.Fatal(err)
				}
				return float64(width-40-12*(columns-1)) / float64(columns)
			}},
			{"backend", "senior", 5, 1, func(p string) float64 {
				var load int
				if _, err := fmt.Sscanf(p, "Пик %d запросов/с.", &load); err != nil {
					t.Fatal(err)
				}
				return float64((load + 49) / 50)
			}},
			{"analytics", "middle", 1, 2, func(p string) float64 {
				var a, b int
				if _, err := fmt.Sscanf(p, "В когорте A %d пользователей с конверсией 10%%, в B %d", &a, &b); err != nil {
					t.Fatal(err)
				}
				return float64(a+2*b) * 10 / float64(a+b)
			}},
			{"analytics", "senior", 5, 2, func(p string) float64 {
				var n int
				if _, err := fmt.Sscanf(p, "Семейная вероятность ошибки первого рода задана 0,05. Применяем поправку Бонферрони для %d", &n); err != nil {
					t.Fatal(err)
				}
				return 5 / float64(n)
			}},
		}
		for _, c := range cases {
			q := expandedScenario(c.spec, c.grade, c.slot, c.family, rand.New(rand.NewPCG(seed, 99)), "topic")
			answer := strconv.FormatFloat(c.solve(q.Prompt), 'f', 2, 64)
			q.ID = "q"
			if s, _ := Score([]PrivateQuestion{q}, map[string]string{"q": answer}); s != 100 {
				t.Fatalf("wrong numeric key: %s; independently solved %s, stored %s", q.Prompt, answer, q.Expected)
			}
		}
	}
}
