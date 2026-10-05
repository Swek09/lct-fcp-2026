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
