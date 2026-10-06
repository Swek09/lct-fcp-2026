package app

import (
	"context"
	"encoding/json"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"strings"
	"time"
)

type Invitation struct {
	ID                   string    `json:"id"`
	EmployerID           string    `json:"employer_id"`
	CandidateID          string    `json:"candidate_id"`
	CandidateName        string    `json:"candidate_name"`
	CompanyName          string    `json:"company_name"`
	CompanyID            string    `json:"company_id"`
	CompanyAvatarVersion int64     `json:"company_avatar_version"`
	CompanyEmail         string    `json:"company_email"`
	CandidateEmail       string    `json:"candidate_email,omitempty"`
	Title                string    `json:"title"`
	Message              string    `json:"message"`
	SalaryMin            int       `json:"salary_min"`
	SalaryMax            int       `json:"salary_max"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
}

func (a *App) invitationRoutes(api huma.API) {
	register(api, "GET", "/api/invitations", "Мои приглашения и статусы", func(ctx context.Context, in *struct{}) (*Response[[]Invitation], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		rows, err := a.DB.Query(ctx, `SELECT row_to_json(t) FROM (
   SELECT i.*,c.name AS company_name,c.user_id AS company_id,c.avatar_version AS company_avatar_version,e.email AS company_email,u.name AS candidate_name,
    CASE WHEN i.status='accepted' AND i.employer_id=$1 THEN u.email ELSE '' END AS candidate_email
   FROM invitations i JOIN companies c ON c.user_id=i.employer_id
   JOIN users e ON e.id=i.employer_id JOIN users u ON u.id=i.candidate_id
   WHERE i.employer_id=$1 OR i.candidate_id=$1 ORDER BY i.created_at DESC
  ) t`, u.ID)
		if err != nil {
			return nil, internal(err)
		}
		defer rows.Close()
		out := []Invitation{}
		for rows.Next() {
			var raw []byte
			if err = rows.Scan(&raw); err != nil {
				return nil, internal(err)
			}
			var v Invitation
			if err = json.Unmarshal(raw, &v); err != nil {
				return nil, internal(err)
			}
			out = append(out, v)
		}
		if err = rows.Err(); err != nil {
			return nil, internal(err)
		}
		return ok(out), nil
	})
	type Create struct {
		Body struct {
			CandidateID string `json:"candidate_id"`
			Title       string `json:"title" minLength:"3" maxLength:"160"`
			Message     string `json:"message" minLength:"20" maxLength:"4000"`
			SalaryMin   int    `json:"salary_min" minimum:"1" maximum:"10000000"`
			SalaryMax   int    `json:"salary_max" minimum:"1" maximum:"10000000"`
		}
	}
	register(api, "POST", "/api/invitations", "Пригласить кандидата с зарплатным предложением", func(ctx context.Context, in *Create) (*Response[Message], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		if err = a.limit(ctx, "invitations:"+u.ID, 30, time.Hour); err != nil {
			return nil, err
		}
		b := in.Body
		if b.SalaryMax < b.SalaryMin {
			return nil, huma.Error400BadRequest("Верхняя граница зарплаты меньше нижней")
		}
		if len(strings.TrimSpace(b.Message)) < 20 || len(strings.TrimSpace(b.Title)) < 3 {
			return nil, huma.Error400BadRequest("Опишите предложение подробнее")
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		// Per-employer lock makes duplicate prevention safe under concurrent requests.
		var companyName string
		err = tx.QueryRow(ctx, "SELECT name FROM companies WHERE user_id=$1 FOR UPDATE", u.ID).Scan(&companyName)
		if err != nil {
			return nil, internal(err)
		}
		if len(strings.TrimSpace(companyName)) < 2 {
			return nil, huma.Error400BadRequest("Сначала заполните название компании")
		}
		var visible bool
		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM profiles p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1 AND p.published AND p.grade<>'' AND u.verified)", b.CandidateID).Scan(&visible)
		if err != nil {
			return nil, internal(err)
		}
		if !visible {
			return nil, huma.Error404NotFound("Кандидат недоступен для приглашений")
		}
		var exists bool
		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM invitations WHERE employer_id=$1 AND candidate_id=$2 AND status IN ('sent','viewed'))", u.ID, b.CandidateID).Scan(&exists)
		if err != nil {
			return nil, internal(err)
		}
		if exists {
			return nil, huma.Error409Conflict("У вас уже есть активное приглашение этому кандидату")
		}
		_, err = tx.Exec(ctx, "INSERT INTO invitations(id,employer_id,candidate_id,title,message,salary_min,salary_max) VALUES($1,$2,$3,$4,$5,$6,$7)", uuid.NewString(), u.ID, b.CandidateID, b.Title, b.Message, b.SalaryMin, b.SalaryMax)
		if err != nil {
			return nil, internal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Приглашение отправлено"), nil
	})
	type Action struct {
		ID   string `path:"id"`
		Body struct {
			Status string `json:"status" enum:"viewed,accepted,declined"`
		}
	}
	register(api, "PATCH", "/api/invitations/{id}", "Просмотреть, принять или отклонить приглашение", func(ctx context.Context, in *Action) (*Response[Message], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		tag, err := a.DB.Exec(ctx, "UPDATE invitations SET status=$3,updated_at=now() WHERE id=$1 AND candidate_id=$2 AND (status IN ('sent','viewed') OR status=$3)", in.ID, u.ID, in.Body.Status)
		if err != nil {
			return nil, internal(err)
		}
		if tag.RowsAffected() == 0 {
			return nil, huma.Error409Conflict("Приглашение недоступно или решение уже принято")
		}
		return msg("Статус обновлён"), nil
	})
}
