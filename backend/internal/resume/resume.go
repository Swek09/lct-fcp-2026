// Package resume holds self-reported experience, distinct from tested qualification.
package resume

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

type Experience struct {
	Company     string `json:"company" maxLength:"150"`
	Role        string `json:"role" maxLength:"150"`
	Start       string `json:"start" maxLength:"7"`
	End         string `json:"end" maxLength:"7"`
	Current     bool   `json:"current"`
	Description string `json:"description" maxLength:"3000"`
	Results     string `json:"results" maxLength:"2000"`
}

type Project struct {
	Name        string   `json:"name" maxLength:"150"`
	Role        string   `json:"role" maxLength:"150"`
	Description string   `json:"description" maxLength:"3000"`
	Results     string   `json:"results" maxLength:"2000"`
	Stack       []string `json:"stack" maxItems:"20"`
	Repository  string   `json:"repository" maxLength:"500"`
	Demo        string   `json:"demo" maxLength:"500"`
}

type Data struct {
	Timezone     string        `json:"timezone" maxLength:"80" required:"false"`
	Domains      []string      `json:"domains" maxItems:"12" required:"false"`
	Interests    string        `json:"interests" maxLength:"2000" required:"false"`
	Education    []Education   `json:"education" maxItems:"8" required:"false"`
	Certificates []Certificate `json:"certificates" maxItems:"12" required:"false"`
	GitHub       string        `json:"github" maxLength:"500"`
	Portfolio    string        `json:"portfolio" maxLength:"500"`
	SalaryMin    int           `json:"salary_min" minimum:"0" maximum:"10000000"`
	SalaryMax    int           `json:"salary_max" minimum:"0" maximum:"10000000"`
	WorkFormats  []string      `json:"work_formats" maxItems:"3"`
	Employment   string        `json:"employment" enum:",full_time,part_time,contract,internship"`
	Availability string        `json:"availability" enum:",immediately,two_weeks,month,negotiable"`
	SearchStatus string        `json:"search_status" enum:",active,open,paused"`
	Relocation   bool          `json:"relocation"`
	Languages    []string      `json:"languages" maxItems:"8"`
	Experience   []Experience  `json:"experience" maxItems:"8"`
	Projects     []Project     `json:"projects" maxItems:"8"`
}

type Education struct {
	Institution string `json:"institution" maxLength:"200"`
	Degree      string `json:"degree" maxLength:"100"`
	Field       string `json:"field" maxLength:"150"`
	StartYear   int    `json:"start_year" minimum:"0" maximum:"2100"`
	EndYear     int    `json:"end_year" minimum:"0" maximum:"2100"`
	Current     bool   `json:"current"`
}

type Certificate struct {
	Title  string `json:"title" maxLength:"200"`
	Issuer string `json:"issuer" maxLength:"150"`
	Year   int    `json:"year" minimum:"0" maximum:"2100"`
	URL    string `json:"url" maxLength:"500"`
}

func webURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || utf8.RuneCountInString(value) > 500 {
		return "", fmt.Errorf("Укажите полную ссылку с https:// или http:// без логина и пароля")
	}
	return u.String(), nil
}

func cleanList(values []string, limit int) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if utf8.RuneCountInString(value) > 80 {
			return nil, fmt.Errorf("Название в списке не должно превышать 80 символов")
		}
		if value != "" && !seen[strings.ToLower(value)] {
			out = append(out, value)
			seen[strings.ToLower(value)] = true
		}
	}
	if len(out) > limit {
		return nil, fmt.Errorf("В списке допускается не более %d элементов", limit)
	}
	return out, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func month(value string) bool { _, err := time.Parse("2006-01", value); return err == nil }

// Normalize validates semantic relationships in addition to API length constraints.
func Normalize(d Data) (Data, error) {
	var err error
	d.Timezone = strings.TrimSpace(d.Timezone)
	if d.Timezone != "" {
		if _, err = time.LoadLocation(d.Timezone); err != nil {
			return d, fmt.Errorf("Укажите часовой пояс, например Europe/Moscow")
		}
	}
	if d.Domains, err = cleanList(d.Domains, 12); err != nil {
		return d, err
	}
	d.Interests = strings.TrimSpace(d.Interests)
	if len(d.Education) > 8 || len(d.Certificates) > 12 {
		return d, fmt.Errorf("Можно добавить до 8 записей об образовании и до 12 сертификатов")
	}
	for i := range d.Education {
		e := &d.Education[i]
		e.Institution = strings.TrimSpace(e.Institution)
		e.Degree = strings.TrimSpace(e.Degree)
		e.Field = strings.TrimSpace(e.Field)
		if e.Institution == "" {
			return d, fmt.Errorf("Укажите учебное заведение")
		}
		if (e.StartYear != 0 && (e.StartYear < 1950 || e.StartYear > time.Now().Year())) || (e.EndYear != 0 && (e.EndYear < 1950 || e.EndYear > time.Now().Year()+10 || (e.StartYear != 0 && e.EndYear < e.StartYear))) || (e.Current && e.EndYear != 0) {
			return d, fmt.Errorf("Проверьте годы обучения")
		}
	}
	for i := range d.Certificates {
		c := &d.Certificates[i]
		c.Title = strings.TrimSpace(c.Title)
		c.Issuer = strings.TrimSpace(c.Issuer)
		if c.Title == "" || (c.Year != 0 && (c.Year < 1950 || c.Year > time.Now().Year())) {
			return d, fmt.Errorf("Укажите название сертификата и проверьте год получения")
		}
		if c.URL, err = webURL(c.URL); err != nil {
			return d, err
		}
	}
	if d.GitHub, err = webURL(d.GitHub); err != nil {
		return d, err
	}
	if d.Portfolio, err = webURL(d.Portfolio); err != nil {
		return d, err
	}
	if d.SalaryMin < 0 || d.SalaryMax < 0 || d.SalaryMin > 10000000 || d.SalaryMax > 10000000 || (d.SalaryMax > 0 && d.SalaryMax < d.SalaryMin) {
		return d, fmt.Errorf("Проверьте диапазон зарплатных ожиданий")
	}
	if !oneOf(d.Employment, "", "full_time", "part_time", "contract", "internship") || !oneOf(d.Availability, "", "immediately", "two_weeks", "month", "negotiable") || !oneOf(d.SearchStatus, "", "active", "open", "paused") {
		return d, fmt.Errorf("Выберите значение предпочтений из списка")
	}
	if d.WorkFormats, err = cleanList(d.WorkFormats, 3); err != nil {
		return d, err
	}
	for _, format := range d.WorkFormats {
		if !oneOf(format, "remote", "hybrid", "office") {
			return d, fmt.Errorf("Неизвестный формат работы")
		}
	}
	if d.Languages, err = cleanList(d.Languages, 8); err != nil {
		return d, err
	}
	if len(d.Experience) > 8 || len(d.Projects) > 8 {
		return d, fmt.Errorf("Можно добавить до 8 мест работы и до 8 проектов")
	}
	if d.Experience == nil {
		d.Experience = []Experience{}
	}
	if d.Projects == nil {
		d.Projects = []Project{}
	}
	for i := range d.Experience {
		e := &d.Experience[i]
		e.Company, e.Role = strings.TrimSpace(e.Company), strings.TrimSpace(e.Role)
		e.Description, e.Results = strings.TrimSpace(e.Description), strings.TrimSpace(e.Results)
		if e.Company == "" || e.Role == "" {
			return d, fmt.Errorf("Укажите компанию и роль для каждого места работы")
		}
		if !month(e.Start) || (!e.Current && (!month(e.End) || e.End < e.Start)) || (e.Current && e.End != "") {
			return d, fmt.Errorf("Проверьте период работы: месяц начала, окончания либо отметку «Работаю сейчас»")
		}
	}
	for i := range d.Projects {
		p := &d.Projects[i]
		p.Name, p.Role = strings.TrimSpace(p.Name), strings.TrimSpace(p.Role)
		p.Description, p.Results = strings.TrimSpace(p.Description), strings.TrimSpace(p.Results)
		if p.Name == "" || p.Role == "" || p.Description == "" {
			return d, fmt.Errorf("Укажите название проекта, вашу роль и описание")
		}
		if p.Repository, err = webURL(p.Repository); err != nil {
			return d, err
		}
		if p.Demo, err = webURL(p.Demo); err != nil {
			return d, err
		}
		if p.Stack, err = cleanList(p.Stack, 20); err != nil {
			return d, err
		}
	}
	return d, nil
}
