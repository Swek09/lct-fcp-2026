// Evaluate the parser and ranker on manually labelled synthetic fixtures.
// This checks the matching algorithm in isolation, without HTTP or database access.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"

	"github.com/Swek09/lct-fcp-2026/backend/internal/matching"
)

type person struct {
	Name           string   `json:"name"`
	Specialization string   `json:"specialization"`
	Grade          string   `json:"grade"`
	Score          int      `json:"score"`
	Skills         []string `json:"skills"`
	Achievements   int      `json:"achievements"`
}
type query struct {
	Need     string   `json:"need"`
	K        int      `json:"k"`
	Relevant []string `json:"relevant"`
}
type ranked struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}
type result struct {
	query
	Top       []ranked `json:"top"`
	Total     int      `json:"total"`
	Precision float64  `json:"precision_at_k"`
}

func main() {
	file := "../docs/matching-dataset.json"
	if len(os.Args) > 1 {
		file = os.Args[1]
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	var dataset struct {
		Candidates []person `json:"candidates"`
		Cases      []query  `json:"cases"`
	}
	if err = json.Unmarshal(raw, &dataset); err != nil {
		panic(err)
	}
	results := []result{}
	mean := 0.0
	for _, q := range dataset.Cases {
		if q.K < 1 {
			panic("k must be positive")
		}
		spec, grade, skills := matching.ParseNeed(q.Need)
		rows := []ranked{}
		for _, p := range dataset.Candidates {
			if spec != "" && p.Specialization != spec || grade != "" && p.Grade != grade {
				continue
			}
			score := matching.Rank(matching.Evidence{TestScore: p.Score, Skills: p.Skills, Achievements: p.Achievements}, skills)
			rows = append(rows, ranked{p.Name, score.Total})
		}
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Score > rows[j].Score })
		top := rows[:min(q.K, len(rows))]
		hits := 0
		for _, r := range top {
			if slices.Contains(q.Relevant, r.Name) {
				hits++
			}
		}
		precision := float64(hits) / float64(q.K)
		mean += precision
		results = append(results, result{q, top, len(rows), precision})
	}
	if len(results) == 0 {
		panic("empty dataset")
	}
	out := map[string]any{
		"dataset":             fmt.Sprintf("%d synthetic candidates, %d manually labelled descriptions", len(dataset.Candidates), len(dataset.Cases)),
		"mode":                "offline parser and ranker; no HTTP/database validation in this experiment",
		"limitations":         "Same fixtures used for development. Small category groups; no independent expert or user validation. The HTTP API is covered separately by integration tests.",
		"mean_precision_at_k": mean / float64(len(results)), "results": results,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err = enc.Encode(out); err != nil {
		panic(err)
	}
}
