package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Swek09/lct-fcp-2026/backend/internal/assessment"
	"github.com/Swek09/lct-fcp-2026/backend/migrations"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type testClient struct {
	handler http.Handler
	cookie  *http.Cookie
	csrf    string
}

func (c *testClient) call(method, path string, body any) (int, []byte) {
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Requested-With", "fsp")
	r.Header.Set("Origin", "http://localhost:8080")
	if c.cookie != nil {
		r.AddCookie(c.cookie)
		r.Header.Set("X-CSRF-Token", c.csrf)
	}
	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, r)
	return w.Code, w.Body.Bytes()
}
func (c *testClient) login(t *testing.T, email, password string) {
	t.Helper()
	raw, _ := json.Marshal(map[string]string{"email": email, "password": password})
	r := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Requested-With", "fsp")
	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var u User
	_ = json.Unmarshal(w.Body.Bytes(), &u)
	c.csrf = u.CSRF
	c.cookie = w.Result().Cookies()[0]
}
func expect(t *testing.T, c *testClient, method, path string, body any, want int) []byte {
	t.Helper()
	code, raw := c.call(method, path, body)
	if code != want {
		t.Fatalf("%s %s: want %d got %d: %s", method, path, want, code, raw)
	}
	return raw
}
func TestIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL and TEST_REDIS_URL for real integration checks")
	}
	ctx := context.Background()
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	databaseURL = parsed.String()
	d, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	goose.SetBaseFS(migrations.Files)
	_ = goose.SetDialect("postgres")
	if err = goose.UpContext(ctx, d, "."); err != nil {
		t.Fatal(err)
	}
	_ = d.Close()
	cfg := Config{DatabaseURL: databaseURL, RedisURL: os.Getenv("TEST_REDIS_URL"), AppURL: "http://localhost:8080", Env: "test", Demo: true}
	a, err := New(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.rateLimitPrefix = schema + ":"
	defer func() {
		iter := a.Redis.Scan(ctx, 0, "rate:"+a.rateLimitPrefix+"*", 100).Iterator()
		for iter.Next(ctx) {
			a.Redis.Del(ctx, iter.Val())
		}
		if err := iter.Err(); err != nil {
			t.Errorf("clean rate counters: %v", err)
		}
	}()
	if err = a.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	h := a.Router()
	anon := &testClient{handler: h}
	employer := &testClient{handler: h}
	candidate := &testClient{handler: h}
	employer.login(t, "employer@demo.fsp", "DemoPass2026!")
	candidate.login(t, "candidate@demo.fsp", "DemoPass2026!")
	defer func() {
		for _, c := range []*testClient{employer, candidate} {
			if c.cookie != nil {
				a.Redis.Del(ctx, "session:"+digest(c.cookie.Value))
			}
		}
	}()
	t.Run("roles_and_contact_privacy", func(t *testing.T) {
		expect(t, anon, "GET", "/api/candidates", nil, 401)
		expect(t, candidate, "GET", "/api/candidates", nil, 403)
		expect(t, employer, "GET", "/api/profile", nil, 403)
		raw := expect(t, employer, "GET", "/api/candidates?specialization=backend&grade=middle", nil, 200)
		if bytes.Contains(raw, []byte("candidate@demo.fsp")) {
			t.Fatal("contact leaked before consent")
		}
		var search SearchResponse
		_ = json.Unmarshal(raw, &search)
		if search.Total != 4 {
			t.Fatalf("expected 4 middle backend, got %d", search.Total)
		}
		if search.Candidates[0].Grade != "middle" {
			t.Fatal("unverified grade filtering")
		}
		profile := expect(t, employer, "GET", "/api/candidates/"+DemoID("candidate@demo.fsp"), nil, 200)
		if bytes.Contains(profile, []byte("candidate@demo.fsp")) {
			t.Fatal("detail exposed email")
		}
		if !bytes.Contains(raw, []byte("Максим")) {
			t.Fatal("candidate without FSP excluded")
		}
		strict := expect(t, employer, "GET", "/api/candidates?specialization=backend&grade=middle&skills=Go,Redis", nil, 200)
		if bytes.Contains(strict, []byte("Максим")) {
			t.Fatal("explicit stack filter did not filter")
		}
		preview := expect(t, employer, "POST", "/api/assessments/preview", map[string]string{"need": "middle backend Go PostgreSQL"}, 200)
		if bytes.Contains(preview, []byte("expected")) {
			t.Fatal("answer key leaked in preview")
		}
		expect(t, candidate, "POST", "/api/assessments/preview", map[string]string{"need": "middle backend Go PostgreSQL"}, 403)
	})
	t.Run("CSRF_rejected", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/api/auth/logout", bytes.NewReader([]byte("{}")))
		r.AddCookie(candidate.cookie)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("missing CSRF accepted: %d", w.Code)
		}
		r = httptest.NewRequest("POST", "/api/auth/logout", bytes.NewReader([]byte("{}")))
		r.AddCookie(candidate.cookie)
		r.Header.Set("X-Requested-With", "fsp")
		r.Header.Set("X-CSRF-Token", candidate.csrf)
		r.Header.Set("Origin", "https://attacker.example")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("cross-origin mutation accepted")
		}
	})
	t.Run("invitation_state_and_salary", func(t *testing.T) {
		id := DemoID("invitation")
		expect(t, employer, "PATCH", "/api/invitations/"+id, map[string]string{"status": "accepted"}, 403)
		expect(t, candidate, "PATCH", "/api/invitations/"+id, map[string]string{"status": "viewed"}, 200)
		expect(t, candidate, "PATCH", "/api/invitations/"+id, map[string]string{"status": "accepted"}, 200)
		raw := expect(t, employer, "GET", "/api/candidates/"+DemoID("candidate@demo.fsp"), nil, 200)
		if !bytes.Contains(raw, []byte("candidate@demo.fsp")) {
			t.Fatal("accepted contact absent")
		}
		expect(t, candidate, "PATCH", "/api/invitations/"+id, map[string]string{"status": "declined"}, 409)
		body := map[string]any{"candidate_id": DemoID("maxim@demo.fsp"), "title": "Backend разработчик", "message": "Предлагаем работу в команде платформы на Go.", "salary_min": 200000, "salary_max": 100000}
		expect(t, employer, "POST", "/api/invitations", body, 400)
		body["salary_max"] = 300000
		expect(t, employer, "POST", "/api/invitations", body, 200)
		expect(t, employer, "POST", "/api/invitations", body, 409)
		if raw = expect(t, candidate, "GET", "/api/invitations", nil, 200); bytes.Contains(raw, []byte("Максим")) {
			t.Fatal("other candidate invitations leaked")
		}
	})
	var fresh *testClient
	t.Run("weekly_challenges_and_review_privacy", func(t *testing.T) {
		expect(t, anon, "GET", "/api/challenges", nil, 401)
		body := map[string]any{"title": "Проверка идемпотентности", "body": "Предложите безопасный способ обработки повторных событий, опишите транзакцию и проверку результата.", "specialization": "backend", "grade": "middle", "estimated_minutes": 10}
		expect(t, candidate, "POST", "/api/challenges", body, 403)
		expect(t, employer, "POST", "/api/challenges", body, 200)
		var feed ChallengeFeed
		if err := json.Unmarshal(expect(t, candidate, "GET", "/api/challenges", nil, 200), &feed); err != nil {
			t.Fatal(err)
		}
		var taskID string
		for _, c := range feed.Challenges {
			if c.Title == body["title"] {
				taskID = c.ID
			}
		}
		if taskID == "" {
			t.Fatal("matching category did not receive task")
		}
		answer := map[string]string{"answer": "Используем уникальный ключ события в PostgreSQL и одну транзакцию для записи ключа и обновления баланса. Повторный ключ не меняет баланс."}
		expect(t, employer, "POST", "/api/challenges/"+taskID+"/submissions", answer, 403)
		expect(t, candidate, "GET", "/api/challenges/"+taskID+"/submissions", nil, 403)
		expect(t, candidate, "POST", "/api/challenges/missing/submissions", answer, 404)
		expect(t, candidate, "POST", "/api/challenges/"+taskID+"/submissions", answer, 200)
		expect(t, candidate, "POST", "/api/challenges/"+taskID+"/submissions", answer, 409)
		expect(t, candidate, "POST", "/api/challenges/"+DemoID("challenge")+"/submissions", answer, 409)
		var p Profile
		_ = json.Unmarshal(expect(t, candidate, "GET", "/api/profile", nil, 200), &p)
		if p.LastPracticeAt != nil {
			t.Fatal("unreviewed answer changed profile freshness")
		}
		raw := expect(t, employer, "GET", "/api/challenges/"+taskID+"/submissions", nil, 200)
		if bytes.Contains(raw, []byte("candidate@demo.fsp")) {
			t.Fatal("submission leaked candidate contact")
		}
		var solutions []ChallengeSubmission
		if err := json.Unmarshal(raw, &solutions); err != nil || len(solutions) != 1 {
			t.Fatalf("solutions: %s", raw)
		}
		otherID := uuid.NewString()
		_, err := a.DB.Exec(ctx, "INSERT INTO users(id,email,password_hash,role,name,verified) SELECT $1,$2,password_hash,'employer','Другая компания',true FROM users WHERE id=$3", otherID, "other-"+otherID+"@example.test", DemoID("employer@demo.fsp"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = a.DB.Exec(ctx, "INSERT INTO companies(user_id,name) VALUES($1,'Другая компания')", otherID); err != nil {
			t.Fatal(err)
		}
		other := &testClient{handler: h}
		other.login(t, "other-"+otherID+"@example.test", "DemoPass2026!")
		defer a.Redis.Del(ctx, "session:"+digest(other.cookie.Value))
		expect(t, other, "GET", "/api/challenges/"+taskID+"/submissions", nil, 404)
		review := map[string]string{"status": "accepted", "feedback": "Хороший подход: одна транзакция исключает повторное списание."}
		expect(t, other, "PATCH", "/api/challenge-submissions/"+solutions[0].ID, review, 409)
		expect(t, candidate, "PATCH", "/api/challenge-submissions/"+solutions[0].ID, review, 403)
		expect(t, employer, "PATCH", "/api/challenge-submissions/"+solutions[0].ID, review, 200)
		expect(t, employer, "PATCH", "/api/challenge-submissions/"+solutions[0].ID, review, 409)
		_ = json.Unmarshal(expect(t, candidate, "GET", "/api/profile", nil, 200), &p)
		if p.LastPracticeAt == nil || p.Grade != "middle" || p.TestScore != 92 {
			t.Fatal("review did not preserve verified grade or update freshness")
		}
		_ = json.Unmarshal(expect(t, candidate, "GET", "/api/challenges", nil, 200), &feed)
		if feed.NextSubmissionAt == nil {
			t.Fatal("weekly cooldown missing")
		}
		for _, c := range feed.Challenges {
			if c.ID == taskID && (c.Submission == nil || c.Submission.Feedback != review["feedback"]) {
				t.Fatal("feedback missing from candidate history")
			}
		}
		// Another category and expired tasks remain inaccessible even with a known ID.
		if _, err = a.DB.Exec(ctx, "UPDATE challenges SET grade='senior' WHERE id=$1", DemoID("challenge")); err != nil {
			t.Fatal(err)
		}
		expect(t, candidate, "POST", "/api/challenges/"+DemoID("challenge")+"/submissions", answer, 404)
		if _, err = a.DB.Exec(ctx, "UPDATE challenges SET grade='middle',closes_at=now()-interval '1 second' WHERE id=$1", DemoID("challenge")); err != nil {
			t.Fatal(err)
		}
		expect(t, candidate, "POST", "/api/challenges/"+DemoID("challenge")+"/submissions", answer, 404)
		// After a week, concurrent requests to different tasks still allow only one.
		if _, err = a.DB.Exec(ctx, "UPDATE challenge_submissions SET submitted_at=now()-interval '8 days' WHERE candidate_id=$1", DemoID("candidate@demo.fsp")); err != nil {
			t.Fatal(err)
		}
		body["title"] = "Следующая недельная задача A"
		expect(t, employer, "POST", "/api/challenges", body, 200)
		body["title"] = "Следующая недельная задача B"
		expect(t, employer, "POST", "/api/challenges", body, 200)
		_ = json.Unmarshal(expect(t, candidate, "GET", "/api/challenges", nil, 200), &feed)
		var nextIDs []string
		for _, c := range feed.Challenges {
			if strings.HasPrefix(c.Title, "Следующая недельная задача") {
				nextIDs = append(nextIDs, c.ID)
			}
		}
		if len(nextIDs) != 2 {
			t.Fatal("new weekly tasks missing")
		}
		var wg sync.WaitGroup
		codes := make([]int, 2)
		for i, id := range nextIDs {
			wg.Add(1)
			go func(i int, id string) {
				defer wg.Done()
				codes[i], _ = candidate.call("POST", "/api/challenges/"+id+"/submissions", answer)
			}(i, id)
		}
		wg.Wait()
		if !((codes[0] == 200 && codes[1] == 409) || (codes[1] == 200 && codes[0] == 409)) {
			t.Fatalf("concurrent weekly limit: %v", codes)
		}
		for _, id := range nextIDs {
			_ = json.Unmarshal(expect(t, employer, "GET", "/api/challenges/"+id+"/submissions", nil, 200), &solutions)
			if len(solutions) == 0 {
				continue
			}
			review["status"] = "needs_work"
			expect(t, employer, "PATCH", "/api/challenge-submissions/"+solutions[0].ID, review, 200)
		}
		var afterReview Profile
		_ = json.Unmarshal(expect(t, candidate, "GET", "/api/profile", nil, 200), &afterReview)
		if afterReview.LastPracticeAt == nil || !afterReview.LastPracticeAt.Equal(*p.LastPracticeAt) {
			t.Fatal("negative review changed freshness")
		}
		maxim := &testClient{handler: h}
		maxim.login(t, "maxim@demo.fsp", "DemoPass2026!")
		defer a.Redis.Del(ctx, "session:"+digest(maxim.cookie.Value))
		expect(t, maxim, "POST", "/api/challenges/"+nextIDs[0]+"/submissions", answer, 200)
		if _, err = a.DB.Exec(ctx, "UPDATE challenge_submissions SET submitted_at=now()-interval '40 days' WHERE candidate_id=$1", DemoID("maxim@demo.fsp")); err != nil {
			t.Fatal(err)
		}
		_ = json.Unmarshal(expect(t, employer, "GET", "/api/challenges/"+nextIDs[0]+"/submissions", nil, 200), &solutions)
		review["status"] = "accepted"
		for _, s := range solutions {
			if s.CandidateID == DemoID("maxim@demo.fsp") {
				expect(t, employer, "PATCH", "/api/challenge-submissions/"+s.ID, review, 200)
			}
		}
		var maximProfile Profile
		_ = json.Unmarshal(expect(t, maxim, "GET", "/api/profile", nil, 200), &maximProfile)
		if maximProfile.LastPracticeAt == nil || maximProfile.LastPracticeAt.After(time.Now().Add(-39*24*time.Hour)) {
			t.Fatal("late review made an old solution fresh")
		}
		expect(t, maxim, "POST", "/api/challenges/"+nextIDs[1]+"/submissions", answer, 200)
		if _, err = a.DB.Exec(ctx, "UPDATE profiles SET specialization='frontend',grade='junior',last_practice_at=NULL WHERE user_id=$1", DemoID("maxim@demo.fsp")); err != nil {
			t.Fatal(err)
		}
		_ = json.Unmarshal(expect(t, employer, "GET", "/api/challenges/"+nextIDs[1]+"/submissions", nil, 200), &solutions)
		for _, s := range solutions {
			if s.CandidateID == DemoID("maxim@demo.fsp") {
				expect(t, employer, "PATCH", "/api/challenge-submissions/"+s.ID, review, 200)
			}
		}
		_ = json.Unmarshal(expect(t, maxim, "GET", "/api/profile", nil, 200), &maximProfile)
		if maximProfile.LastPracticeAt != nil {
			t.Fatal("review of another category changed current freshness")
		}
	})
	var freshID, email string
	t.Run("registration_and_email_verification", func(t *testing.T) {
		email = "test-" + uuid.NewString() + "@example.test"
		body := map[string]any{"name": "Тестовый участник", "email": email, "password": "StrongTest2026!", "role": "candidate", "consent": true}
		expect(t, anon, "POST", "/api/auth/register", body, 200)
		expect(t, anon, "POST", "/api/auth/login", map[string]string{"email": email, "password": "StrongTest2026!"}, 403)
		var letter string
		if err := a.DB.QueryRow(ctx, "SELECT body FROM mail_outbox WHERE recipient=$1 ORDER BY id DESC LIMIT 1", email).Scan(&letter); err != nil {
			t.Fatal(err)
		}
		tok := regexp.MustCompile("token=([a-f0-9]+)").FindStringSubmatch(letter)[1]
		expect(t, anon, "POST", "/api/auth/verify", map[string]string{"token": tok}, 200)
		expect(t, anon, "POST", "/api/auth/verify", map[string]string{"token": tok}, 400)
		fresh = &testClient{handler: h}
		fresh.login(t, email, "StrongTest2026!")
		raw := expect(t, fresh, "GET", "/api/profile", nil, 200)
		var p Profile
		_ = json.Unmarshal(raw, &p)
		freshID = p.UserID
		if p.Grade != "" {
			t.Fatal("self-assigned verified grade")
		}
		expect(t, fresh, "PATCH", "/api/invitations/"+DemoID("invitation"), map[string]string{"status": "accepted"}, 409)
	})
	if fresh == nil {
		t.Fatal("registration prerequisite failed")
	}
	defer func() {
		if fresh.cookie != nil {
			a.Redis.Del(ctx, "session:"+digest(fresh.cookie.Value))
		}
	}()
	t.Run("assessment_failure_retry_success_and_cooldown", func(t *testing.T) {
		start := map[string]string{"specialization": "frontend", "grade": "senior"}
		raw := expect(t, fresh, "POST", "/api/assessments", start, 200)
		if bytes.Contains(raw, []byte("expected")) {
			t.Fatal("answer key leaked")
		}
		var at Attempt
		_ = json.Unmarshal(raw, &at)
		expect(t, candidate, "POST", "/api/assessments/"+at.ID+"/submit", map[string]any{"answers": map[string]string{}}, 404)
		expect(t, fresh, "POST", "/api/assessments", start, 409)
		expect(t, fresh, "POST", "/api/assessments/"+at.ID+"/submit", map[string]any{"answers": map[string]string{}}, 200)
		expect(t, fresh, "POST", "/api/assessments", start, 409)
		start["grade"] = "junior"
		raw = expect(t, fresh, "POST", "/api/assessments", start, 200)
		_ = json.Unmarshal(raw, &at)
		var privateRaw []byte
		if err := a.DB.QueryRow(ctx, "SELECT questions FROM assessments WHERE id=$1", at.ID).Scan(&privateRaw); err != nil {
			t.Fatal(err)
		}
		var qs []assessment.PrivateQuestion
		_ = json.Unmarshal(privateRaw, &qs)
		answers := map[string]string{}
		for _, q := range qs {
			answers[q.ID] = q.Expected
		}
		raw = expect(t, fresh, "POST", "/api/assessments/"+at.ID+"/submit", map[string]any{"answers": answers}, 200)
		if !bytes.Contains(raw, []byte("\"passed\":true")) {
			t.Fatal("valid answers not passed")
		}
		expect(t, fresh, "POST", "/api/assessments/"+at.ID+"/submit", map[string]any{"answers": answers}, 409)
		start["grade"] = "middle"
		expect(t, fresh, "POST", "/api/assessments", start, 409)
		pRaw := expect(t, fresh, "GET", "/api/profile", nil, 200)
		var p Profile
		_ = json.Unmarshal(pRaw, &p)
		if p.Grade != "junior" || p.Specialization != "frontend" || p.TestScore != 100 || p.NextGradeChangeAt == nil {
			t.Fatal("category or cooldown not persisted")
		}
		evidenceRaw := expect(t, fresh, "GET", "/api/profile/evidence", nil, 200)
		var evidence QualificationEvidence
		if err := json.Unmarshal(evidenceRaw, &evidence); err != nil {
			t.Fatal(err)
		}
		if evidence.Source != "assessment" || len(evidence.Topics) != 6 || evidence.ConfirmedAt == nil {
			t.Fatal("real assessment evidence missing")
		}
		if bytes.Contains(evidenceRaw, []byte("expected")) || bytes.Contains(evidenceRaw, []byte("answers")) {
			t.Fatal("evidence exposed private answer key")
		}
		expect(t, employer, "GET", "/api/candidates/"+freshID, nil, 404)
		expect(t, fresh, "PUT", "/api/profile", map[string]any{"name": "Тестовый участник", "headline": "Frontend", "city": "Москва", "about": "Профиль без ФСП", "skills": []string{"React"}, "experience_years": 1, "published": true}, 200)
		expect(t, employer, "GET", "/api/candidates/"+freshID, nil, 200)
	})
	t.Run("vacancies_shortlist_consent_and_isolation", func(t *testing.T) {
		body := map[string]any{"title": "Frontend в продуктовую команду", "description": "Разрабатываем доступные интерфейсы на React. Нужен разработчик для работы с командой и проверки пользовательских сценариев.", "specialization": "frontend", "grade": "junior", "skills": []string{"React", "TypeScript"}, "work_format": "remote", "city": "", "salary_min": 100000, "salary_max": 160000}
		expect(t, anon, "GET", "/api/vacancies", nil, 401)
		expect(t, fresh, "POST", "/api/vacancies", body, 403)
		body["salary_max"] = 90000
		expect(t, employer, "POST", "/api/vacancies", body, 400)
		body["salary_max"] = 160000
		raw := expect(t, employer, "POST", "/api/vacancies", body, 200)
		var created struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &created); err != nil || created.ID == "" {
			t.Fatal("missing vacancy ID")
		}
		id := created.ID
		expect(t, fresh, "GET", "/api/vacancies/"+id, nil, 404)
		otherID := uuid.NewString()
		otherEmail := "company-" + otherID + "@example.test"
		if _, err := a.DB.Exec(ctx, "INSERT INTO users(id,email,password_hash,role,name,verified) SELECT $1,$2,password_hash,'employer','Другая компания',true FROM users WHERE id=$3", otherID, otherEmail, DemoID("employer@demo.fsp")); err != nil {
			t.Fatal(err)
		}
		if _, err := a.DB.Exec(ctx, "INSERT INTO companies(user_id,name) VALUES($1,'Другая компания')", otherID); err != nil {
			t.Fatal(err)
		}
		other := &testClient{handler: h}
		other.login(t, otherEmail, "DemoPass2026!")
		defer a.Redis.Del(ctx, "session:"+digest(other.cookie.Value))
		expect(t, other, "GET", "/api/vacancies/"+id, nil, 404)
		expect(t, other, "PUT", "/api/vacancies/"+id, body, 404)
		expect(t, other, "PATCH", "/api/vacancies/"+id, map[string]string{"status": "published"}, 404)
		expect(t, employer, "PATCH", "/api/vacancies/"+id, map[string]string{"status": "published"}, 200)
		expect(t, fresh, "GET", "/api/vacancies/"+id, nil, 200)
		if _, err := a.DB.Exec(ctx, "UPDATE profiles SET published=false WHERE user_id=$1", freshID); err != nil {
			t.Fatal(err)
		}
		expect(t, employer, "GET", "/api/candidates/"+freshID, nil, 404)
		expect(t, fresh, "POST", "/api/vacancies/"+id+"/applications", map[string]any{"cover_letter": "Интересуют задачи команды", "consent": false}, 400)
		apply := map[string]any{"cover_letter": "Интересуют задачи команды", "consent": true}
		expect(t, fresh, "POST", "/api/vacancies/"+id+"/applications", apply, 200)
		expect(t, fresh, "POST", "/api/vacancies/"+id+"/applications", apply, 409)
		raw = expect(t, employer, "GET", "/api/applications?vacancy_id="+id, nil, 200)
		var apps []Application
		if err := json.Unmarshal(raw, &apps); err != nil {
			t.Fatal(err)
		}
		if len(apps) != 1 || apps[0].CandidateEmail != email {
			t.Fatal("self-application did not grant scoped contact")
		}
		appID := apps[0].ID
		if raw := expect(t, other, "GET", "/api/applications?vacancy_id="+id, nil, 200); string(raw) != "[]\n" && string(raw) != "[]" {
			t.Fatalf("foreign applications leaked: %s", raw)
		}
		expect(t, other, "GET", "/api/candidates/"+freshID, nil, 404)
		expect(t, employer, "GET", "/api/candidates/"+freshID, nil, 200)
		expect(t, employer, "GET", "/api/candidates/"+freshID+"/evidence", nil, 200)
		expect(t, employer, "POST", "/api/shortlist/"+freshID, nil, 200)
		expect(t, fresh, "GET", "/api/shortlist", nil, 403)
		if raw := expect(t, other, "GET", "/api/shortlist", nil, 200); bytes.Contains(raw, []byte(freshID)) {
			t.Fatal("shortlist leaked")
		}
		if raw := expect(t, employer, "GET", "/api/shortlist", nil, 200); !bytes.Contains(raw, []byte(freshID)) {
			t.Fatal("shortlist missing saved candidate")
		}
		if raw := expect(t, employer, "GET", "/api/candidates?saved=true", nil, 200); !bytes.Contains(raw, []byte(freshID)) {
			t.Fatal("saved search did not honor self-application access")
		}
		expect(t, other, "PATCH", "/api/applications/"+appID, map[string]string{"status": "shortlisted"}, 409)
		expect(t, fresh, "PATCH", "/api/applications/"+appID, map[string]string{"status": "shortlisted"}, 403)
		expect(t, employer, "PATCH", "/api/applications/"+appID, map[string]string{"status": "withdrawn"}, 403)
		expect(t, fresh, "PATCH", "/api/applications/"+appID, map[string]string{"status": "withdrawn"}, 200)
		expect(t, employer, "GET", "/api/candidates/"+freshID, nil, 404)
		expect(t, employer, "GET", "/api/candidates/"+freshID+"/evidence", nil, 404)
		if raw := expect(t, employer, "GET", "/api/applications?vacancy_id="+id, nil, 200); bytes.Contains(raw, []byte(email)) {
			t.Fatal("withdrawal did not revoke contact")
		}
		if raw := expect(t, employer, "GET", "/api/shortlist", nil, 200); bytes.Contains(raw, []byte(freshID)) {
			t.Fatal("hidden withdrawn profile still visible in shortlist")
		}
		var wg sync.WaitGroup
		statuses := make(chan int, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				code, _ := fresh.call("POST", "/api/vacancies/"+id+"/applications", apply)
				statuses <- code
			}()
		}
		wg.Wait()
		close(statuses)
		counts := map[int]int{}
		for s := range statuses {
			counts[s]++
		}
		if counts[200] != 1 || counts[409] != 1 {
			t.Fatalf("concurrent application: %v", counts)
		}
		expect(t, employer, "PATCH", "/api/applications/"+appID, map[string]string{"status": "shortlisted"}, 200)
		expect(t, employer, "PATCH", "/api/applications/"+appID, map[string]string{"status": "viewed"}, 409)
		expect(t, employer, "PATCH", "/api/vacancies/"+id, map[string]string{"status": "closed"}, 200)
		expect(t, fresh, "PATCH", "/api/applications/"+appID, map[string]string{"status": "withdrawn"}, 200)
		expect(t, fresh, "POST", "/api/vacancies/"+id+"/applications", apply, 404)
		expect(t, employer, "DELETE", "/api/shortlist/"+freshID, nil, 200)
		expect(t, employer, "GET", "/api/overview", nil, 200)
		expect(t, fresh, "GET", "/api/overview", nil, 200)
		expect(t, anon, "GET", "/api/overview", nil, 401)
		if _, err := a.DB.Exec(ctx, "UPDATE profiles SET published=true WHERE user_id=$1", freshID); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("expiry_uses_database_deadline", func(t *testing.T) {
		_, err := a.DB.Exec(ctx, "UPDATE profiles SET next_grade_change_at=NULL WHERE user_id=$1", freshID)
		if err != nil {
			t.Fatal(err)
		}
		raw := expect(t, fresh, "POST", "/api/assessments", map[string]string{"specialization": "backend", "grade": "middle"}, 200)
		var at Attempt
		_ = json.Unmarshal(raw, &at)
		_, err = a.DB.Exec(ctx, "UPDATE assessments SET expires_at=now()-interval '1 second' WHERE id=$1", at.ID)
		if err != nil {
			t.Fatal(err)
		}
		raw = expect(t, fresh, "POST", "/api/assessments/"+at.ID+"/submit", map[string]any{"answers": map[string]string{}}, 200)
		if !bytes.Contains(raw, []byte("\"passed\":false")) {
			t.Fatal("expired attempt passed")
		}
		var grade string
		_ = a.DB.QueryRow(ctx, "SELECT grade FROM profiles WHERE user_id=$1", freshID).Scan(&grade)
		if grade != "junior" {
			t.Fatal("failure demoted grade")
		}
	})
	t.Run("own_pdf_and_password_reset_revokes_sessions", func(t *testing.T) {
		raw := expect(t, fresh, "GET", "/api/profile/pdf", nil, 200)
		if !bytes.HasPrefix(raw, []byte("%PDF")) {
			t.Fatal("invalid PDF")
		}
		expect(t, employer, "GET", "/api/profile/pdf", nil, 403)
		expect(t, anon, "POST", "/api/auth/forgot", map[string]string{"email": email}, 200)
		var letter string
		_ = a.DB.QueryRow(ctx, "SELECT body FROM mail_outbox WHERE recipient=$1 ORDER BY id DESC LIMIT 1", email).Scan(&letter)
		tok := regexp.MustCompile("token=([a-f0-9]+)").FindStringSubmatch(letter)[1]
		body := map[string]string{"token": tok, "password": "ChangedPass2026!"}
		expect(t, anon, "POST", "/api/auth/reset", body, 200)
		expect(t, anon, "POST", "/api/auth/reset", body, 400)
		expect(t, fresh, "GET", "/api/auth/me", nil, 401)
		// A stale cookie must not trap a user at the login screen after reset.
		fresh.csrf = ""
		expect(t, fresh, "POST", "/api/auth/login", map[string]string{"email": email, "password": "ChangedPass2026!"}, 200)
		expect(t, anon, "POST", "/api/auth/login", map[string]string{"email": email, "password": "StrongTest2026!"}, 401)
	})
	t.Run("openapi_documents_operations", func(t *testing.T) {
		raw := expect(t, anon, "GET", "/api/openapi.json", nil, 200)
		if !bytes.Contains(raw, []byte("/api/assessments/{id}/submit")) {
			t.Fatal("missing operation")
		}
		if bytes.Contains(raw, []byte("\"expected\"")) {
			t.Fatal("private answers in schema")
		}
		if !bytes.Contains(raw, []byte("application/pdf")) {
			t.Fatal("PDF response format missing")
		}
	})
	t.Run("session_store_loss_does_not_erase_grade", func(t *testing.T) {
		_ = a.Redis.Del(ctx, "session:"+digest(candidate.cookie.Value)).Err()
		expect(t, candidate, "GET", "/api/auth/me", nil, 401)
		var score int
		_ = a.DB.QueryRow(ctx, "SELECT test_score FROM profiles WHERE user_id=$1", DemoID("candidate@demo.fsp")).Scan(&score)
		if score != 92 {
			t.Fatal("grade lost with session")
		}
	})
}
