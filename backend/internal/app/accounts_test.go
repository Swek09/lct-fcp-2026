package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/Swek09/lct-fcp-2026/backend/internal/resume"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func avatarFixture(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.NRGBA{R: 64, G: 47, B: 255, A: 255})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func TestAvatarRasterValidation(t *testing.T) {
	raw, err := normalizeAvatar(avatarFixture(t, 31, 19))
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || format != "jpeg" || cfg.Width != 256 || cfg.Height != 256 {
		t.Fatalf("unsafe output: %s %+v %v", format, cfg, err)
	}
	img, _, _ := image.Decode(bytes.NewReader(raw))
	r, g, b, _ := img.At(255, 255).RGBA()
	if r/257 < 20 || r/257 > 35 || g/257 < 20 || g/257 > 35 || b/257 < 25 || b/257 > 40 {
		t.Fatal("transparent pixel not composited on brand background")
	}
	for _, bad := range [][]byte{nil, []byte(`<svg onload="alert(1)"></svg>`), []byte("not an image"), bytes.Repeat([]byte{1}, 512*1024+1), avatarFixture(t, 2049, 1)} {
		if _, err := normalizeAvatar(bad); err == nil {
			t.Fatal("unsafe input accepted")
		}
	}
}

func TestCompanySemanticValidation(t *testing.T) {
	d := CompanyDetails{ContactEmail: " hr@example.test ", Telegram: "@fsp_team", Phone: "+7 (900) 000-00-00", Technologies: []string{"Go", "go", " Redis "}, WorkFormats: []string{"remote"}, FoundedYear: 2018}
	if err := d.clean(); err != nil {
		t.Fatal(err)
	}
	if len(d.Technologies) != 2 || d.Telegram != "fsp_team" || d.ContactEmail != "hr@example.test" {
		t.Fatal("normalization failed")
	}
	for _, d := range []CompanyDetails{{CareersURL: "javascript:alert(1)"}, {CareersURL: "https://user:pass@example.org"}, {INN: "123"}, {ContactEmail: "a\r\nb@example.test"}, {Telegram: "https://t.me/name"}, {FoundedYear: 2200}, {WorkFormats: []string{"unknown"}}, {Phone: "-------"}} {
		if err := d.clean(); err == nil {
			t.Fatalf("invalid company data accepted: %+v", d)
		}
	}
}

func checkAccounts(t *testing.T, a *App, h http.Handler) {
	t.Helper()
	ctx := context.Background()
	id := uuid.NewString()
	email := "account-" + id + "@example.test"
	_, err := a.DB.Exec(ctx, `INSERT INTO users(id,email,password_hash,role,name,verified) SELECT $1,$2,password_hash,'candidate','Профиль для проверки',true FROM users WHERE email='candidate@demo.fsp'`, id, email)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.DB.Exec(ctx, `INSERT INTO profiles(user_id,specialization,grade,test_score,published) VALUES($1,'backend','middle',88,true)`, id); err != nil {
		t.Fatal(err)
	}
	defer func() {
		a.DB.Exec(ctx, "DELETE FROM invitations WHERE candidate_id=$1", id)
		a.DB.Exec(ctx, "DELETE FROM users WHERE id=$1", id)
	}()
	owner := &testClient{handler: h}
	owner.login(t, email, "DemoPass2026!")
	employer := &testClient{handler: h}
	employer.login(t, "employer@demo.fsp", "DemoPass2026!")
	anon := &testClient{handler: h}
	defer func() {
		a.Redis.Del(ctx, "session:"+digest(owner.cookie.Value), "session:"+digest(employer.cookie.Value))
	}()
	expect(t, anon, "POST", "/api/avatar", map[string]string{"image": "bad"}, 401)
	expect(t, owner, "POST", "/api/avatar", map[string]string{"image": base64.StdEncoding.EncodeToString([]byte("<svg/>"))}, 400)
	raw := expect(t, owner, "POST", "/api/avatar", map[string]string{"image": base64.StdEncoding.EncodeToString(avatarFixture(t, 32, 32))}, 200)
	var info AvatarInfo
	json.Unmarshal(raw, &info)
	if info.Version == 0 {
		t.Fatal("missing avatar version")
	}
	expect(t, owner, "GET", "/api/avatars/"+id, nil, 200)
	expect(t, employer, "GET", "/api/avatars/"+id, nil, 200)
	expect(t, anon, "GET", "/api/avatars/"+id, nil, 401)
	body := map[string]any{"name": "Профиль для проверки", "published": true, "skills": []string{"Go"}, "professional": map[string]any{"timezone": "Europe/Moscow", "domains": []string{"Финтех"}, "education": []map[string]any{{"institution": "Университет · демо", "start_year": 2018, "end_year": 2022}}, "certificates": []map[string]any{{"title": "Курс · демо", "year": 2024, "url": "https://example.org/certificate"}}}, "contacts": map[string]string{"phone": "+7 900 123-45-67", "telegram": "fsp_candidate"}}
	body["headline"], body["city"], body["about"], body["experience_years"] = "Backend-разработчик", "Москва", "Описание", 2
	body["professional"] = resume.Data{Timezone: "Europe/Moscow", Domains: []string{"Финтех"}, Education: []resume.Education{{Institution: "Университет · демо", StartYear: 2018, EndYear: 2022}}, Certificates: []resume.Certificate{{Title: "Курс · демо", Year: 2024, URL: "https://example.org/certificate"}}}
	expect(t, owner, "PUT", "/api/profile", body, 200)
	var p Profile
	json.Unmarshal(expect(t, owner, "GET", "/api/profile", nil, 200), &p)
	if p.Contacts == nil || p.Contacts.Telegram != "fsp_candidate" || len(p.Professional.Education) != 1 || p.AvatarVersion == 0 {
		t.Fatal("account details not persisted")
	}
	for _, path := range []string{"/api/candidates/" + id, "/api/candidates"} {
		raw := expect(t, employer, "GET", path, nil, 200)
		if bytes.Contains(raw, []byte("fsp_candidate")) || bytes.Contains(raw, []byte("123-45-67")) {
			t.Fatal("private contacts leaked")
		}
	}
	delete(body, "professional")
	delete(body, "contacts")
	expect(t, owner, "PUT", "/api/profile", body, 200)
	json.Unmarshal(expect(t, owner, "GET", "/api/profile", nil, 200), &p)
	if p.Contacts == nil || len(p.Professional.Education) != 1 {
		t.Fatal("legacy update cleared details")
	}
	invitationID := uuid.NewString()
	_, err = a.DB.Exec(ctx, `INSERT INTO invitations(id,employer_id,candidate_id,title,message,salary_min,salary_max,status) VALUES($1,$2,$3,'Проверка','Тестовое приглашение',100000,200000,'accepted')`, invitationID, DemoID("employer@demo.fsp"), id)
	if err != nil {
		t.Fatal(err)
	}
	raw = expect(t, employer, "GET", "/api/candidates/"+id, nil, 200)
	if !bytes.Contains(raw, []byte("fsp_candidate")) {
		t.Fatal("consented contacts unavailable")
	}
	body["published"] = false
	expect(t, owner, "PUT", "/api/profile", body, 200)
	expect(t, employer, "GET", "/api/avatars/"+id, nil, 200)
	if _, err = a.DB.Exec(ctx, "UPDATE invitations SET status='declined' WHERE id=$1", invitationID); err != nil {
		t.Fatal(err)
	}
	expect(t, employer, "GET", "/api/avatars/"+id, nil, 404)
	expect(t, employer, "GET", "/api/candidates/"+id, nil, 404)
	expect(t, owner, "GET", "/api/avatars/"+id, nil, 200)
	expect(t, owner, "DELETE", "/api/avatar", nil, 200)
	expect(t, owner, "GET", "/api/avatars/"+id, nil, 404)
	companyID := DemoID("employer@demo.fsp")
	expect(t, owner, "GET", "/api/companies/"+companyID, nil, 200)
	expect(t, anon, "GET", "/api/companies/"+companyID, nil, 401)
	expect(t, owner, "PUT", "/api/company", map[string]string{"name": "Подмена", "description": "", "website": ""}, 403)
	var original Company
	json.Unmarshal(expect(t, employer, "GET", "/api/company", nil, 200), &original)
	defer func() {
		expect(t, employer, "PUT", "/api/company", map[string]any{"name": original.Name, "description": original.Description, "website": original.Website, "details": original.Details}, 200)
	}()
	companyBody := map[string]any{"name": "Команда для проверки", "description": "Демонстрационная команда", "website": "https://example.org", "details": map[string]any{"industry": "ИТ", "team_size": "11-50", "technologies": []string{"Go", "Redis"}, "contact_email": "hr@example.test", "telegram": "@fsp_team"}}
	expect(t, employer, "PUT", "/api/company", companyBody, 200)
	raw = expect(t, owner, "GET", "/api/companies/"+companyID, nil, 200)
	if !bytes.Contains(raw, []byte("hr@example.test")) || !bytes.Contains(raw, []byte("fsp_team")) {
		t.Fatal("public company details unavailable")
	}
	delete(companyBody, "details")
	expect(t, employer, "PUT", "/api/company", companyBody, 200)
	raw = expect(t, employer, "GET", "/api/company", nil, 200)
	if !bytes.Contains(raw, []byte("hr@example.test")) {
		t.Fatal("legacy company update erased details")
	}
	companyBody["website"] = "javascript:alert(1)"
	expect(t, employer, "PUT", "/api/company", companyBody, 400)
	raw = expect(t, employer, "GET", "/api/company", nil, 200)
	if strings.Contains(string(raw), "javascript:") {
		t.Fatal("invalid URL persisted")
	}
	expect(t, employer, "POST", "/api/avatar", map[string]string{"image": base64.StdEncoding.EncodeToString(avatarFixture(t, 32, 32))}, 200)
	expect(t, owner, "GET", "/api/avatars/"+companyID, nil, 200)
	expect(t, employer, "DELETE", "/api/avatar", nil, 200)
}
