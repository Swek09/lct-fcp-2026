package pdfprofile

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/go-pdf/fpdf"
	"strings"
)

//go:embed fonts/JetBrainsMono-Regular.ttf
var font []byte

type Data struct {
	Name, Email, Headline, City, Specialization, Grade, About string
	Skills                                                    []string
	Score, Experience                                         int
	Achievements                                              []string
}

func Generate(d Data) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)
	p.AddUTF8FontFromBytes("profile", "", font)
	p.AddPage()
	p.SetFont("profile", "", 10)
	p.SetTextColor(80, 65, 210)
	p.Cell(0, 8, "ФСП / КАРЬЕРА")
	p.Ln(16)
	p.SetTextColor(24, 32, 42)
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
		p.SetTextColor(80, 65, 210)
		p.SetFont("profile", "", 11)
		p.MultiCell(170, 7, title, "", "L", false)
		p.Ln(2)
		p.SetTextColor(24, 32, 42)
		p.SetFont("profile", "", 10)
		p.MultiCell(170, 6, body, "", "L", false)
		p.Ln(8)
	}
	level := "Уровень пока не подтверждён"
	if d.Grade != "" {
		level = fmt.Sprintf("%s / %s\nТест: %d из 100", d.Specialization, d.Grade, d.Score)
	}
	block("ПОДТВЕРЖДЁННАЯ КАТЕГОРИЯ", level)
	block("НАВЫКИ И ОПЫТ", strings.Join(d.Skills, ", ")+fmt.Sprintf("\nОпыт: %d лет. Сведения указаны кандидатом.", d.Experience))
	if d.About != "" {
		block("О СЕБЕ", d.About)
	}
	if len(d.Achievements) > 0 {
		block("ДОСТИЖЕНИЯ ФСП (ДЕМО)", strings.Join(d.Achievements, "\n"))
	} else {
		block("ДОСТИЖЕНИЯ ФСП", "История ФСП не привязана.")
	}
	p.SetTextColor(100, 105, 115)
	p.SetFont("profile", "", 8)
	p.MultiCell(170, 5, "Профиль сформирован кандидатом. Проверяйте актуальный статус на платформе. Демонстрационные достижения не являются подтверждением реального участия.", "", "L", false)
	var b bytes.Buffer
	err := p.Output(&b)
	return b.Bytes(), err
}
