package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Swek09/lct-fcp-2026/backend/internal/resume"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func DemoID(email string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("fsp-demo:"+email)).String()
}
func (a *App) Seed(ctx context.Context) error {
	if a.Config.Env == "production" {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("DemoPass2026!"), 12)
	if err != nil {
		return err
	}
	type candidate struct {
		email, name, headline, city, spec, grade string
		skills                                   []string
		score, years, achievements               int
	}
	people := []candidate{
		{"candidate@demo.fsp", "Александр Морозов", "Backend-разработчик · Go", "Москва", "backend", "middle", []string{"Go", "PostgreSQL", "Redis", "Docker"}, 92, 3, 2},
		{"elena@demo.fsp", "Елена Волкова", "Backend-разработчик · высоконагруженные системы", "Санкт-Петербург", "backend", "senior", []string{"Go", "PostgreSQL", "Kubernetes", "Redis"}, 96, 7, 3},
		{"maxim@demo.fsp", "Максим Лебедев", "Backend-разработчик · Python", "Казань", "backend", "middle", []string{"Python", "PostgreSQL", "Docker"}, 84, 3, 0},
		{"anna@demo.fsp", "Анна Соколова", "Frontend-разработчик · React", "Москва", "frontend", "middle", []string{"React", "TypeScript", "CSS", "HTML"}, 94, 4, 1},
		{"ivan@demo.fsp", "Иван Ковалёв", "Начинающий Go-разработчик", "Новосибирск", "backend", "junior", []string{"Go", "SQL", "Docker"}, 83, 1, 0},
		{"daria@demo.fsp", "Дарья Белова", "Продуктовый аналитик", "Екатеринбург", "analytics", "middle", []string{"SQL", "Python", "Excel"}, 91, 3, 2},
		{"nikita@demo.fsp", "Никита Орлов", "Frontend-разработчик", "Самара", "frontend", "junior", []string{"React", "JavaScript", "CSS"}, 84, 1, 0},
		{"olga@demo.fsp", "Ольга Миронова", "Аналитик данных · эксперименты", "Москва", "analytics", "senior", []string{"SQL", "Python"}, 96, 6, 1},
		{"sofia@demo.fsp", "София Смирнова", "Go-разработчик · платёжные сервисы", "Москва", "backend", "middle", []string{"Go", "PostgreSQL", "Docker"}, 88, 4, 0},
		{"andrey@demo.fsp", "Андрей Новиков", "Backend-разработчик · распределённые системы", "Пермь", "backend", "middle", []string{"Go", "Redis", "Kubernetes"}, 86, 3, 1},
		{"vera@demo.fsp", "Вера Павлова", "Frontend-разработчик · интерфейсы для бизнеса", "Санкт-Петербург", "frontend", "middle", []string{"React", "TypeScript", "CSS", "JavaScript"}, 90, 3, 0},
		{"pavel@demo.fsp", "Павел Фомин", "Аналитик · продуктовые метрики", "Казань", "analytics", "middle", []string{"SQL", "Python", "Excel"}, 86, 2, 0},
	}
	tx, err := a.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, p := range people {
		id := DemoID(p.email)
		_, err = tx.Exec(ctx, "INSERT INTO users(id,email,password_hash,role,name,verified) VALUES($1,$2,$3,'candidate',$4,true) ON CONFLICT DO NOTHING", id, p.email, string(hash), p.name)
		if err != nil {
			return err
		}
		achievements := []Achievement{}
		for i := 0; i < p.achievements; i++ {
			achievements = append(achievements, Achievement{[]string{"Кубок по спортивному программированию", "Командный хакатон", "Региональный чемпионат"}[i], 2025 + i%2, i + 1, "demo"})
		}
		raw, _ := json.Marshal(achievements)
		_, err = tx.Exec(ctx, `INSERT INTO profiles(user_id,headline,city,about,skills,experience_years,specialization,grade,test_score,published,achievements)
  VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,true,$10) ON CONFLICT DO NOTHING`, id, p.headline, p.city, "Люблю понятные задачи, надёжные решения и командную работу. Открыт к предложениям от продуктовых команд.", p.skills, p.years, p.spec, p.grade, p.score, raw)
		if err != nil {
			return err
		}
		// Only enrich untouched demo records. Real edits and deliberately cleared
		// normalized profiles must survive subsequent restarts.
		professional, err := json.Marshal(demoProfessional(p.email, p.spec, p.grade, p.skills, p.years))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE profiles SET professional=$2 WHERE user_id=$1 AND professional='{}'::jsonb", id, professional)
		if err != nil {
			return err
		}
	}
	email := "employer@demo.fsp"
	id := DemoID(email)
	_, err = tx.Exec(ctx, "INSERT INTO users(id,email,password_hash,role,name,verified) VALUES($1,$2,$3,'employer','Мария · Контур',true) ON CONFLICT DO NOTHING", id, email, string(hash))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO companies(user_id,name,description,website) VALUES($1,'Контур · демо','Создаём сервисы для бизнеса. Ищем людей, которые любят решать сложные задачи.','') ON CONFLICT DO NOTHING", id)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO invitations(id,employer_id,candidate_id,title,message,salary_min,salary_max)
 VALUES($1,$2,$3,'Go-разработчик в продуктовую команду','Привет! Нам понравились ваши результаты и опыт с Go. Предлагаем познакомиться с командой платформы: развиваем API и сервисы для бизнеса. Можно работать удалённо.',180000,250000) ON CONFLICT DO NOTHING`, DemoID("invitation"), id, DemoID("candidate@demo.fsp"))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO challenges(id,employer_id,title,body,specialization,grade,estimated_minutes)
 VALUES($1,$2,'Повторная доставка событий', 'Сервис получает одно событие несколько раз из очереди. Предложите способ избежать повторного списания денег: где хранить ключ идемпотентности, как обработать параллельные запросы и что делать при сбое после записи в БД. Достаточно описать подход, писать приложение не нужно.', 'backend','middle',10) ON CONFLICT DO NOTHING`, DemoID("challenge"), id)
	if err != nil {
		return err
	}
	for _, v := range []struct {
		key, title, description, spec, grade string
		skills                               []string
		min, max                             int
	}{
		{"vacancy-go", "Go-разработчик в команду платежей", "Развиваем платёжную платформу: API на Go, обработка событий, PostgreSQL и Redis. Предстоит проектировать идемпотентные операции, проводить ревью и повышать надёжность сервиса. Удалённая работа, гибкое начало дня, прозрачные цели и поддержка команды.", "backend", "middle", []string{"Go", "PostgreSQL", "Redis"}, 180000, 250000},
		{"vacancy-react", "React-разработчик · продукты для бизнеса", "Создаём интерфейсы для предпринимателей. Работаем с React и TypeScript, следим за доступностью, скоростью загрузки и качеством пользовательских сценариев. Предлагаем удалённую работу, ревью с командой и задачи, результат которых виден пользователям.", "frontend", "middle", []string{"React", "TypeScript", "CSS"}, 160000, 220000},
		{"vacancy-data", "Продуктовый аналитик", "Ищем аналитика для продуктовой команды. SQL и Python, определение метрик, проверка гипотез и A/B-эксперименты. Важно объяснять выводы и учитывать качество данных. Удалённый формат, работа вместе с разработчиками и продуктовым менеджером.", "analytics", "middle", []string{"SQL", "Python"}, 150000, 210000},
	} {
		_, err = tx.Exec(ctx, `INSERT INTO vacancies(id,employer_id,title,description,specialization,grade,skills,work_format,salary_min,salary_max,status) VALUES($1,$2,$3,$4,$5,$6,$7,'remote',$8,$9,'published') ON CONFLICT DO NOTHING`, DemoID(v.key), id, v.title, v.description, v.spec, v.grade, v.skills, v.min, v.max)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func demoProfessional(email, spec, grade string, skills []string, years int) resume.Data {
	type sample struct{ name, role, description, results string }
	projects := map[string]sample{
		"candidate@demo.fsp": {"API обработки заказов", "Backend-разработчик", "Демонстрационный проект: REST API заказов, идемпотентные платежи и обработка событий через outbox.", "Спроектировал транзакции, добавил интеграционные тесты и метрики задержек. Повторная доставка не создаёт второй заказ."},
		"elena@demo.fsp":     {"Платформа событий", "Технический лидер", "Демонстрационный проект: доставка событий между сервисами, обработка повторов и восстановление после отказа.", "Описала SLO, провела нагрузочные испытания и подготовила план миграции без остановки сервиса."},
		"maxim@demo.fsp":     {"Сервис обработки документов", "Python-разработчик", "Демонстрационный проект: асинхронная обработка документов и поиск результатов через API.", "Добавил повторяемые задания, валидацию входных данных и журнал ошибок."},
		"anna@demo.fsp":      {"Кабинет команды", "Frontend-разработчик", "Демонстрационный проект: управление задачами, фильтры, формы и согласование доступа в кабинете команды.", "Собрала доступные компоненты форм, обработку ошибок и отмену устаревших поисковых запросов."},
		"ivan@demo.fsp":      {"Каталог мероприятий", "Автор учебного проекта", "Демонстрационный учебный проект: каталог, регистрация участников и API на Go.", "Реализовал CRUD, миграции PostgreSQL и тесты основных обработчиков."},
		"daria@demo.fsp":     {"Аналитика воронки", "Продуктовый аналитик", "Демонстрационный проект: расчёт конверсий, когортного удержания и проверка продуктовой гипотезы.", "Зафиксировала определения метрик, проверила дубликаты событий и подготовила рекомендации по эксперименту."},
		"nikita@demo.fsp":    {"Интерфейс регистрации", "Автор учебного проекта", "Демонстрационный учебный проект: адаптивная регистрация и личный кабинет участника.", "Добавил валидацию полей, управление фокусом и сообщения об ошибках."},
		"olga@demo.fsp":      {"Система экспериментов", "Ведущий аналитик", "Демонстрационный проект: анализ A/B-тестов с сегментацией и контролем качества распределения.", "Подготовила критерии остановки, контроль множественных проверок и отчёты с доверительными интервалами."},
		"sofia@demo.fsp":     {"Платёжный шлюз", "Go-разработчик", "Демонстрационный проект: приём платежей и согласование статусов внешних провайдеров.", "Реализовала ключи идемпотентности, ограниченные повторы и сверку состояний."},
		"andrey@demo.fsp":    {"Очередь фоновых задач", "Backend-разработчик", "Демонстрационный проект: планирование фоновых задач, блокировки и восстановление после сбоев.", "Добавил дедлайны, дедупликацию и метрики очереди."},
		"vera@demo.fsp":      {"Интерфейс отчётности", "Frontend-разработчик", "Демонстрационный проект: таблицы, сравнение показателей и экспорт бизнес-отчётов.", "Реализовала виртуализацию строк, состояния загрузки и клавиатурную навигацию."},
		"pavel@demo.fsp":     {"Продуктовые метрики", "Аналитик данных", "Демонстрационный проект: витрины заказов, выручки и удержания для продуктовой команды.", "Проверил согласованность источников, исключил незрелые когорты и описал расчёты в SQL."},
	}
	s := projects[email]
	min := map[string]int{"junior": 80000, "middle": 180000, "senior": 300000}[grade]
	d := resume.Data{SalaryMin: min, SalaryMax: min + 60000, WorkFormats: []string{"remote", "hybrid"}, Employment: "full_time", Availability: "two_weeks", SearchStatus: "open", Languages: []string{"Русский — родной", "Английский — B2"}, Experience: []resume.Experience{{Company: "Продуктовая команда · демо", Role: s.role, Start: fmt.Sprintf("%d-01", 2026-years), Current: true, Description: s.description, Results: s.results}}, Projects: []resume.Project{{Name: s.name, Role: s.role, Description: s.description, Results: s.results, Stack: skills}}}
	if grade == "junior" {
		d.Employment = "internship"
		d.Availability = "immediately"
		d.Languages = []string{"Русский — родной", "Английский — B1"}
	}
	if spec == "analytics" {
		d.WorkFormats = []string{"remote"}
		d.Relocation = true
	}
	return d
}
