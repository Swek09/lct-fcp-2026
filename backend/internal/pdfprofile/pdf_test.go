package pdfprofile

import (
	"bytes"
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
