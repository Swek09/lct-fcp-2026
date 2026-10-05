package matching

import (
	"reflect"
	"testing"
)

func TestRankEvidenceAndBound(t *testing.T) {
	high := Rank(Evidence{TestScore: 100, Skills: []string{"Go", "PostgreSQL"}, Achievements: 3}, []string{"golang", "postgres"})
	if high.Total != 100 {
		t.Fatalf("expected 100, got %d", high.Total)
	}
	duplicate := Rank(Evidence{TestScore: 100, Skills: []string{"Go", "PostgreSQL"}, Achievements: 100}, []string{"golang", "go", "postgres"})
	if duplicate.Total != 100 {
		t.Fatal("duplicates or excess achievements inflated score")
	}
	plain := Rank(Evidence{TestScore: 100, Skills: []string{"Go"}, Achievements: 0}, []string{"go"})
	if plain.Total != 90 {
		t.Fatal("candidate without FSP should keep test and stack points")
	}
	weak := Rank(Evidence{TestScore: 50, Skills: []string{"Python"}, Achievements: 3}, []string{"go"})
	if plain.Total <= weak.Total {
		t.Fatal("achievements dominate qualification")
	}
}
func TestRequestedStackOutranksUnrelatedAchievements(t *testing.T) {
	python := Rank(Evidence{TestScore: 84, Skills: []string{"Python", "Docker"}}, []string{"Python", "Docker"})
	goCandidate := Rank(Evidence{TestScore: 92, Skills: []string{"Go", "Docker"}, Achievements: 2}, []string{"Python", "Docker"})
	if python.Total <= goCandidate.Total {
		t.Fatal("achievements overrode requested stack")
	}
}
func TestParseNeed(t *testing.T) {
	s, g, skills := ParseNeed("Нужен middle backend на Golang, Postgres и Redis")
	if s != "backend" || g != "middle" || !reflect.DeepEqual(skills, []string{"go", "postgresql", "redis"}) {
		t.Fatalf("unexpected parse: %s %s %v", s, g, skills)
	}
}
func TestNeedAliasesAmbiguityAndWordBoundaries(t *testing.T) {
	for _, c := range []struct {
		text, spec, grade string
		skills            []string
	}{
		{"Миддл бэкендер: Golang, Postgres, k8s", "backend", "middle", []string{"go", "kubernetes", "postgresql"}},
		{"Ведущий фронтенд-разработчик React.js TS Vue.js", "frontend", "senior", []string{"react", "typescript", "vue"}},
		{"Джун аналитик SQL Python", "analytics", "junior", []string{"python", "sql"}},
		{"middle frontend/backend Go React", "", "middle", []string{"go", "react"}},
		{"middle или senior backend Go", "backend", "", []string{"go"}},
		{"Django backend без Redis", "backend", "", []string{"django"}},
		{"Google developer JavaScript", "", "", []string{"javascript"}},
	} {
		s, g, skills := ParseNeed(c.text)
		if s != c.spec || g != c.grade || !reflect.DeepEqual(skills, c.skills) {
			t.Errorf("%q -> %s/%s %v", c.text, s, g, skills)
		}
	}
}
func TestScoreBreakdownExplainsExactTotal(t *testing.T) {
	for _, wanted := range [][]string{nil, {"go", "redis"}, {"python"}} {
		for score := 0; score <= 100; score++ {
			for achievements := 0; achievements <= 4; achievements++ {
				s := Rank(Evidence{TestScore: score, Skills: []string{"Go"}, Achievements: achievements}, wanted)
				if s.Total != s.Test+s.Stack+s.FSP || s.Total < 0 || s.Total > 100 {
					t.Fatalf("invalid breakdown: %+v", s)
				}
			}
		}
	}
}
