package pdfprofile

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/Swek09/lct-fcp-2026/backend/internal/resume"
	"github.com/go-pdf/fpdf"
	"strings"
)

//go:embed fonts/JetBrainsMono-Regular.ttf
var font []byte

type Data struct {
	Name, Email, Headline, City, Specialization, Grade, About string
	QualificationSource, QualificationVersion                 string
	Skills                                                    []string
	Score, Experience                                         int
	Achievements                                              []string
	Professional                                              resume.Data
}

func Generate(d Data) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)
	p.AddUTF8FontFromBytes("profile", "", font)
	p.AddPage()
	p.SetFont("profile", "", 10)
	p.SetTextColor(64, 47, 255)
	p.Cell(0, 8, "ФСП / КАРЬЕРА")
	p.Ln(16)
	p.SetTextColor(27, 28, 33)
	p.SetFont("profile", "", 22)
	p.MultiCell(170, 11, d.Name, "", "L", false)
	p.Ln(4)
	p.SetFont("profile", "", 11)
	p.MultiCell(170, 7, d.Headline, "", "L", false)
	p.Ln(4)
	p.SetFont("profile", "", 9)
	p.MultiCell(170, 6, d.Email+" | "+d.City, "", "L", false)
	p.Ln(8)
	block := func(title, body string) {
		// Keep a section heading with at least two lines of its content.
		p.SetFont("profile", "", 11)
		if p.GetY()+float64(len(p.SplitText(title, 170)))*7+14 > 277 {
			p.AddPage()
		}
		p.SetTextColor(64, 47, 255)
		p.SetFont("profile", "", 11)
		p.MultiCell(170, 7, title, "", "L", false)
		p.Ln(2)
		p.SetTextColor(27, 28, 33)
		p.SetFont("profile", "", 10)
		p.MultiCell(170, 6, body, "", "L", false)
		p.Ln(8)
	}
	level := "Уровень пока не подтверждён"
	title := "КАТЕГОРИЯ"
	if d.Grade != "" {
		spec := map[string]string{"backend": "Backend-разработка", "frontend": "Frontend-разработка", "analytics": "Аналитика данных"}[d.Specialization]
		if spec == "" {
			spec = d.Specialization
		}
		grade := map[string]string{"junior": "Junior", "middle": "Middle", "senior": "Senior"}[d.Grade]
		if grade == "" {
			grade = d.Grade
		}
		level = fmt.Sprintf("%s / %s\nРезультат: %d из 100", spec, grade, d.Score)
		switch d.QualificationSource {
		case "demo":
			level += "\nДемонстрационные категория и баллы. Реальная попытка отсутствует."
		case "imported":
			level += "\nИмпортированные сведения; подтверждающая попытка на платформе отсутствует."
		case "assessment":
			title = "ПОДТВЕРЖДЁННАЯ КАТЕГОРИЯ"
			level += "\nВерсия теста: " + d.QualificationVersion
		}
	}
	block(title, level)
	block("НАВЫКИ И ОПЫТ", strings.Join(d.Skills, ", ")+fmt.Sprintf("\nОпыт: %d лет. Сведения указаны кандидатом.", d.Experience))
	if d.About != "" {
		block("О СЕБЕ", d.About)
	}
	r := d.Professional
	preferences := []string{}
	if value := map[string]string{"active": "Активно ищу работу", "open": "Рассматриваю предложения", "paused": "Пока не ищу работу"}[r.SearchStatus]; value != "" {
		preferences = append(preferences, value)
	}
	if r.SalaryMin > 0 || r.SalaryMax > 0 {
		salary := fmt.Sprintf("от %d ₽", r.SalaryMin)
		if r.SalaryMin == 0 {
			salary = fmt.Sprintf("до %d ₽", r.SalaryMax)
		} else if r.SalaryMax > 0 {
			salary += fmt.Sprintf(" до %d ₽", r.SalaryMax)
		}
		preferences = append(preferences, "Ожидания: "+salary+" в месяц, до налогов")
	}
	formats := []string{}
	for _, value := range r.WorkFormats {
		formats = append(formats, map[string]string{"remote": "Удалённо", "hybrid": "Гибрид", "office": "В офисе"}[value])
	}
	if len(formats) > 0 {
		preferences = append(preferences, "Формат: "+strings.Join(formats, ", "))
	}
	if value := map[string]string{"full_time": "Полная занятость", "part_time": "Частичная занятость", "contract": "Проектная работа", "internship": "Стажировка"}[r.Employment]; value != "" {
		preferences = append(preferences, value)
	}
	if value := map[string]string{"immediately": "Готов выйти сейчас", "two_weeks": "Через две недели", "month": "Через месяц", "negotiable": "По договорённости"}[r.Availability]; value != "" {
		preferences = append(preferences, value)
	}
	if r.Relocation {
		preferences = append(preferences, "Готов к переезду")
	}
	if len(r.Languages) > 0 {
		preferences = append(preferences, "Языки: "+strings.Join(r.Languages, ", "))
	}
	if r.Timezone != "" {
		preferences = append(preferences, "Часовой пояс: "+r.Timezone)
	}
	if len(r.Domains) > 0 {
		preferences = append(preferences, "Отраслевой опыт: "+strings.Join(r.Domains, ", "))
	}
	if len(preferences) > 0 {
		block("ПОЖЕЛАНИЯ К РАБОТЕ", strings.Join(preferences, "\n"))
	}
	links := []string{}
	if r.GitHub != "" {
		links = append(links, "GitHub / GitLab: "+r.GitHub)
	}
	if r.Portfolio != "" {
		links = append(links, "Портфолио: "+r.Portfolio)
	}
	if len(links) > 0 {
		block("ССЫЛКИ НА РАБОТЫ", strings.Join(links, "\n"))
	}
	if r.Interests != "" {
		block("ИНТЕРЕСУЮЩИЕ ЗАДАЧИ", r.Interests)
	}
	for _, item := range r.Experience {
		end := item.End
		if item.Current {
			end = "по настоящее время"
		}
		body := item.Company + " | " + item.Start + " — " + end + "\n" + item.Description
		if item.Results != "" {
			body += "\nРезультаты: " + item.Results
		}
		block("ОПЫТ / "+item.Role, body)
	}
	for _, item := range r.Projects {
		body := "Роль: " + item.Role + "\n" + item.Description
		if item.Results != "" {
			body += "\nВклад и результат: " + item.Results
		}
		if len(item.Stack) > 0 {
			body += "\nСтек: " + strings.Join(item.Stack, ", ")
		}
		if item.Repository != "" {
			body += "\nРепозиторий: " + item.Repository
		}
		if item.Demo != "" {
			body += "\nДемонстрация: " + item.Demo
		}
		block("ПРОЕКТ / "+item.Name, body)
	}
	for _, e := range r.Education {
		period := ""
		if e.StartYear > 0 {
			period = fmt.Sprintf("%d", e.StartYear)
		}
		if e.Current {
			period += " — по настоящее время"
		} else if e.EndYear > 0 {
			period += fmt.Sprintf(" — %d", e.EndYear)
		}
		block("ОБРАЗОВАНИЕ / "+e.Institution, strings.Join([]string{e.Degree, e.Field, period}, "\n"))
	}
	for _, c := range r.Certificates {
		body := c.Issuer
		if c.Year > 0 {
			body += fmt.Sprintf(" | %d", c.Year)
		}
		if c.URL != "" {
			body += "\n" + c.URL
		}
		block("СЕРТИФИКАТ / "+c.Title, body)
	}
	if len(d.Achievements) > 0 {
		block("ДОСТИЖЕНИЯ ФСП (ДЕМО)", strings.Join(d.Achievements, "\n"))
	} else {
		block("ДОСТИЖЕНИЯ ФСП", "История ФСП не привязана.")
	}
	p.SetTextColor(27, 28, 33)
	p.SetFont("profile", "", 8)
	p.MultiCell(170, 5, "Профиль сформирован кандидатом. Проверяйте актуальный статус на платформе. Демонстрационные достижения не являются подтверждением реального участия.", "", "L", false)
	var b bytes.Buffer
	err := p.Output(&b)
	return b.Bytes(), err
}
