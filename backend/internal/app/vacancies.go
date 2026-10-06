package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type VacancyBody struct {
	Title          string   `json:"title" minLength:"4" maxLength:"160"`
	Description    string   `json:"description" minLength:"40" maxLength:"6000"`
	Specialization string   `json:"specialization" enum:"backend,frontend,analytics"`
	Grade          string   `json:"grade" enum:"junior,middle,senior"`
	Skills         []string `json:"skills" maxItems:"20"`
	WorkFormat     string   `json:"work_format" enum:"remote,hybrid,office"`
	City           string   `json:"city" maxLength:"100"`
	SalaryMin      int      `json:"salary_min" minimum:"1" maximum:"10000000"`
	SalaryMax      int      `json:"salary_max" minimum:"1" maximum:"10000000"`
}
type Vacancy struct {
	VacancyBody
	ID                   string       `json:"id"`
	CompanyName          string       `json:"company_name"`
	CompanyID            string       `json:"company_id"`
	CompanyAvatarVersion int64        `json:"company_avatar_version"`
	CompanyDescription   string       `json:"company_description"`
	CompanyWebsite       string       `json:"company_website"`
	Status               string       `json:"status"`
	CreatedAt            time.Time    `json:"created_at"`
	ApplicationCount     int          `json:"application_count"`
	MyApplication        *Application `json:"my_application"`
}
type Application struct {
	ID             string    `json:"id"`
	VacancyID      string    `json:"vacancy_id"`
	CandidateID    string    `json:"candidate_id"`
	CandidateName  string    `json:"candidate_name"`
	CandidateEmail string    `json:"candidate_email,omitempty"`
	Specialization string    `json:"specialization"`
	Grade          string    `json:"grade"`
	TestScore      int       `json:"test_score"`
	VacancyTitle   string    `json:"vacancy_title"`
	CompanyName    string    `json:"company_name"`
	SalaryMin      int       `json:"salary_min"`
	SalaryMax      int       `json:"salary_max"`
	CoverLetter    string    `json:"cover_letter"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func jsonRows[T any](ctx context.Context, a *App, query string, args ...any) ([]T, error) {
	rows, err := a.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var raw []byte
		var item T
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

const vacancyFields = `SELECT v.*,c.name AS company_name,c.user_id AS company_id,c.avatar_version AS company_avatar_version,c.description AS company_description,c.website AS company_website,
 (SELECT count(*) FROM applications x WHERE x.vacancy_id=v.id AND x.status<>'withdrawn') AS application_count,
 (SELECT row_to_json(x) FROM applications x WHERE x.vacancy_id=v.id AND x.candidate_id=$1) AS my_application
 FROM vacancies v JOIN companies c ON c.user_id=v.employer_id`

func (b *VacancyBody) clean() error {
	b.Title = strings.TrimSpace(b.Title)
	b.Description = strings.TrimSpace(b.Description)
	b.City = strings.TrimSpace(b.City)
	if len([]rune(b.Title)) < 4 || len([]rune(b.Description)) < 40 {
		return huma.Error400BadRequest("Добавьте название и подробное описание вакансии")
	}
	if b.SalaryMax < b.SalaryMin {
		return huma.Error400BadRequest("Верхняя граница зарплаты меньше нижней")
	}
	if b.WorkFormat != "remote" && b.City == "" {
		return huma.Error400BadRequest("Укажите город для работы в офисе или гибридного формата")
	}
	cleaned := []string{}
	seen := map[string]bool{}
	for _, s := range b.Skills {
		s = strings.TrimSpace(s)
		if len([]rune(s)) > 50 {
			return huma.Error400BadRequest("Название навыка слишком длинное")
		}
		key := strings.ToLower(s)
		if s != "" && !seen[key] {
			cleaned = append(cleaned, s)
			seen[key] = true
		}
	}
	b.Skills = cleaned
	return nil
}

func (a *App) vacancyRoutes(api huma.API) {
	type Feed struct {
		Specialization string `query:"specialization" enum:",backend,frontend,analytics"`
		Grade          string `query:"grade" enum:",junior,middle,senior"`
		WorkFormat     string `query:"work_format" enum:",remote,hybrid,office"`
		Salary         int    `query:"salary" minimum:"0" maximum:"10000000"`
		Page           int    `query:"page" default:"1" minimum:"1" maximum:"1000"`
	}
	type FeedResult struct {
		Vacancies []Vacancy `json:"vacancies"`
		Total     int       `json:"total"`
	}
	register(api, "GET", "/api/vacancies", "Вакансии с открытой зарплатой", func(ctx context.Context, in *Feed) (*Response[FeedResult], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		query := "SELECT row_to_json(t) FROM (" + vacancyFields + ` WHERE (($2='employer' AND v.employer_id=$1) OR ($2='candidate' AND v.status='published'))
   AND ($3='' OR v.specialization=$3) AND ($4='' OR v.grade=$4) AND ($5='' OR v.work_format=$5) AND v.salary_max >= $6 ORDER BY v.created_at DESC,v.id) t`
		rows, err := jsonRows[Vacancy](ctx, a, query, u.ID, u.Role, in.Specialization, in.Grade, in.WorkFormat, in.Salary)
		if err != nil {
			return nil, internal(err)
		}
		total := len(rows)
		start := min((in.Page-1)*20, total)
		return ok(FeedResult{rows[start:min(start+20, total)], total}), nil
	})
	type Detail struct {
		ID string `path:"id"`
	}
	register(api, "GET", "/api/vacancies/{id}", "Вакансия и собственный отклик", func(ctx context.Context, in *Detail) (*Response[Vacancy], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		v, err := readJSON[Vacancy](ctx, a.DB, "SELECT row_to_json(t) FROM ("+vacancyFields+` WHERE v.id=$2 AND ((v.employer_id=$1 AND $3='employer') OR ($3='candidate' AND (v.status='published' OR EXISTS(SELECT 1 FROM applications x WHERE x.vacancy_id=v.id AND x.candidate_id=$1))))) t`, u.ID, in.ID, u.Role)
		if isMissing(err) {
			return nil, huma.Error404NotFound("Вакансия не найдена")
		}
		if err != nil {
			return nil, internal(err)
		}
		return ok(v), nil
	})
	type Create struct{ Body VacancyBody }
	register(api, "POST", "/api/vacancies", "Создать черновик вакансии", func(ctx context.Context, in *Create) (*Response[struct {
		ID string `json:"id"`
	}], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		if err = in.Body.clean(); err != nil {
			return nil, err
		}
		var ready bool
		if err = a.DB.QueryRow(ctx, "SELECT length(trim(name))>=2 FROM companies WHERE user_id=$1", u.ID).Scan(&ready); err != nil {
			return nil, internal(err)
		}
		if !ready {
			return nil, huma.Error400BadRequest("Сначала заполните профиль компании")
		}
		if err = a.limit(ctx, "vacancies:"+u.ID, 10, 24*time.Hour); err != nil {
			return nil, err
		}
		b := in.Body
		id := uuid.NewString()
		_, err = a.DB.Exec(ctx, `INSERT INTO vacancies(id,employer_id,title,description,specialization,grade,skills,work_format,city,salary_min,salary_max) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, u.ID, b.Title, b.Description, b.Specialization, b.Grade, b.Skills, b.WorkFormat, b.City, b.SalaryMin, b.SalaryMax)
		if err != nil {
			return nil, internal(err)
		}
		return ok(struct {
			ID string `json:"id"`
		}{id}), nil
	})
	type Edit struct {
		ID   string `path:"id"`
		Body VacancyBody
	}
	register(api, "PUT", "/api/vacancies/{id}", "Редактировать собственную вакансию", func(ctx context.Context, in *Edit) (*Response[Message], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		if err = in.Body.clean(); err != nil {
			return nil, err
		}
		b := in.Body
		tag, err := a.DB.Exec(ctx, `UPDATE vacancies SET title=$3,description=$4,specialization=$5,grade=$6,skills=$7,work_format=$8,city=$9,salary_min=$10,salary_max=$11,updated_at=now() WHERE id=$1 AND employer_id=$2`, in.ID, u.ID, b.Title, b.Description, b.Specialization, b.Grade, b.Skills, b.WorkFormat, b.City, b.SalaryMin, b.SalaryMax)
		if err != nil {
			return nil, internal(err)
		}
		if tag.RowsAffected() == 0 {
			return nil, huma.Error404NotFound("Вакансия не найдена")
		}
		return msg("Вакансия сохранена"), nil
	})
	type Publish struct {
		ID   string `path:"id"`
		Body struct {
			Status string `json:"status" enum:"draft,published,closed"`
		}
	}
	register(api, "PATCH", "/api/vacancies/{id}", "Публикация или закрытие вакансии", func(ctx context.Context, in *Publish) (*Response[Message], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		tag, err := a.DB.Exec(ctx, `UPDATE vacancies SET status=$3,updated_at=now() WHERE id=$1 AND employer_id=$2 AND EXISTS(SELECT 1 FROM companies WHERE user_id=$2 AND length(trim(name))>=2)`, in.ID, u.ID, in.Body.Status)
		if err != nil {
			return nil, internal(err)
		}
		if tag.RowsAffected() == 0 {
			return nil, huma.Error404NotFound("Вакансия недоступна. Проверьте профиль компании.")
		}
		return msg("Статус вакансии обновлён"), nil
	})
	type Apply struct {
		ID   string `path:"id"`
		Body struct {
			CoverLetter string `json:"cover_letter" maxLength:"3000"`
			Consent     bool   `json:"consent"`
		}
	}
	register(api, "POST", "/api/vacancies/{id}/applications", "Откликнуться и открыть контакты этой компании", func(ctx context.Context, in *Apply) (*Response[Message], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		if !in.Body.Consent {
			return nil, huma.Error400BadRequest("Подтвердите передачу профиля и почты этой компании")
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		var grade string
		err = tx.QueryRow(ctx, "SELECT grade FROM profiles WHERE user_id=$1 FOR UPDATE", u.ID).Scan(&grade)
		if err != nil {
			return nil, internal(err)
		}
		if grade == "" {
			return nil, huma.Error409Conflict("Сначала подтвердите квалификацию тестированием")
		}
		var status string
		err = tx.QueryRow(ctx, "SELECT status FROM vacancies WHERE id=$1 FOR UPDATE", in.ID).Scan(&status)
		if isMissing(err) || status != "published" && err == nil {
			return nil, huma.Error404NotFound("Вакансия больше не принимает отклики")
		}
		if err != nil {
			return nil, internal(err)
		}
		if err = a.limit(ctx, "applications:"+u.ID, 20, 24*time.Hour); err != nil {
			return nil, err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO applications(id,vacancy_id,candidate_id,cover_letter) VALUES($1,$2,$3,$4) ON CONFLICT(vacancy_id,candidate_id) DO UPDATE SET status='new',cover_letter=excluded.cover_letter,consent_at=now(),created_at=now(),updated_at=now() WHERE applications.status='withdrawn'`, uuid.NewString(), in.ID, u.ID, strings.TrimSpace(in.Body.CoverLetter))
		if err != nil {
			return nil, internal(err)
		}
		if tag.RowsAffected() == 0 {
			return nil, huma.Error409Conflict("Вы уже откликнулись на эту вакансию")
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Отклик отправлен. Контакты доступны только этой компании."), nil
	})
	type AppFeed struct {
		VacancyID string `query:"vacancy_id"`
	}
	register(api, "GET", "/api/applications", "Отклики кандидата или на вакансии своей компании", func(ctx context.Context, in *AppFeed) (*Response[[]Application], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		out, err := jsonRows[Application](ctx, a, `SELECT row_to_json(t) FROM (SELECT x.*,u.name AS candidate_name,
   CASE WHEN $2='employer' AND x.status<>'withdrawn' THEN u.email ELSE '' END AS candidate_email,
   p.specialization,p.grade,p.test_score,v.title AS vacancy_title,c.name AS company_name,v.salary_min,v.salary_max
   FROM applications x JOIN vacancies v ON v.id=x.vacancy_id JOIN companies c ON c.user_id=v.employer_id JOIN users u ON u.id=x.candidate_id JOIN profiles p ON p.user_id=x.candidate_id
   WHERE (($2='employer' AND v.employer_id=$1) OR ($2='candidate' AND x.candidate_id=$1)) AND ($3='' OR x.vacancy_id=$3) ORDER BY x.created_at DESC LIMIT 300) t`, u.ID, u.Role, in.VacancyID)
		if err != nil {
			return nil, internal(err)
		}
		return ok(out), nil
	})
	type AppAction struct {
		ID   string `path:"id"`
		Body struct {
			Status string `json:"status" enum:"viewed,shortlisted,rejected,withdrawn"`
		}
	}
	register(api, "PATCH", "/api/applications/{id}", "Изменить статус отклика или отозвать его", func(ctx context.Context, in *AppAction) (*Response[Message], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		s := in.Body.Status
		if (u.Role == "candidate" && s != "withdrawn") || (u.Role == "employer" && s == "withdrawn") {
			return nil, huma.Error403Forbidden("Это действие недоступно для вашей роли")
		}
		tag, err := a.DB.Exec(ctx, `UPDATE applications x SET status=$3,updated_at=now() FROM vacancies v WHERE x.id=$1 AND v.id=x.vacancy_id AND
   (($4='candidate' AND x.candidate_id=$2 AND $3='withdrawn') OR ($4='employer' AND v.employer_id=$2 AND (x.status IN ('new','viewed') OR (x.status='shortlisted' AND $3='rejected'))))`, in.ID, u.ID, s, u.Role)
		if err != nil {
			return nil, internal(err)
		}
		if tag.RowsAffected() == 0 {
			return nil, huma.Error409Conflict("Отклик недоступен или статус уже изменён")
		}
		return msg("Статус отклика обновлён"), nil
	})
}
