package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Swek09/lct-fcp-2026/backend/internal/assessment"
	"github.com/Swek09/lct-fcp-2026/backend/internal/matching"
	"github.com/danielgtaylor/huma/v2"
)

// Shared policy: self-applications disclose contacts only to the vacancy owner.
// Withdrawal revokes this grant, without revoking a separately accepted invite.
const contactGrant = `EXISTS(SELECT 1 FROM invitations i WHERE i.candidate_id=p.user_id AND i.employer_id=$2 AND i.status='accepted')
 OR EXISTS(SELECT 1 FROM applications x JOIN vacancies v ON v.id=x.vacancy_id WHERE x.candidate_id=p.user_id AND v.employer_id=$2 AND x.status<>'withdrawn')`
const candidateAccess = `(p.published AND p.grade<>'' AND u.verified) OR (` + contactGrant + `)`

type TopicEvidence struct {
	Topic  string `json:"topic"`
	Passed bool   `json:"passed"`
}
type QualificationEvidence struct {
	Source            string          `json:"source"`
	Specialization    string          `json:"specialization"`
	Grade             string          `json:"grade"`
	Score             int             `json:"score"`
	ConfirmedAt       *time.Time      `json:"confirmed_at"`
	Version           string          `json:"version"`
	Topics            []TopicEvidence `json:"topics"`
	NextGradeChangeAt *time.Time      `json:"next_grade_change_at"`
	LastPracticeAt    *time.Time      `json:"last_practice_at"`
}

func (a *App) evidence(ctx context.Context, id string) (QualificationEvidence, error) {
	var e QualificationEvidence
	var email string
	err := a.DB.QueryRow(ctx, `SELECT p.specialization,p.grade,p.test_score,p.grade_changed_at,p.next_grade_change_at,p.last_practice_at,u.email FROM profiles p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1`, id).Scan(&e.Specialization, &e.Grade, &e.Score, &e.ConfirmedAt, &e.NextGradeChangeAt, &e.LastPracticeAt, &email)
	e.Topics = []TopicEvidence{}
	e.Source = "unconfirmed"
	if err != nil {
		return e, err
	}
	if e.Grade == "" {
		return e, nil
	}
	var questions, answers []byte
	var at time.Time
	err = a.DB.QueryRow(ctx, `SELECT questions,answers,template_version,submitted_at FROM assessments WHERE user_id=$1 AND passed AND specialization=$2 AND grade=$3 AND score=$4 ORDER BY submitted_at DESC LIMIT 1`, id, e.Specialization, e.Grade, e.Score).Scan(&questions, &answers, &e.Version, &at)
	if isMissing(err) {
		// Imported demo scores are labelled, never presented as an actual attempt.
		if a.Config.Demo && DemoID(email) == id {
			e.Source = "demo"
		} else {
			e.Source = "imported"
		}
		return e, nil
	}
	if err != nil {
		return e, err
	}
	var qs []assessment.PrivateQuestion
	var as map[string]string
	if err = json.Unmarshal(questions, &qs); err != nil {
		return e, err
	}
	if err = json.Unmarshal(answers, &as); err != nil {
		return e, err
	}
	_, topics := assessment.Score(qs, as)
	e.Source = "assessment"
	e.ConfirmedAt = &at
	for _, q := range qs {
		e.Topics = append(e.Topics, TopicEvidence{q.Topic, topics[q.Topic]})
	}
	return e, nil
}

func (a *App) evidenceRoutes(api huma.API) {
	register(api, "GET", "/api/profile/evidence", "Мой паспорт квалификации", func(ctx context.Context, in *struct{}) (*Response[QualificationEvidence], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		e, err := a.evidence(ctx, u.ID)
		if err != nil {
			return nil, internal(err)
		}
		return ok(e), nil
	})
	type Detail struct {
		ID string `path:"id"`
	}
	register(api, "GET", "/api/candidates/{id}/evidence", "Проверяемые основания квалификации кандидата", func(ctx context.Context, in *Detail) (*Response[QualificationEvidence], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		var visible bool
		err = a.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM profiles p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1 AND (`+candidateAccess+`))`, in.ID, u.ID).Scan(&visible)
		if err != nil {
			return nil, internal(err)
		}
		if !visible {
			return nil, huma.Error404NotFound("Профиль не найден")
		}
		e, err := a.evidence(ctx, in.ID)
		if err != nil {
			return nil, internal(err)
		}
		return ok(e), nil
	})
	register(api, "GET", "/api/shortlist", "Мой список избранных кандидатов", func(ctx context.Context, in *struct{}) (*Response[[]Candidate], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		rows, err := jsonRows[Candidate](ctx, a, `SELECT row_to_json(t) FROM (SELECT p.*,u.name,CASE WHEN (`+contactGrant+`) THEN u.email ELSE '' END AS email
   FROM shortlist s JOIN profiles p ON p.user_id=s.candidate_id JOIN users u ON u.id=p.user_id WHERE s.employer_id=$1 AND (`+candidateAccess+`) ORDER BY s.created_at DESC) t`, u.ID, u.ID)
		if err != nil {
			return nil, internal(err)
		}
		for i := range rows {
			rank := matching.Rank(matching.Evidence{TestScore: rows[i].TestScore, Skills: rows[i].Skills, Achievements: len(rows[i].Achievements)}, nil)
			rows[i].MatchScore = rank.Total
			rows[i].Reasons = rank.Reasons
			rows[i].Match = rank
		}
		return ok(rows), nil
	})
	register(api, "POST", "/api/shortlist/{id}", "Сохранить кандидата в свой список", func(ctx context.Context, in *Detail) (*Response[Message], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		var locked string
		if err = tx.QueryRow(ctx, "SELECT user_id FROM companies WHERE user_id=$1 FOR UPDATE", u.ID).Scan(&locked); err != nil {
			return nil, internal(err)
		}
		var count int
		if err = tx.QueryRow(ctx, "SELECT count(*) FROM shortlist WHERE employer_id=$1", u.ID).Scan(&count); err != nil {
			return nil, internal(err)
		}
		if count >= 100 {
			return nil, huma.Error409Conflict("В избранном можно сохранить до 100 кандидатов")
		}
		tag, err := tx.Exec(ctx, `INSERT INTO shortlist(employer_id,candidate_id) SELECT $2,p.user_id FROM profiles p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1 AND (`+candidateAccess+`) ON CONFLICT DO NOTHING`, in.ID, u.ID)
		if err != nil {
			return nil, internal(err)
		}
		if tag.RowsAffected() == 0 {
			var exists bool
			err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM shortlist WHERE employer_id=$1 AND candidate_id=$2)", u.ID, in.ID).Scan(&exists)
			if err != nil {
				return nil, internal(err)
			}
			if !exists {
				return nil, huma.Error404NotFound("Кандидат недоступен")
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Кандидат сохранён"), nil
	})
	register(api, "DELETE", "/api/shortlist/{id}", "Удалить кандидата из своего списка", func(ctx context.Context, in *Detail) (*Response[Message], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		_, err = a.DB.Exec(ctx, "DELETE FROM shortlist WHERE employer_id=$1 AND candidate_id=$2", u.ID, in.ID)
		if err != nil {
			return nil, internal(err)
		}
		return msg("Кандидат удалён из списка"), nil
	})
}
