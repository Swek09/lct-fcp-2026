import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  ArrowRight,
  Download,
  Eye,
  EyeOff,
  ShieldCheck,
  Trophy,
} from "lucide-react";
import {
  api,
  type Config,
  date,
  grades,
  initials,
  type Profile,
  queryClient,
  specs,
} from "../api";
import { Button } from "../components/ui/button";
import { EvidencePassport } from "../components/Evidence";
import {
  ErrorBox,
  Field,
  Loading,
  PageTitle,
  Success,
} from "../components/shared";
export function CandidateProfile() {
  const q = useQuery({
    queryKey: ["profile"],
    queryFn: () => api<Profile>("/profile"),
  });
  const cfg = useQuery({
    queryKey: ["config"],
    queryFn: () => api<Config>("/config"),
  });
  const [error, setError] = useState<unknown>();
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setBusy(true);
    setMessage("");
    setError(undefined);
    const f = Object.fromEntries(new FormData(e.currentTarget));
    try {
      await api("/profile", "PUT", {
        ...f,
        skills: String(f.skills)
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean),
        experience_years: Number(f.experience_years),
        published: f.published === "on",
      });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["profile"] }),
        queryClient.invalidateQueries({ queryKey: ["me"] }),
      ]);
      setMessage("Изменения сохранены");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  const fsp = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    try {
      await api(
        "/profile/fsp",
        "POST",
        Object.fromEntries(new FormData(e.currentTarget)),
      );
      await queryClient.invalidateQueries({ queryKey: ["profile"] });
      setMessage("Достижения обновлены");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <PageTitle
        eyebrow="Личный кабинет"
        title="Ваш профессиональный профиль"
        action={
          <Button asChild variant="secondary">
            <a href="/api/profile/pdf" download>
              <Download size={16} />
              PDF-профиль
            </a>
          </Button>
        }
      >
        Покажите, что вы умеете. Следующий шаг сделает работодатель.
      </PageTitle>
      <ErrorBox error={q.error || error} />
      {message && <Success>{message}</Success>}
      {q.isPending ? (
        <Loading />
      ) : (
        q.data && (
          <div className="profile-layout">
            <section>
              <EvidencePassport />
              <div className="profile-summary">
                <span className="avatar large">{initials(q.data.name)}</span>
                <div>
                  <h2>{q.data.name}</h2>
                  <p>
                    {q.data.headline || "Добавьте профессиональный заголовок"}
                  </p>
                  <span className="muted">
                    {q.data.city || "Город не указан"}
                  </span>
                </div>
              </div>
              <form className="form-stack" onSubmit={submit}>
                <div className="section-heading">
                  <h2>О вас</h2>
                  <span>01 / Профиль</span>
                </div>
                <div className="form-grid">
                  <Field label="Имя и фамилия">
                    <input
                      name="name"
                      defaultValue={q.data.name}
                      required
                      minLength={2}
                      maxLength={100}
                    />
                  </Field>
                  <Field label="Город">
                    <input
                      name="city"
                      defaultValue={q.data.city}
                      maxLength={100}
                      placeholder="Например, Москва"
                    />
                  </Field>
                </div>
                <Field label="Профессиональный заголовок">
                  <input
                    name="headline"
                    defaultValue={q.data.headline}
                    maxLength={150}
                    placeholder="Backend-разработчик · Go"
                  />
                </Field>
                <Field
                  label="Навыки"
                  hint="Через запятую. Эти сведения видны как ваше самоописание; категорию подтверждает тест."
                >
                  <input
                    name="skills"
                    defaultValue={q.data.skills.join(", ")}
                    placeholder="Go, PostgreSQL, Redis"
                  />
                </Field>
                <Field label="Опыт в годах">
                  <input
                    className="short-input"
                    name="experience_years"
                    type="number"
                    min={0}
                    max={60}
                    defaultValue={q.data.experience_years}
                  />
                </Field>
                <Field label="О себе">
                  <textarea
                    name="about"
                    rows={5}
                    defaultValue={q.data.about}
                    maxLength={4000}
                    placeholder="Над чем работали и какие задачи интересны сейчас?"
                  />
                </Field>
                <div className="section-heading">
                  <h2>Видимость</h2>
                  <span>02 / Приватность</span>
                </div>
                <label className="check publication">
                  <input
                    type="checkbox"
                    name="published"
                    defaultChecked={q.data.published}
                  />
                  <span>
                    <strong>Разрешаю публикацию профиля работодателям</strong>
                    <small>
                      Имя, резюме и результаты будут доступны после
                      подтверждения уровня. Почта откроется компании, чьё
                      приглашение вы примете или на чью вакансию откликнетесь с
                      согласием.
                    </small>
                  </span>
                </label>
                <Button disabled={busy}>
                  {busy ? "Сохраняем…" : "Сохранить профиль"}
                </Button>
              </form>
            </section>
            <aside className="profile-aside">
              <div className="panel verification">
                <ShieldCheck size={26} />
                <span className="eyebrow">Подтверждённый уровень</span>
                <h2>
                  {q.data.grade ? grades[q.data.grade] : "Пока не определён"}
                </h2>
                <p>
                  {q.data.specialization
                    ? specs[q.data.specialization]
                    : "Пройдите тест, чтобы получить категорию и появиться в поиске."}
                </p>
                {q.data.grade && (
                  <div className="score-line">
                    <span>Результат тестирования</span>
                    <strong>
                      {q.data.test_score}
                      <small>/100</small>
                    </strong>
                  </div>
                )}
                <Button
                  asChild
                  variant={q.data.grade ? "secondary" : "default"}
                >
                  <Link to="/assessments">
                    {q.data.grade ? "Результаты и тесты" : "Подтвердить навыки"}
                    <ArrowRight size={16} />
                  </Link>
                </Button>
                {q.data.next_grade_change_at && (
                  <small>
                    Следующая смена уровня — {date(q.data.next_grade_change_at)}
                  </small>
                )}
              </div>
              <div className="visibility-state">
                {q.data.published && q.data.grade ? (
                  <Eye size={19} />
                ) : (
                  <EyeOff size={19} />
                )}
                <p>
                  {q.data.published && q.data.grade
                    ? "Работодатели могут найти ваш профиль"
                    : q.data.published
                      ? "Профиль появится в поиске после теста"
                      : "Профиль скрыт из поиска"}
                </p>
              </div>
              <div className="panel practice-summary">
                <h3>Практика навыков</h3>
                <p className="muted">
                  {q.data.last_practice_at
                    ? `Последнее решение подтверждено работодателем ${date(q.data.last_practice_at)}.`
                    : "Решайте короткие задачи от команд и получайте обратную связь."}
                </p>
                <Link className="text-link" to="/challenges">
                  Короткие задания
                  <ArrowRight size={16} />
                </Link>
              </div>
              <div className="panel">
                <div className="inline-heading">
                  <Trophy size={20} />
                  <h3>Достижения ФСП</h3>
                </div>
                {q.data.achievements.length ? (
                  q.data.achievements.map((a, i) => (
                    <div className="achievement" key={i}>
                      <span>{String(a.place).padStart(2, "0")}</span>
                      <div>
                        <strong>{a.title}</strong>
                        <small>
                          {a.year} · {a.place}-е место · демо
                        </small>
                      </div>
                    </div>
                  ))
                ) : (
                  <p className="muted">
                    Пока без истории ФСП. Вы можете подтвердить навыки тестом и
                    получать предложения.
                  </p>
                )}
                {cfg.data?.demo ? (
                  <form className="form-stack" onSubmit={fsp}>
                    <Field
                      label="Демонстрационный ФСП ID"
                      hint="DEMO-1001 или DEMO-1002. Пустое поле удалит связь."
                    >
                      <input
                        name="id"
                        defaultValue={q.data.fsp_id}
                        placeholder="DEMO-1001"
                      />
                    </Field>
                    <Button variant="secondary" size="sm" disabled={busy}>
                      Обновить связь
                    </Button>
                    <small>
                      Реестр ФСП ещё не подключён. Здесь используются тестовые
                      достижения.
                    </small>
                  </form>
                ) : (
                  <p className="muted">Подключение реестра ФСП готовится.</p>
                )}
              </div>
            </aside>
          </div>
        )
      )}
    </>
  );
}
