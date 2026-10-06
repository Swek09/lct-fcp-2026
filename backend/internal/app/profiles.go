package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Swek09/lct-fcp-2026/backend/internal/pdfprofile"
	"github.com/Swek09/lct-fcp-2026/backend/internal/resume"
	"github.com/danielgtaylor/huma/v2"
	"strings"
	"time"
)

type Achievement struct {
	Title  string `json:"title"`
	Year   int    `json:"year"`
	Place  int    `json:"place"`
	Source string `json:"source"`
}
type Profile struct {
	UserID            string        `json:"user_id"`
	Name              string        `json:"name"`
	Headline          string        `json:"headline"`
	City              string        `json:"city"`
	About             string        `json:"about"`
	Skills            []string      `json:"skills"`
	ExperienceYears   int           `json:"experience_years"`
	Specialization    string        `json:"specialization"`
	Grade             string        `json:"grade"`
	TestScore         int           `json:"test_score"`
	Published         bool          `json:"published"`
	FspID             string        `json:"fsp_id"`
	Achievements      []Achievement `json:"achievements"`
	NextGradeChangeAt *time.Time    `json:"next_grade_change_at"`
	LastPracticeAt    *time.Time    `json:"last_practice_at"`
	Professional      resume.Data   `json:"professional"`
	AvatarVersion     int64         `json:"avatar_version"`
	Contacts          *Contacts     `json:"contacts,omitempty"`
}

const profileSQL = "SELECT row_to_json(t) FROM (SELECT p.*,u.name,(SELECT row_to_json(c) FROM (SELECT phone,telegram FROM account_contacts WHERE user_id=p.user_id) c) AS contacts FROM profiles p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1) t"

type Company struct {
	UserID        string         `json:"user_id"`
	Name          string         `json:"name" minLength:"2" maxLength:"150"`
	Description   string         `json:"description" maxLength:"4000"`
	Website       string         `json:"website" maxLength:"250"`
	AvatarVersion int64          `json:"avatar_version"`
	Details       CompanyDetails `json:"details"`
}

func (a *App) profileRoutes(api huma.API) {
	type PDFOutput struct {
		ContentType        string `header:"Content-Type"`
		ContentDisposition string `header:"Content-Disposition"`
		Body               []byte
	}
	register(api, "GET", "/api/profile/pdf", "Скачать собственный PDF-профиль", func(ctx context.Context, in *struct{}) (*PDFOutput, error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		p, err := readJSON[Profile](ctx, a.DB, profileSQL, u.ID)
		if err != nil {
			return nil, internal(err)
		}
		d := pdfprofile.Data{Name: p.Name, Email: u.Email, Headline: p.Headline, City: p.City, Specialization: p.Specialization, Grade: p.Grade, Score: p.TestScore, Skills: p.Skills, About: p.About, Experience: p.ExperienceYears, Professional: p.Professional}
		e, err := a.evidence(ctx, u.ID)
		if err != nil {
			return nil, internal(err)
		}
		d.QualificationSource, d.QualificationVersion = e.Source, e.Version
		for _, item := range p.Achievements {
			d.Achievements = append(d.Achievements, fmt.Sprintf("%s / %d / %d-е место", item.Title, item.Year, item.Place))
		}
		raw, err := pdfprofile.Generate(d)
		if err != nil {
			return nil, internal(err)
		}
		return &PDFOutput{ContentType: "application/pdf", ContentDisposition: "attachment; filename=fsp-profile.pdf", Body: raw}, nil
	})
	// Huma defaults []byte to JSON/base64. Document the actual download format.
	api.OpenAPI().Paths["/api/profile/pdf"].Get.Responses["200"].Content = map[string]*huma.MediaType{
		"application/pdf": {Schema: &huma.Schema{Type: "string", Format: "binary"}},
	}
	register(api, "GET", "/api/profile", "Мой профиль кандидата", func(ctx context.Context, in *struct{}) (*Response[Profile], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		p, err := readJSON[Profile](ctx, a.DB, profileSQL, u.ID)
		if err != nil {
			return nil, internal(err)
		}
		return ok(p), nil
	})
	type Update struct {
		Body struct {
			Name            string       `json:"name" minLength:"2" maxLength:"100"`
			Headline        string       `json:"headline" maxLength:"150"`
			City            string       `json:"city" maxLength:"100"`
			About           string       `json:"about" maxLength:"4000"`
			Skills          []string     `json:"skills" maxItems:"20"`
			ExperienceYears int          `json:"experience_years" minimum:"0" maximum:"60"`
			Published       bool         `json:"published"`
			Professional    *resume.Data `json:"professional,omitempty"`
			Contacts        *Contacts    `json:"contacts,omitempty"`
		}
	}
	register(api, "PUT", "/api/profile", "Обновить резюме и видимость", func(ctx context.Context, in *Update) (*Response[Message], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		b := in.Body
		if b.Contacts != nil {
			if validationErr := b.Contacts.clean(); validationErr != nil {
				return nil, huma.Error400BadRequest(validationErr.Error())
			}
		}
		var professional []byte
		if b.Professional != nil {
			normalized, validationErr := resume.Normalize(*b.Professional)
			if validationErr != nil {
				return nil, huma.Error400BadRequest(validationErr.Error())
			}
			professional, err = json.Marshal(normalized)
			if err != nil {
				return nil, internal(err)
			}
		}
		skills := []string{}
		seen := map[string]bool{}
		for _, s := range b.Skills {
			s = strings.TrimSpace(s)
			if len(s) > 50 {
				return nil, huma.Error400BadRequest("Название навыка слишком длинное")
			}
			if s != "" && !seen[strings.ToLower(s)] {
				skills = append(skills, s)
				seen[strings.ToLower(s)] = true
			}
		}
		if len(strings.TrimSpace(b.Name)) < 2 {
			return nil, huma.Error400BadRequest("Укажите имя")
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		_, err = tx.Exec(ctx, "UPDATE users SET name=$2 WHERE id=$1", u.ID, strings.TrimSpace(b.Name))
		if err != nil {
			return nil, internal(err)
		}
		_, err = tx.Exec(ctx, "UPDATE profiles SET headline=$2,city=$3,about=$4,skills=$5,experience_years=$6,published=$7,professional=COALESCE($8::jsonb,professional),updated_at=now() WHERE user_id=$1", u.ID, b.Headline, b.City, b.About, skills, b.ExperienceYears, b.Published, professional)
		if err != nil {
			return nil, internal(err)
		}
		if b.Contacts != nil {
			_, err = tx.Exec(ctx, "INSERT INTO account_contacts(user_id,phone,telegram) VALUES($1,$2,$3) ON CONFLICT(user_id) DO UPDATE SET phone=EXCLUDED.phone,telegram=EXCLUDED.telegram", u.ID, b.Contacts.Phone, b.Contacts.Telegram)
			if err != nil {
				return nil, internal(err)
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Профиль сохранён"), nil
	})
	register(api, "GET", "/api/company", "Профиль компании", func(ctx context.Context, in *struct{}) (*Response[Company], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		v, err := readJSON[Company](ctx, a.DB, "SELECT row_to_json(t) FROM (SELECT * FROM companies WHERE user_id=$1) t", u.ID)
		if err != nil {
			return nil, internal(err)
		}
		return ok(v), nil
	})
	type CompanyInput struct {
		Body struct {
			Name        string          `json:"name" minLength:"2" maxLength:"150"`
			Description string          `json:"description" maxLength:"4000"`
			Website     string          `json:"website" maxLength:"250"`
			Details     *CompanyDetails `json:"details,omitempty"`
		}
	}
	register(api, "PUT", "/api/company", "Изменить профиль компании", func(ctx context.Context, in *CompanyInput) (*Response[Message], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		b := in.Body
		if len([]rune(strings.TrimSpace(b.Name))) < 2 {
			return nil, huma.Error400BadRequest("Укажите название компании")
		}
		if b.Website, err = cleanWebURL(b.Website); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		var details []byte
		if b.Details != nil {
			if err = b.Details.clean(); err != nil {
				return nil, huma.Error400BadRequest(err.Error())
			}
			details, err = json.Marshal(b.Details)
			if err != nil {
				return nil, internal(err)
			}
		}
		_, err = a.DB.Exec(ctx, "UPDATE companies SET name=$2,description=$3,website=$4,details=COALESCE($5::jsonb,details) WHERE user_id=$1", u.ID, strings.TrimSpace(b.Name), b.Description, b.Website, details)
		if err != nil {
			return nil, internal(err)
		}
		return msg("Компания сохранена"), nil
	})
	type PublicCompany struct {
		ID string `path:"id"`
	}
	register(api, "GET", "/api/companies/{id}", "Информация о компании для кандидатов", func(ctx context.Context, in *PublicCompany) (*Response[Company], error) {
		if _, err := a.require(ctx, ""); err != nil {
			return nil, err
		}
		v, err := readJSON[Company](ctx, a.DB, "SELECT row_to_json(t) FROM (SELECT c.* FROM companies c JOIN users u ON u.id=c.user_id WHERE c.user_id=$1 AND u.verified AND length(trim(c.name))>=2) t", in.ID)
		if isMissing(err) {
			return nil, huma.Error404NotFound("Компания не найдена")
		}
		if err != nil {
			return nil, internal(err)
		}
		return ok(v), nil
	})
	type FSPInput struct {
		Body struct {
			ID string `json:"id" maxLength:"50"`
		}
	}
	register(api, "POST", "/api/profile/fsp", "Демонстрационная связь с ФСП ID", func(ctx context.Context, in *FSPInput) (*Response[Message], error) {
		u, err := a.require(ctx, "candidate")
		if err != nil {
			return nil, err
		}
		if !a.Config.Demo {
			return nil, huma.Error503ServiceUnavailable("Реестр ФСП ещё не подключён")
		}
		id := in.Body.ID
		items := []Achievement{}
		switch id {
		case "":
		case "DEMO-1001":
			items = []Achievement{{"Кубок по спортивному программированию", 2025, 3, "demo"}}
		case "DEMO-1002":
			items = []Achievement{{"Региональный чемпионат", 2025, 1, "demo"}, {"Командный хакатон", 2026, 2, "demo"}}
		default:
			return nil, huma.Error404NotFound("Демонстрационный ID не найден. Доступны DEMO-1001 и DEMO-1002.")
		}
		raw, _ := json.Marshal(items)
		_, err = a.DB.Exec(ctx, "UPDATE profiles SET fsp_id=$2,achievements=$3,updated_at=now() WHERE user_id=$1", u.ID, id, raw)
		if err != nil {
			return nil, internal(err)
		}
		return msg("Демонстрационные достижения обновлены"), nil
	})
}
