import { useEffect, useState, type FormEvent } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import {
  Link,
  useParams,
  useSearchParams,
  useLocation,
} from "react-router-dom";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  LockKeyhole,
  MapPin,
  Search,
  Send,
  ShieldCheck,
  SlidersHorizontal,
  Trophy,
} from "lucide-react";
import {
  api,
  type Candidate,
  type Invitation,
  date,
  type Question,
  grades,
  initials,
  queryClient,
  specs,
  plural,
} from "../api";
import { Button } from "../components/ui/button";
import {
  Empty,
  ErrorBox,
  Field,
  Loading,
  PageTitle,
  Success,
} from "../components/shared";
interface SearchResult {
  candidates: Candidate[];
  total: number;
  specialization: string;
  grade: string;
  skills: string[];
}
export function SearchPage() {
  const [params, setParams] = useSearchParams();
  const [need, setNeed] = useState(params.get("need") || "");
  const [advanced, setAdvanced] = useState(!!params.get("skills"));
  const appliedNeed = params.get("need") || "";
  useEffect(() => {
    setNeed(appliedNeed);
    setPreview(undefined);
  }, [appliedNeed]);
  const [preview, setPreview] = useState<{
    specialization: string;
    grade: string;
    questions: Question[];
  }>();
  const [previewError, setPreviewError] = useState<unknown>();
  const [previewBusy, setPreviewBusy] = useState(false);
  const previewTest = async () => {
    setPreviewBusy(true);
    setPreviewError(undefined);
    try {
      setPreview(await api("/assessments/preview", "POST", { need }));
    } catch (error) {
      setPreviewError(error);
    } finally {
      setPreviewBusy(false);
    }
  };
  const q = useQuery({
    queryKey: ["candidates", params.toString()],
    queryFn: () => api<SearchResult>("/candidates?" + params),
    placeholderData: keepPreviousData,
  });
  const update = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    next.delete("page");
    setParams(next);
  };
  const submit = (e: FormEvent) => {
    e.preventDefault();
    update("need", need);
  };
  return (
    <>
      <PageTitle eyebrow="Банк кандидатов" title="Найдите тех, кто умеет.">
        Специалисты с подтверждённым уровнем. Вы выбираете, с кем начать
        разговор.
      </PageTitle>
      <form className="need-search" onSubmit={submit}>
        <label htmlFor="need">
          <Search size={20} />
          <span>Кого ищем в команду?</span>
        </label>
        <div>
          <input
            id="need"
            value={need}
            onChange={(e) => setNeed(e.target.value)}
            placeholder="Например: middle backend, Go, PostgreSQL, Redis"
            maxLength={2000}
          />
          <Button type="submit">
            Подобрать
            <ArrowRight size={17} />
          </Button>
        </div>
        <small>
          Опишите направление, уровень и стек — мы выделим их из запроса.
        </small>
        {need.trim().length >= 3 && (
          <button
            type="button"
            className="text-link preview-link"
            disabled={previewBusy}
            onClick={previewTest}
          >
            {previewBusy
              ? "Формируем образец…"
              : "Как проверяется эта категория?"}
          </button>
        )}
      </form>
      <ErrorBox error={previewError} />
      {preview && (
        <details className="panel test-preview" open>
          <summary>
            Образец теста · {specs[preview.specialization]} ·{" "}
            {grades[preview.grade]}
          </summary>
          <p className="muted">
            6 заданий по общему плану категории. В попытке кандидата будут
            другие параметры. Стек из описания влияет на подбор; этот тест
            подтверждает специализацию и грейд.
          </p>
          <ol>
            {preview.questions.map((question) => (
              <li key={question.id}>
                <strong>{question.topic}</strong>
                <p>{question.prompt}</p>
              </li>
            ))}
          </ol>
        </details>
      )}
      <div className="filters">
        <label>
          <span>Специализация</span>
          <select
            value={params.get("specialization") || ""}
            onChange={(e) => update("specialization", e.target.value)}
          >
            <option value="">Все направления</option>
            {Object.entries(specs).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>Уровень</span>
          <select
            value={params.get("grade") || ""}
            onChange={(e) => update("grade", e.target.value)}
          >
            <option value="">Все уровни</option>
            {Object.entries(grades).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </select>
        </label>
        <label className="check fsp-filter">
          <input
            type="checkbox"
            checked={params.get("fsp") === "true"}
            onChange={(e) => update("fsp", e.target.checked ? "true" : "")}
          />
          <Trophy size={16} />
          <span>С достижениями ФСП</span>
        </label>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setAdvanced(!advanced)}
          aria-expanded={advanced}
        >
          <SlidersHorizontal size={16} />
          Стек
        </Button>
      </div>
      {advanced && (
        <form
          className="inline-form"
          onSubmit={(e) => {
            e.preventDefault();
            update(
              "skills",
              String(new FormData(e.currentTarget).get("skills") || ""),
            );
          }}
        >
          <Field label="Нужные навыки через запятую">
            <input
              name="skills"
              defaultValue={params.get("skills") || ""}
              placeholder="Go, PostgreSQL"
            />
          </Field>
          <Button variant="secondary" size="sm">
            Применить
          </Button>
        </form>
      )}
      <ErrorBox error={q.error} />
      <div className="results-heading">
        <h2>
          {q.data ? (
            <>
              Кандидаты <span>{q.data.total}</span>
            </>
          ) : (
            "Кандидаты"
          )}
        </h2>
        <div>
          {q.isFetching && <span>Обновляем…</span>}
          <span>По соответствию запросу</span>
        </div>
      </div>
      {q.data && params.size > 0 && (
        <div className="parsed-need">
          <span>В подборке:</span>
          {q.data.specialization && (
            <span className="tag">{specs[q.data.specialization]}</span>
          )}
          {q.data.grade && <span className="tag">{grades[q.data.grade]}</span>}
          {q.data.skills.map((s) => (
            <span className="tag" key={s}>
              {s}
            </span>
          ))}
          {params.get("fsp") === "true" && (
            <span className="tag">Достижения ФСП</span>
          )}
          <button
            onClick={() => {
              setParams({});
              setNeed("");
            }}
          >
            Сбросить
          </button>
        </div>
      )}
      {q.isPending ? (
        <Loading />
      ) : q.error && !q.data ? (
        <Button variant="secondary" onClick={() => void q.refetch()}>
          Повторить загрузку
        </Button>
      ) : q.data?.candidates.length ? (
        <div
          className={
            "candidate-list " + (q.isPlaceholderData ? "updating" : "")
          }
        >
          {q.data.candidates.map((c) => (
            <article className="candidate-row" key={c.user_id}>
              <div className="candidate-main">
                <span className="avatar">{initials(c.name)}</span>
                <div className="candidate-info">
                  <div className="candidate-name">
                    <Link
                      to={"/candidates/" + c.user_id}
                      state={{ returnTo: "/candidates?" + params.toString() }}
                    >
                      {c.name}
                    </Link>
                    <span className="tag grade">{grades[c.grade]}</span>
                  </div>
                  <p>{c.headline || specs[c.specialization]}</p>
                  <span className="location">
                    <MapPin size={13} />
                    {c.city || "Город не указан"}
                    <span>·</span>
                    {c.experience_years}{" "}
                    {plural(c.experience_years, ["год", "года", "лет"])} опыта
                  </span>
                  <div className="tags">
                    {c.skills.map((s) => (
                      <span className="skill" key={s}>
                        {s}
                      </span>
                    ))}
                  </div>
                </div>
                <div className="candidate-evidence">
                  <span>
                    <ShieldCheck size={17} />
                    Тест <strong>{c.test_score}/100</strong>
                  </span>
                  {c.last_practice_at && (
                    <span className="practice-date">
                      Практика: {date(c.last_practice_at)}
                    </span>
                  )}
                  {c.achievements.length > 0 ? (
                    <span>
                      <Trophy size={16} />
                      {c.achievements.length}{" "}
                      {plural(c.achievements.length, [
                        "достижение",
                        "достижения",
                        "достижений",
                      ])}{" "}
                      ФСП
                    </span>
                  ) : (
                    <span className="muted">Без истории ФСП</span>
                  )}
                  <Link
                    className="text-link"
                    to={"/candidates/" + c.user_id}
                    state={{ returnTo: "/candidates?" + params.toString() }}
                  >
                    Открыть профиль
                    <ArrowRight size={16} />
                  </Link>
                </div>
              </div>
              <details className="match-explanation">
                <summary>
                  <span className="match-score">
                    {c.match_score}
                    <small>/100</small>
                  </span>
                  Почему в подборке <span>+</span>
                </summary>
                <ul>
                  {c.reasons.map((r) => (
                    <li key={r}>
                      <Check size={14} />
                      {r}
                    </li>
                  ))}
                </ul>
              </details>
            </article>
          ))}
        </div>
      ) : (
        <Empty title="Пока нет точных совпадений">
          Попробуйте другой уровень или расширьте фильтры. Ваш запрос сохранён.
        </Empty>
      )}
      {q.data && q.data.total > 20 && (
        <div className="pagination">
          <Button
            variant="secondary"
            disabled={Number(params.get("page") || 1) <= 1}
            onClick={() => {
              const n = new URLSearchParams(params);
              n.set("page", String(Number(n.get("page") || 1) - 1));
              setParams(n);
            }}
          >
            Назад
          </Button>
          <span>Страница {params.get("page") || 1}</span>
          <Button
            variant="secondary"
            disabled={Number(params.get("page") || 1) * 20 >= q.data.total}
            onClick={() => {
              const n = new URLSearchParams(params);
              n.set("page", String(Number(n.get("page") || 1) + 1));
              setParams(n);
            }}
          >
            Далее
          </Button>
        </div>
      )}
      <p className="search-note">
        <ShieldCheck size={15} />
        Категория определяется тестированием. Навыки и опыт указаны кандидатами.
      </p>
    </>
  );
}
export function CandidateDetail() {
  const { id } = useParams();
  const location = useLocation();
  const returnTo =
    typeof location.state?.returnTo === "string" &&
    location.state.returnTo.startsWith("/candidates?")
      ? location.state.returnTo
      : "/candidates";
  const q = useQuery({
    queryKey: ["candidate", id],
    queryFn: () => api<Candidate>("/candidates/" + id),
    refetchInterval: 15000,
  });
  const invitations = useQuery({
    queryKey: ["invitations"],
    queryFn: () => api<Invitation[]>("/invitations"),
    refetchInterval: 15000,
  });
  const activeInvitation = invitations.data?.find(
    (i) => i.candidate_id === id && ["sent", "viewed"].includes(i.status),
  );
  const [error, setError] = useState<unknown>();
  const [sent, setSent] = useState(false);
  const [busy, setBusy] = useState(false);
  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    const f = Object.fromEntries(new FormData(e.currentTarget));
    try {
      await api("/invitations", "POST", {
        ...f,
        candidate_id: id,
        salary_min: Number(f.salary_min),
        salary_max: Number(f.salary_max),
      });
      setSent(true);
      await queryClient.invalidateQueries({ queryKey: ["invitations"] });
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <Link className="text-link back" to={returnTo}>
        <ArrowLeft size={16} />К подборке
      </Link>
      <ErrorBox error={q.error} />
      {q.isPending ? (
        <Loading />
      ) : (
        q.data && (
          <>
            <PageTitle eyebrow="Профиль кандидата" title={q.data.name}>
              {q.data.headline}
            </PageTitle>
            <div className="profile-layout">
              <section>
                <div className="candidate-detail-meta">
                  <span className="tag grade">{grades[q.data.grade]}</span>
                  <span>{specs[q.data.specialization]}</span>
                  <span>
                    <MapPin size={15} />
                    {q.data.city}
                  </span>
                </div>
                <div className="proof-strip">
                  <ShieldCheck size={24} />
                  <div>
                    <strong>Квалификация подтверждена</strong>
                    <span>
                      Результат тестирования — {q.data.test_score} из 100
                    </span>
                  </div>
                </div>
                {q.data.last_practice_at && (
                  <div className="practice-rule">
                    <Check size={18} />
                    <p>
                      Практика подтверждена {date(q.data.last_practice_at)}
                      <span>
                        Работодатель положительно оценил решение короткой
                        задачи. Грейд определяется отдельным тестированием.
                      </span>
                    </p>
                  </div>
                )}
                <div className="section-heading">
                  <h2>О специалисте</h2>
                </div>
                <p className="profile-about">
                  {q.data.about || "Кандидат пока не добавил описание."}
                </p>
                <div className="tags">
                  {q.data.skills.map((s) => (
                    <span className="skill" key={s}>
                      {s}
                    </span>
                  ))}
                </div>
                <p className="muted">
                  Опыт: {q.data.experience_years}{" "}
                  {plural(q.data.experience_years, ["год", "года", "лет"])} ·
                  сведения из резюме
                </p>
                <div className="section-heading">
                  <h2>Достижения ФСП</h2>
                </div>
                {q.data.achievements.length ? (
                  q.data.achievements.map((a, i) => (
                    <div className="achievement" key={i}>
                      <Trophy size={22} />
                      <div>
                        <strong>{a.title}</strong>
                        <span>
                          {a.year} · {a.place}-е место
                        </span>
                        <small>Демонстрационные данные</small>
                      </div>
                    </div>
                  ))
                ) : (
                  <p className="muted">
                    Истории ФСП пока нет. Уровень подтверждён тестированием.
                  </p>
                )}
                <div className="section-heading">
                  <h2>Контакты</h2>
                </div>
                {q.data.email ? (
                  <a className="text-link" href={"mailto:" + q.data.email}>
                    {q.data.email}
                  </a>
                ) : (
                  <div className="notice">
                    <LockKeyhole size={19} />
                    <p>
                      Контакты откроются, когда кандидат примет ваше
                      приглашение.
                    </p>
                  </div>
                )}
              </section>
              <aside>
                <div className="panel offer-panel">
                  <Send size={24} />
                  <h2>Начните знакомство</h2>
                  <p>
                    Расскажите о задаче и укажите зарплату. Публиковать вакансию
                    необязательно.
                  </p>
                  <ErrorBox error={error} />
                  {invitations.isPending ? (
                    <Loading />
                  ) : sent || activeInvitation ? (
                    <>
                      <Success>
                        {activeInvitation?.status === "viewed"
                          ? "Кандидат просмотрел приглашение"
                          : "Приглашение отправлено"}
                      </Success>
                      <Link className="text-link" to="/invitations">
                        Следить за статусом
                        <ArrowRight size={16} />
                      </Link>
                    </>
                  ) : (
                    <form className="form-stack" onSubmit={submit}>
                      <Field label="Предложение">
                        <input
                          name="title"
                          placeholder="Go-разработчик в продуктовую команду"
                          minLength={3}
                          maxLength={160}
                          required
                        />
                      </Field>
                      <Field label="О команде и задачах">
                        <textarea
                          name="message"
                          rows={5}
                          minLength={20}
                          maxLength={4000}
                          required
                          placeholder="Почему выбрали этого специалиста? Над чем предстоит работать?"
                        />
                      </Field>
                      <div className="salary-fields">
                        <Field label="Зарплата от, ₽">
                          <input
                            name="salary_min"
                            type="number"
                            min={1}
                            max={10000000}
                            step={1000}
                            placeholder="180000"
                            required
                          />
                        </Field>
                        <Field label="До, ₽">
                          <input
                            name="salary_max"
                            type="number"
                            min={1}
                            max={10000000}
                            step={1000}
                            placeholder="250000"
                            required
                          />
                        </Field>
                      </div>
                      <small>
                        В месяц, до вычета налогов. Кандидат увидит условия до
                        начала общения.
                      </small>
                      <Button disabled={busy}>
                        {busy ? "Отправляем…" : "Отправить приглашение"}
                        <ArrowRight size={16} />
                      </Button>
                    </form>
                  )}
                </div>
              </aside>
            </div>
          </>
        )
      )}
    </>
  );
}
