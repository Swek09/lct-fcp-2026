package pdfprofile

import (
	"bytes"
	"github.com/Swek09/lct-fcp-2026/backend/internal/resume"
	"regexp"
	"strings"
	"testing"
)

func TestCyrillicProfilePDF(t *testing.T) {
	b, err := Generate(Data{Name: "Александр Морозов", Headline: "Go-разработчик", About: "Проверка кириллицы и длинного описания.", Skills: []string{"Go", "PostgreSQL"}, Grade: "Middle", Specialization: "Backend", Score: 90})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) || len(b) < 5000 {
		t.Fatal("invalid PDF output")
	}
}

func TestProfessionalProfileSpansPages(t *testing.T) {
	d := Data{Name: "Тестовый кандидат", Headline: "Backend-разработчик", Professional: resume.Data{SalaryMin: 180000, SalaryMax: 240000, WorkFormats: []string{"remote"}, Employment: "full_time", Languages: []string{"Русский", "English B2"}, Experience: []resume.Experience{}, Projects: []resume.Project{}}}
	for i := 0; i < 8; i++ {
		d.Professional.Experience = append(d.Professional.Experience, resume.Experience{Company: "Демонстрационная компания", Role: "Разработчик", Start: "2020-01", End: "2021-12", Description: strings.Repeat("Разработка API и интеграций. ", 20), Results: "Поддержка транзакций и интеграционные тесты."})
		d.Professional.Projects = append(d.Professional.Projects, resume.Project{Name: "Проект", Role: "Автор", Description: strings.Repeat("Подробное описание проекта. ", 20), Results: "Проверка экспорта длинного резюме", Stack: []string{"Go", "Redis"}, Repository: "https://example.org/" + strings.Repeat("long-path-", 30)})
	}
	b, err := Generate(d)
	if err != nil {
		t.Fatal(err)
	}
	if count := len(regexp.MustCompile(`/Type /Page\b`).FindAll(b, -1)); count < 3 {
		t.Fatalf("long profile not paginated: %d pages", count)
	}
}
