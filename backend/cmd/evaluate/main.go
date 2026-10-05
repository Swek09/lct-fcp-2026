// Evaluate reports reproducible mechanical checks on synthetic test variants.
// It does not establish psychometric validity or expert agreement on grades.
package main

import (
	"encoding/json"
	"github.com/Swek09/lct-fcp-2026/backend/internal/assessment"
	"os"
)

type result struct {
	Category          string  `json:"category"`
	Variants          int     `json:"variants"`
	UniqueFacts       int     `json:"unique_facts"`
	CopiedKeyPasses   int     `json:"copied_key_passes"`
	CorrectKeyPasses  int     `json:"correct_key_passes"`
	CopiedKeyPassRate float64 `json:"copied_key_pass_rate"`
	TaskFamilies      int     `json:"task_families"`
}

func main() {
	out := []result{}
	for _, spec := range []string{"backend", "frontend", "analytics"} {
		for _, grade := range []string{"junior", "middle", "senior"} {
			base := assessment.Generate(spec, grade, 1)
			key := map[string]string{}
			for _, q := range base {
				key[q.ID] = q.Expected
			}
			unique := map[string]bool{}
			families := map[string]bool{}
			copied, correct := 0, 0
			for seed := uint64(2); seed <= 101; seed++ {
				qs := assessment.Generate(spec, grade, seed)
				facts := []string{}
				own := map[string]string{}
				for _, q := range qs {
					families[q.TemplateID] = true
					facts = append(facts, q.Prompt)
					own[q.ID] = q.Expected
				}
				raw, _ := json.Marshal(facts)
				unique[string(raw)] = true
				if score, _ := assessment.Score(qs, key); score >= assessment.PassScore {
					copied++
				}
				if score, _ := assessment.Score(qs, own); score >= assessment.PassScore {
					correct++
				}
			}
			out = append(out, result{spec + "/" + grade, 100, len(unique), copied, correct, float64(copied) / 100, len(families)})
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"template_version": assessment.Version, "questions_per_attempt": assessment.QuestionCount, "pass_score": assessment.PassScore, "experiment": "100 new variants per category; copy the answer key of seed=1 without solving new facts", "limitations": "Synthetic mechanical validation. Not evidence of real candidate grade accuracy or resistance to LLM assistance.", "results": out})
}
