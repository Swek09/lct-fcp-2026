package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type ChallengeSubmission struct {
	ID            string     `json:"id"`
	CandidateID   string     `json:"candidate_id"`
	CandidateName string     `json:"candidate_name"`
	Answer        string     `json:"answer"`
	Status        string     `json:"status"`
	Feedback      string     `json:"feedback"`
	SubmittedAt   time.Time  `json:"submitted_at"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
}

type Challenge struct {
	ID               string               `json:"id"`
	Title            string               `json:"title"`
	Body             string               `json:"body"`
	CompanyName      string               `json:"company_name"`
	Specialization   string               `json:"specialization"`
	Grade            string               `json:"grade"`
	EstimatedMinutes int                  `json:"estimated_minutes"`
	CreatedAt        time.Time            `json:"created_at"`
	ClosesAt         time.Time            `json:"closes_at"`
	SubmissionCount  int                  `json:"submission_count"`
	Submission       *ChallengeSubmission `json:"submission"`
}

type ChallengeFeed struct {
	Challenges       []Challenge `json:"challenges"`
	NextSubmissionAt *time.Time  `json:"next_submission_at"`
}

func (a *App) challengeRoutes(api huma.API) {
	register(api, "GET", "/api/challenges", "Короткие задания моей категории или компании", func(ctx context.Context, in *struct{}) (*Response[ChallengeFeed], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		spec, grade := "", ""
		var next *time.Time
		if u.Role == "candidate" {
			if err = a.DB.QueryRow(ctx, "SELECT specialization,grade FROM profiles WHERE user_id=$1", u.ID).Scan(&spec, &grade); err != nil {
				return nil, internal(err)
			}
			if err = a.DB.QueryRow(ctx, "SELECT max(submitted_at)+interval '7 days' FROM challenge_submissions WHERE candidate_id=$1", u.ID).Scan(&next); err != nil {
				return nil, internal(err)
			}
		}
		rows, err := a.DB.Query(ctx, `SELECT row_to_json(t) FROM (
 SELECT c.*,co.name AS company_name,
 (SELECT count(*) FROM challenge_submissions s WHERE s.challenge_id=c.id AND $2='employer') AS submission_count,
 (SELECT row_to_json(x) FROM (SELECT s.*,u.name AS candidate_name FROM challenge_submissions s JOIN users u ON u.id=s.candidate_id WHERE s.challenge_id=c.id AND s.candidate_id=$1) x) AS submission
 FROM challenges c JOIN companies co ON co.user_id=c.employer_id
 WHERE ($2='employer' AND c.employer_id=$1) OR ($2='candidate' AND (
 (c.specialization=$3 AND c.grade=$4 AND c.closes_at>now()) OR EXISTS(SELECT 1 FROM challenge_submissions s WHERE s.challenge_id=c.id AND s.candidate_id=$1)))
 ORDER BY c.created_at DESC,c.id LIMIT 100) t`, u.ID, u.Role, spec, grade)
		if err != nil {
			return nil, internal(err)
		}
		defer rows.Close()
		items := []Challenge{}
		for rows.Next() {
			var raw []byte
			var c Challenge
			if err = rows.Scan(&raw); err != nil {
				return nil, internal(err)
			}
			if err = json.Unmarshal(raw, &c); err != nil {
				return nil, internal(err)
			}
			items = append(items, c)
		}
		if err = rows.Err(); err != nil {
			return nil, internal(err)
		}
		return ok(ChallengeFeed{items, next}), nil
	})
	type Create struct {
		Body struct {
			Title            string `json:"title" minLength:"4" maxLength:"150"`
			Body             string `json:"body" minLength:"40" maxLength:"6000"`
			Specialization   string `json:"specialization" enum:"backend,frontend,analytics"`
			Grade            string `json:"grade" enum:"junior,middle,senior"`
			EstimatedMinutes int    `json:"estimated_minutes" minimum:"5" maximum:"30" default:"10"`
		}
	}
	register(api, "POST", "/api/challenges", "Предложить категории короткую задачу на неделю", func(ctx context.Context, in *Create) (*Response[Message], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		b := in.Body
		b.Title, b.Body = strings.TrimSpace(b.Title), strings.TrimSpace(b.Body)
		if utf8.RuneCountInString(b.Title) < 4 || utf8.RuneCountInString(b.Body) < 40 {
			return nil, huma.Error400BadRequest("Опишите условие и ожидаемый результат подробнее")
		}
		var company string
		if err = a.DB.QueryRow(ctx, "SELECT name FROM companies WHERE user_id=$1", u.ID).Scan(&company); err != nil {
			return nil, internal(err)
		}
		if utf8.RuneCountInString(strings.TrimSpace(company)) < 2 {
			return nil, huma.Error400BadRequest("Сначала заполните профиль компании")
		}
		if err = a.limit(ctx, "challenges:"+u.ID, 10, 24*time.Hour); err != nil {
			return nil, err
		}
		_, err = a.DB.Exec(ctx, "INSERT INTO challenges(id,employer_id,title,body,specialization,grade,estimated_minutes) VALUES($1,$2,$3,$4,$5,$6,$7)", uuid.NewString(), u.ID, b.Title, b.Body, b.Specialization, b.Grade, b.EstimatedMinutes)
		if err != nil {
			return nil, internal(err)
		}
		return msg("Задание опубликовано на 7 дней"), nil
	})
	type Submit struct {
		ID   string `path:"id"`
		Body struct {
			Answer string `json:"answer" minLength:"40" maxLength:"6000"`
		}
	}
	register(api, "POST", "/api/challenges/{id}/submissions", "Отправить решение короткой задачи", func(ctx context.Context, in *Submit) (*Response[Message], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		answer := strings.TrimSpace(in.Body.Answer)
		if utf8.RuneCountInString(answer) < 40 {
			return nil, huma.Error400BadRequest("Опишите решение или подход подробнее")
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		// Serialize submissions for this candidate, including different tasks.
		var spec, grade string
		if err = tx.QueryRow(ctx, "SELECT specialization,grade FROM profiles WHERE user_id=$1 FOR UPDATE", u.ID).Scan(&spec, &grade); err != nil {
			return nil, internal(err)
		}
		var eligible bool
		if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM challenges WHERE id=$1 AND specialization=$2 AND grade=$3 AND closes_at>now())", in.ID, spec, grade).Scan(&eligible); err != nil {
			return nil, internal(err)
		}
		if !eligible {
			return nil, huma.Error404NotFound("Задание закрыто или не относится к вашей подтверждённой категории")
		}
		var recent bool
		if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM challenge_submissions WHERE candidate_id=$1 AND (challenge_id=$2 OR submitted_at>now()-interval '7 days'))", u.ID, in.ID).Scan(&recent); err != nil {
			return nil, internal(err)
		}
		if recent {
			return nil, huma.Error409Conflict("Решение уже отправлено. Следующая короткая задача доступна через 7 дней после отправки")
		}
		_, err = tx.Exec(ctx, "INSERT INTO challenge_submissions(id,challenge_id,candidate_id,answer) VALUES($1,$2,$3,$4)", uuid.NewString(), in.ID, u.ID, answer)
		if err != nil {
			return nil, internal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Решение отправлено. Работодатель сможет дать обратную связь"), nil
	})
	type TaskID struct {
		ID string `path:"id"`
	}
	register(api, "GET", "/api/challenges/{id}/submissions", "Решения кандидатов для моего задания", func(ctx context.Context, in *TaskID) (*Response[[]ChallengeSubmission], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		var owns bool
		if err = a.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM challenges WHERE id=$1 AND employer_id=$2)", in.ID, u.ID).Scan(&owns); err != nil {
			return nil, internal(err)
		}
		if !owns {
			return nil, huma.Error404NotFound("Задание не найдено")
		}
		rows, err := a.DB.Query(ctx, "SELECT row_to_json(t) FROM (SELECT s.*,u.name AS candidate_name FROM challenge_submissions s JOIN users u ON u.id=s.candidate_id WHERE s.challenge_id=$1 ORDER BY s.submitted_at DESC LIMIT 300) t", in.ID)
		if err != nil {
			return nil, internal(err)
		}
		defer rows.Close()
		items := []ChallengeSubmission{}
		for rows.Next() {
			var raw []byte
			var s ChallengeSubmission
			if err = rows.Scan(&raw); err != nil {
				return nil, internal(err)
			}
			if err = json.Unmarshal(raw, &s); err != nil {
				return nil, internal(err)
			}
			items = append(items, s)
		}
		if err = rows.Err(); err != nil {
			return nil, internal(err)
		}
		return ok(items), nil
	})
	type Review struct {
		ID   string `path:"id"`
		Body struct {
			Status   string `json:"status" enum:"accepted,needs_work"`
			Feedback string `json:"feedback" minLength:"10" maxLength:"2000"`
		}
	}
	register(api, "PATCH", "/api/challenge-submissions/{id}", "Дать обратную связь по решению моей задачи", func(ctx context.Context, in *Review) (*Response[Message], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		feedback := strings.TrimSpace(in.Body.Feedback)
		if utf8.RuneCountInString(feedback) < 10 {
			return nil, huma.Error400BadRequest("Напишите полезную обратную связь")
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		var candidateID string
		err = tx.QueryRow(ctx, `UPDATE challenge_submissions s SET status=$3,feedback=$4,reviewed_at=now()
 FROM challenges c WHERE s.id=$1 AND c.id=s.challenge_id AND c.employer_id=$2 AND s.status='submitted' RETURNING s.candidate_id`, in.ID, u.ID, in.Body.Status, feedback).Scan(&candidateID)
		if isMissing(err) {
			return nil, huma.Error409Conflict("Решение недоступно или уже проверено")
		}
		if err != nil {
			return nil, internal(err)
		}
		if in.Body.Status == "accepted" {
			if _, err = tx.Exec(ctx, `UPDATE profiles p SET last_practice_at=greatest(p.last_practice_at,s.submitted_at)
 FROM challenge_submissions s JOIN challenges c ON c.id=s.challenge_id
 WHERE p.user_id=$1 AND s.id=$2 AND p.specialization=c.specialization AND p.grade=c.grade`, candidateID, in.ID); err != nil {
				return nil, internal(err)
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Обратная связь отправлена"), nil
	})
}
