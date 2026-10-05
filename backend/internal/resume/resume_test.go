package resume

import (
	"strings"
	"testing"
)

func TestNormalizeProfessionalProfile(t *testing.T) {
	d, err := Normalize(Data{GitHub: " https://github.com/example ", SalaryMin: 180000, SalaryMax: 240000, WorkFormats: []string{"remote", "remote"}, Languages: []string{" Русский ", "русский", "English B2"}, Experience: []Experience{{Company: " Компания ", Role: " Go-разработчик ", Start: "2024-01", Current: true}}, Projects: []Project{{Name: " API ", Role: "Автор", Description: " Сервис ", Stack: []string{" Go ", "go", "Redis"}, Repository: "https://example.org/repo"}}})
	if err != nil {
		t.Fatal(err)
	}
	if d.GitHub != "https://github.com/example" || len(d.Languages) != 2 || len(d.WorkFormats) != 1 || len(d.Projects[0].Stack) != 2 || d.Experience[0].Company != "Компания" {
		t.Fatalf("incorrect normalization: %+v", d)
	}
	empty, err := Normalize(Data{})
	if err != nil || empty.Experience == nil || empty.Projects == nil || empty.Languages == nil || empty.WorkFormats == nil {
		t.Fatal("empty profile must be usable")
	}
}

func TestRejectInvalidProfessionalData(t *testing.T) {
	for _, link := range []string{"javascript:alert(1)", "data:text/html,test", "mailto:user@example.org", "//example.org", "https://user:password@example.org/", strings.Repeat("x", 501), "https://"} {
		for _, d := range []Data{{GitHub: link}, {Portfolio: link}, {Projects: []Project{{Name: "Test", Role: "Author", Description: "Description", Repository: link}}}} {
			if _, err := Normalize(d); err == nil {
				t.Fatalf("accepted unsafe/invalid link %q", link)
			}
		}
	}
	for _, d := range []Data{{SalaryMin: -1}, {SalaryMin: 200000, SalaryMax: 100000}, {SalaryMax: 10000001}, {Employment: "unknown"}, {Availability: "tomorrow"}, {SearchStatus: "invisible"}, {WorkFormats: []string{"unknown"}}, {Languages: []string{strings.Repeat("я", 81)}}, {Experience: make([]Experience, 9)}, {Projects: make([]Project, 9)}, {Projects: []Project{{Name: "Test", Role: "Author"}}}} {
		if _, err := Normalize(d); err == nil {
			t.Fatalf("accepted invalid data: %+v", d)
		}
	}
	for _, e := range []Experience{{Company: "X", Role: "Dev", Start: "2025-13", Current: true}, {Company: "X", Role: "Dev", Start: "2025-01", End: "2024-12"}, {Company: "X", Role: "Dev", Start: "2025-01", End: "2025-02", Current: true}, {Company: " ", Role: "Dev", Start: "2025-01", Current: true}} {
		if _, err := Normalize(Data{Experience: []Experience{e}}); err == nil {
			t.Fatalf("accepted invalid work history: %+v", e)
		}
	}
}
