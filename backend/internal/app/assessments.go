package app

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"github.com/Swek09/lct-fcp-2026/backend/internal/assessment"
	"github.com/Swek09/lct-fcp-2026/backend/internal/matching"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"time"
)

type Attempt struct {
	ID              string                `json:"id"`
	Specialization  string                `json:"specialization"`
	Grade           string                `json:"grade"`
	TemplateVersion string                `json:"template_version"`
	Questions       []assessment.Question `json:"questions"`
	StartedAt       time.Time             `json:"started_at"`
	ExpiresAt       time.Time             `json:"expires_at"`
	SubmittedAt     *time.Time            `json:"submitted_at"`
	Score           *int                  `json:"score"`
	Passed          *bool                 `json:"passed"`
}
type storedAttempt struct {
	ID              string                       `json:"id"`
	UserID          string                       `json:"user_id"`
	Specialization  string                       `json:"specialization"`
	Grade           string                       `json:"grade"`
	TemplateVersion string                       `json:"template_version"`
	Questions       []assessment.PrivateQuestion `json:"questions"`
	StartedAt       time.Time                    `json:"started_at"`
	ExpiresAt       time.Time                    `json:"expires_at"`
	SubmittedAt     *time.Time                   `json:"submitted_at"`
	Score           *int                         `json:"score"`
	Passed          *bool                        `json:"passed"`
}

func (s storedAttempt) public() Attempt {
	return Attempt{s.ID, s.Specialization, s.Grade, s.TemplateVersion, assessment.Public(s.Questions), s.StartedAt, s.ExpiresAt, s.SubmittedAt, s.Score, s.Passed}
}
func (a *App) assessmentRoutes(api huma.API) {
	type PreviewInput struct {
		Body struct {
			Need string `json:"need" minLength:"3" maxLength:"2000"`
		}
	}
	type Preview struct {
		Specialization  string                `json:"specialization"`
		Grade           string                `json:"grade"`
		Skills          []string              `json:"skills"`
		Questions       []assessment.Question `json:"questions"`
		TemplateVersion string                `json:"template_version"`
	}
	register(api, "POST", "/api/assessments/preview", "Образец теста по описанию потребности работодателя", func(ctx context.Context, in *PreviewInput) (*Response[Preview], error) {
		if _, err := a.require(ctx, "employer"); err != nil {
			return nil, err
		}
		spec, grade, skills := matching.ParseNeed(in.Body.Need)
		if !assessment.Valid(spec, grade) {
			return nil, huma.Error400BadRequest("Укажите в описании специализацию и грейд: например, middle backend Go PostgreSQL")
		}
		var seed [8]byte
		if _, err := rand.Read(seed[:]); err != nil {
			return nil, internal(err)
		}
		return ok(Preview{spec, grade, skills, assessment.Public(assessment.Generate(spec, grade, binary.LittleEndian.Uint64(seed[:]))), assessment.Version}), nil
	})
	register(api, "GET", "/api/assessments", "История и текущий тест", func(ctx context.Context, in *struct{}) (*Response[[]Attempt], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		rows, err := a.DB.Query(ctx, "SELECT row_to_json(t) FROM (SELECT * FROM assessments WHERE user_id=$1 ORDER BY started_at DESC LIMIT 30) t", u.ID)
		if err != nil {
			return nil, internal(err)
		}
		defer rows.Close()
		out := []Attempt{}
		for rows.Next() {
			var raw []byte
			if err = rows.Scan(&raw); err != nil {
				return nil, internal(err)
			}
			var s storedAttempt
			if err = json.Unmarshal(raw, &s); err != nil {
				return nil, internal(err)
			}
			out = append(out, s.public())
		}
		if err = rows.Err(); err != nil {
			return nil, internal(err)
		}
		return ok(out), nil
	})
	type Start struct {
		Body struct {
			Specialization string `json:"specialization" enum:"backend,frontend,analytics"`
			Grade          string `json:"grade" enum:"junior,middle,senior"`
		}
	}
	register(api, "POST", "/api/assessments", "Начать тест выбранного уровня", func(ctx context.Context, in *Start) (*Response[Attempt], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		var next *time.Time
		err = tx.QueryRow(ctx, "SELECT next_grade_change_at FROM profiles WHERE user_id=$1 FOR UPDATE", u.ID).Scan(&next)
		if err != nil {
			return nil, internal(err)
		}
		// Lock the candidate profile to serialize starts and grade changes.
		if next != nil && next.After(time.Now()) {
			return nil, huma.Error409Conflict("Следующая смена категории доступна " + next.Format("02.01.2006"))
		}
		_, err = tx.Exec(ctx, "UPDATE assessments SET submitted_at=now(),score=0,passed=false,answers='{}' WHERE user_id=$1 AND submitted_at IS NULL AND expires_at<=now()", u.ID)
		if err != nil {
			return nil, internal(err)
		}
		var active bool
		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assessments WHERE user_id=$1 AND submitted_at IS NULL)", u.ID).Scan(&active)
		if err != nil {
			return nil, internal(err)
		}
		if active {
			return nil, huma.Error409Conflict("Сначала завершите текущий тест")
		}
		var recent bool
		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assessments WHERE user_id=$1 AND specialization=$2 AND grade=$3 AND started_at>now()-interval '24 hours')", u.ID, in.Body.Specialization, in.Body.Grade).Scan(&recent)
		if err != nil {
			return nil, internal(err)
		}
		if recent {
			return nil, huma.Error409Conflict("Повторный тест этого уровня доступен через 24 часа")
		}
		var seed [8]byte
		if _, err = rand.Read(seed[:]); err != nil {
			return nil, internal(err)
		}
		qs := assessment.Generate(in.Body.Specialization, in.Body.Grade, binary.LittleEndian.Uint64(seed[:]))
		raw, _ := json.Marshal(qs)
		id := uuid.NewString()
		stored, err := readJSON[storedAttempt](ctx, tx, "WITH x AS (INSERT INTO assessments(id,user_id,specialization,grade,template_version,questions,expires_at) VALUES($1,$2,$3,$4,$5,$6,now()+interval '20 minutes') RETURNING *) SELECT row_to_json(x) FROM x", id, u.ID, in.Body.Specialization, in.Body.Grade, assessment.Version, raw)
		if err != nil {
			return nil, internal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return ok(stored.public()), nil
	})
	type Submit struct {
		ID   string `path:"id"`
		Body struct {
			Answers map[string]string `json:"answers"`
		}
	}
	type Result struct {
		Attempt Attempt         `json:"attempt"`
		Topics  map[string]bool `json:"topics"`
		Message string          `json:"message"`
	}
	register(api, "POST", "/api/assessments/{id}/submit", "Отправить ответы на проверку", func(ctx context.Context, in *Submit) (*Response[Result], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		var lockID string
		err = tx.QueryRow(ctx, "SELECT user_id FROM profiles WHERE user_id=$1 FOR UPDATE", u.ID).Scan(&lockID)
		if err != nil {
			return nil, internal(err)
		}
		s, err := readJSON[storedAttempt](ctx, tx, "SELECT row_to_json(t) FROM (SELECT * FROM assessments WHERE id=$1 AND user_id=$2 FOR UPDATE) t", in.ID, u.ID)
		if isMissing(err) {
			return nil, huma.Error404NotFound("Тест не найден")
		}
		if err != nil {
			return nil, internal(err)
		}
		if s.SubmittedAt != nil {
			return nil, huma.Error409Conflict("Ответы уже отправлены")
		}
		score, topics := assessment.Score(s.Questions, in.Body.Answers)
		expired := time.Now().After(s.ExpiresAt)
		if expired {
			score = 0
		}
		passed := score >= assessment.PassScore && !expired
		raw, _ := json.Marshal(in.Body.Answers)
		_, err = tx.Exec(ctx, "UPDATE assessments SET answers=$2,score=$3,passed=$4,submitted_at=now() WHERE id=$1", s.ID, raw, score, passed)
		if err != nil {
			return nil, internal(err)
		}
		if passed {
			_, err = tx.Exec(ctx, "UPDATE profiles SET last_practice_at=CASE WHEN specialization=$2 AND grade=$3 THEN last_practice_at ELSE NULL END,specialization=$2,grade=$3,test_score=$4,grade_changed_at=now(),next_grade_change_at=now()+interval '90 days',updated_at=now() WHERE user_id=$1", u.ID, s.Specialization, s.Grade, score)
			if err != nil {
				return nil, internal(err)
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		now := time.Now()
		s.Score = &score
		s.Passed = &passed
		s.SubmittedAt = &now
		message := "Уровень пока не подтверждён. Можно попробовать другой уровень или повторить этот через 24 часа."
		if passed {
			message = "Уровень подтверждён. Разрешите публикацию профиля, чтобы работодатели могли вас найти."
		}
		if expired {
			message = "Время тестирования истекло. Ответы не засчитаны."
		}
		return ok(Result{s.public(), topics, message}), nil
	})
}
