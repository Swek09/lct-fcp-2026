import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight, CheckCircle2, Clock3, ShieldCheck } from "lucide-react";
import {
  api,
  type Attempt,
  date,
  grades,
  type Profile,
  queryClient,
  specs,
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
export function Assessments() {
  const q = useQuery({
    queryKey: ["assessments"],
    queryFn: () => api<Attempt[]>("/assessments"),
  });
  const p = useQuery({
    queryKey: ["profile"],
    queryFn: () => api<Profile>("/profile"),
  });
  const [spec, setSpec] = useState("backend");
  const [grade, setGrade] = useState("junior");
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<{
    attempt: Attempt;
    message: string;
    topics: Record<string, boolean>;
  }>();
  const active = q.data?.find((a) => !a.submitted_at);
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, []);
  useEffect(() => {
    if (!active) return;
    try {
      const v = sessionStorage.getItem("assessment:" + active.id);
      setAnswers(v ? JSON.parse(v) : {});
    } catch {
      setAnswers({});
    }
  }, [active?.id]);
  const setAnswer = (id: string, value: string) => {
    const next = { ...answers, [id]: value };
    setAnswers(next);
    if (active)
      sessionStorage.setItem("assessment:" + active.id, JSON.stringify(next));
  };
  const start = async () => {
    setBusy(true);
    setError(undefined);
    setResult(undefined);
    try {
      await api("/assessments", "POST", { specialization: spec, grade });
      await queryClient.invalidateQueries({ queryKey: ["assessments"] });
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  const submit = async () => {
    if (!active) return;
    setBusy(true);
    setError(undefined);
    try {
      const r = await api<{
        attempt: Attempt;
        message: string;
        topics: Record<string, boolean>;
      }>("/assessments/" + active.id + "/submit", "POST", { answers });
      sessionStorage.removeItem("assessment:" + active.id);
      setResult(r);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["assessments"] }),
        queryClient.invalidateQueries({ queryKey: ["profile"] }),
      ]);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  const seconds = active
    ? Math.max(
        0,
        Math.floor((new Date(active.expires_at).getTime() - now) / 1000),
      )
    : 0;
  const locked =
    p.data?.next_grade_change_at &&
    new Date(p.data.next_grade_change_at).getTime() > now;
  return (
    <>
      <PageTitle
        eyebrow="Проверка квалификации"
        title="Навыки, подтверждённые делом"
      >
        Выберите направление и уровень. Результат определит вашу категорию.
      </PageTitle>
      <ErrorBox error={q.error || p.error || error} />
      {q.isPending ? (
        <Loading />
      ) : active ? (
        <div className="assessment-layout">
          <section className="question-list">
            <div className="test-header">
              <div>
                <h2>
                  {specs[active.specialization]} · {grades[active.grade]}
                </h2>
                <p>
                  {Object.values(answers).filter(Boolean).length} из{" "}
                  {active.questions.length} ответов
                </p>
              </div>
              <span className={"timer " + (seconds < 120 ? "urgent" : "")}>
                <Clock3 size={18} />
                {String(Math.floor(seconds / 60)).padStart(2, "0")}:
                {String(seconds % 60).padStart(2, "0")}
              </span>
            </div>
            {seconds === 0 && (
              <div className="notice error" role="alert">
                Время истекло. Завершите попытку, чтобы увидеть результат.
              </div>
            )}
            {active.questions.map((question, i) => (
              <fieldset
                className="question"
                key={question.id}
                disabled={busy || seconds === 0}
              >
                <legend>
                  <span className="question-number">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                  <span>
                    <small>{question.topic}</small>
                    {question.prompt}
                  </span>
                </legend>
                {question.kind === "choice" ? (
                  <div className="options">
                    {question.options.map((option, j) => (
                      <label
                        className={
                          "option " +
                          (answers[question.id] === option ? "selected" : "")
                        }
                        key={option}
                      >
                        <input
                          type="radio"
                          name={question.id}
                          value={option}
                          checked={answers[question.id] === option}
                          onChange={() => setAnswer(question.id, option)}
                        />
                        <span className="option-letter">
                          {String.fromCharCode(65 + j)}
                        </span>
                        <span>{option}</span>
                      </label>
                    ))}
                  </div>
                ) : (
                  <Field label="Ваш ответ">
                    <input
                      inputMode="decimal"
                      autoComplete="off"
                      value={answers[question.id] || ""}
                      onChange={(e) => setAnswer(question.id, e.target.value)}
                      placeholder="Введите число"
                    />
                  </Field>
                )}
              </fieldset>
            ))}
            <div className="submit-test">
              <p>
                После отправки изменить ответы нельзя. Пропущенные задания не
                засчитываются.
              </p>
              <Button disabled={busy} onClick={submit}>
                {busy
                  ? "Проверяем…"
                  : seconds === 0
                    ? "Завершить попытку"
                    : "Отправить на проверку"}
                <ArrowRight size={16} />
              </Button>
            </div>
          </section>
          <aside className="panel test-guide">
            <ShieldCheck size={25} />
            <h3>Равные условия</h3>
            <p>
              Одинаковые темы и правила оценки для каждого варианта. Параметры и
              порядок ответов меняются.
            </p>
            <ul>
              <li>20 минут на весь тест</li>
              <li>6 заданий, одинаковый вес</li>
              <li>Для прохождения — от 70%</li>
              <li>Ответы проверяет сервер</li>
            </ul>
            <small>
              Ответы сохраняются в этой вкладке. Время продолжает идти после
              перезагрузки.
            </small>
          </aside>
        </div>
      ) : (
        <>
          {result && (
            <div className="result-panel">
              <div className="result-score">
                <strong>{result.attempt.score}</strong>
                <span>из 100</span>
              </div>
              <div>
                <h2>
                  {result.attempt.passed
                    ? "Уровень подтверждён"
                    : "Есть пространство для роста"}
                </h2>
                <p>{result.message}</p>
                <div className="tags">
                  {Object.entries(result.topics).map(([topic, ok]) => (
                    <span className={ok ? "tag verified" : "tag"} key={topic}>
                      {ok ? "✓" : "·"} {topic}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          )}
          <div className="assessment-setup">
            <div className="panel">
              <h2>Выберите свою категорию</h2>
              <p className="muted">
                Указывайте уровень, который хотите подтвердить.
              </p>
              <div className="form-grid">
                <Field label="Специализация">
                  <select
                    value={spec}
                    onChange={(e) => setSpec(e.target.value)}
                  >
                    {Object.entries(specs).map(([k, v]) => (
                      <option key={k} value={k}>
                        {v}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="Предполагаемый грейд">
                  <select
                    value={grade}
                    onChange={(e) => setGrade(e.target.value)}
                  >
                    {Object.entries(grades).map(([k, v]) => (
                      <option key={k} value={k}>
                        {v}
                      </option>
                    ))}
                  </select>
                </Field>
              </div>
              <div className="test-facts">
                <span>
                  <Clock3 size={17} />
                  20 минут
                </span>
                <span>
                  <CheckCircle2 size={17} />6 заданий
                </span>
                <span>Проходной балл — 70%</span>
              </div>
              {locked ? (
                <Success>
                  Новая категория доступна с{" "}
                  {date(p.data!.next_grade_change_at!)}
                </Success>
              ) : (
                <Button onClick={start} disabled={busy}>
                  {busy ? "Готовим вариант…" : "Начать тестирование"}
                  <ArrowRight size={16} />
                </Button>
              )}
            </div>
            <div className="setup-note">
              <span className="eyebrow">Вы управляете своим уровнем</span>
              <h3>
                Результат — ориентир
                <br />
                для следующего шага.
              </h3>
              <p>
                Неудачная попытка не понижает подтверждённый грейд. Можно
                попробовать другой уровень, а этот повторить через сутки.
              </p>
              <p>
                После успешного теста новая смена категории доступна через 90
                дней.
              </p>
            </div>
          </div>
          <div className="section-heading">
            <h2>История тестирования</h2>
            <span>
              {q.data?.filter((a) => a.submitted_at).length || 0} попыток
            </span>
          </div>
          {q.data?.some((a) => a.submitted_at) ? (
            <div className="history">
              {q.data
                .filter((a) => a.submitted_at)
                .map((a) => (
                  <div className="history-row" key={a.id}>
                    <div>
                      <strong>{specs[a.specialization]}</strong>
                      <span>
                        {grades[a.grade]} · {date(a.started_at)}
                      </span>
                    </div>
                    <strong>{a.score}/100</strong>
                    <span className={"tag " + (a.passed ? "verified" : "")}>
                      {a.passed ? "Подтверждено" : "Не подтверждено"}
                    </span>
                  </div>
                ))}
            </div>
          ) : (
            <Empty title="Здесь появятся ваши результаты">
              Начните первый тест — он поможет работодателю увидеть вашу
              квалификацию.
            </Empty>
          )}
        </>
      )}
    </>
  );
}
