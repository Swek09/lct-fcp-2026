package assessment

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestVariantsHaveStableBlueprintAndCorrectScoring(t *testing.T) {
	for _, spec := range []string{"backend", "frontend", "analytics"} {
		for _, grade := range []string{"junior", "middle", "senior"} {
			t.Run(spec+"/"+grade, func(t *testing.T) {
				base := Generate(spec, grade, 1)
				if len(base) != 6 {
					t.Fatal("expected 6 questions")
				}
				unique := map[string]bool{}
				for seed := uint64(1); seed <= 100; seed++ {
					qs := Generate(spec, grade, seed)
					answers := map[string]string{}
					for i, q := range qs {
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
					score, _ := Score(qs, answers)
					if score != 100 {
						t.Fatalf("correct answer scored %d", score)
					}
					zero, _ := Score(qs, map[string]string{})
					if zero != 0 {
						t.Fatal("empty answers scored points")
					}
					raw, _ := json.Marshal(qs)
					unique[string(raw)] = true
					public, _ := json.Marshal(Public(qs))
					if strings.Contains(string(public), "\"expected\"") {
						t.Fatal("answer exposed")
					}
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
	for i := 0; i < 4; i++ {
		answers[qs[i].ID] = qs[i].Expected
	}
	s, _ := Score(qs, answers)
	if s >= PassScore {
		t.Fatal("4/6 must fail")
	}
	answers[qs[4].ID] = qs[4].Expected
	s, _ = Score(qs, answers)
	if s < PassScore {
		t.Fatal("5/6 must pass")
	}
}
