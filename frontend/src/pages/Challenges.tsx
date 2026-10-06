import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ArrowRight, CheckCircle2, Clock3, Plus, Send } from "lucide-react";
import {
  api,
  type Challenge,
  type ChallengeFeed,
  type ChallengeSubmission,
  date,
  grades,
  queryClient,
  specs,
} from "../api";
import { useUser } from "../App";
import { Button } from "../components/ui/button";
import { Select } from "../components/ui/select";
import {
  Empty,
  ErrorBox,
  Field,
  Loading,
  PageTitle,
  Success,
} from "../components/shared";

const statusLabels = {
  submitted: "На проверке",
  accepted: "Решение подтверждено",
  needs_work: "Есть рекомендации",
};

export function Challenges() {
  const me = useUser();
  const employer = me.data?.role === "employer";
  const q = useQuery({
    queryKey: ["challenges"],
    queryFn: () => api<ChallengeFeed>("/challenges"),
    refetchInterval: 15000,
  });
  const [creating, setCreating] = useState(false);
  const [busy, setBusy] = useState(false);
  const [filter, setFilter] = useState("all");
  const [error, setError] = useState<unknown>();
  const [message, setMessage] = useState("");
  const next = q.data?.next_submission_at;
  const locked = !!next && new Date(next).getTime() > Date.now();
  const list = q.data?.challenges.filter(
    (c) =>
      filter === "all" ||
      (filter === "active"
        ? new Date(c.closes_at).getTime() > Date.now() &&
          (employer || !c.submission)
        : employer
          ? new Date(c.closes_at).getTime() <= Date.now()
          : !!c.submission),
  );
  async function create(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    setMessage("");
    const data = Object.fromEntries(new FormData(e.currentTarget));
    try {
      await api("/challenges", "POST", {
        ...data,
        estimated_minutes: Number(data.estimated_minutes),
      });
      await queryClient.invalidateQueries({ queryKey: ["challenges"] });
      setCreating(false);
      setFilter("all");
      setMessage(
        "Задание опубликовано. Его увидят кандидаты этой категории в течение 7 дней.",
      );
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <PageTitle
        eyebrow={employer ? "Свежий сигнал о навыках" : "Практика раз в неделю"}
        title={employer ? "Задачи вашей команды" : "Короткие задания"}
        action={
          employer && (
            <Button
              onClick={() => setCreating(!creating)}
              aria-expanded={creating}
            >
              <Plus size={16} />
              {creating ? "Закрыть форму" : "Предложить задачу"}
            </Button>
          )
        }
      >
        {employer
          ? "Предложите небольшую задачу своей категории и дайте обратную связь по решению."
          : "Решите небольшую задачу от команды работодателя. Покажите, как вы мыслите, и получите обратную связь."}
      </PageTitle>
      {!employer && (
        <div className="practice-rule">
          <Clock3 size={18} />
          <p>
            Одна задача в 7 дней · без изменения грейда
            <span>
              После положительной проверки в профиле появится дата подтверждённой
              практики. Контакты открываются только через принятие приглашения.
            </span>
          </p>
        </div>
      )}
      <ErrorBox error={q.error || error} />
      {message && <Success>{message}</Success>}
      {creating && (
        <form className="panel form-stack challenge-create" onSubmit={create}>
          <div className="section-heading">
            <h2>Короткая задача</h2>
            <span>Условие и ожидаемый результат</span>
          </div>
          <Field label="Название">
            <input
              name="title"
              required
              minLength={4}
              maxLength={150}
              placeholder="Например: повторная доставка событий"
            />
          </Field>
          <div className="form-grid">
            <Field label="Специализация">
              <Select
                label="Специализация"
                name="specialization"
                options={specs}
              />
            </Field>
            <Field label="Подтверждённый уровень">
              <Select
                label="Подтверждённый уровень"
                name="grade"
                defaultValue="middle"
                options={grades}
              />
            </Field>
          </div>
          <Field
            label="Условие задачи"
            hint="Опишите небольшой рабочий случай и критерии хорошего ответа. Не запрашивайте реальные данные и выполнение полноценного проекта."
          >
            <textarea
              name="body"
              rows={6}
              minLength={40}
              maxLength={6000}
              required
              placeholder="Что произошло, какие ограничения есть и что нужно предложить?"
            />
          </Field>
          <Field label="Ожидаемое время, минут">
            <input
              name="estimated_minutes"
              type="number"
              min={5}
              max={30}
              defaultValue={10}
              required
            />
          </Field>
          <div className="actions">
            <Button disabled={busy}>
              {busy ? "Публикуем…" : "Опубликовать на неделю"}
              <ArrowRight size={16} />
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={() => setCreating(false)}
            >
              Отмена
            </Button>
          </div>
        </form>
      )}
      {!employer && locked && (
        <div className="notice">
          <Clock3 size={18} />
          <span>
            Вы уже отправили решение. Следующее можно отправить{" "}
            {new Date(next!).toLocaleString("ru-RU", {
              day: "numeric",
              month: "long",
              hour: "2-digit",
              minute: "2-digit",
            })}
            . Обратная связь появится в истории.
          </span>
        </div>
      )}
      <div className="tabs" role="group" aria-label="Фильтр заданий">
        {[
          ["all", "Все"],
          ["active", employer ? "Открытые" : "Доступные"],
          ["history", employer ? "Завершённые" : "Мои решения"],
        ].map(([key, label]) => (
          <button
            key={key}
            className={filter === key ? "active" : ""}
            aria-pressed={filter === key}
            onClick={() => setFilter(key)}
          >
            {label}
          </button>
        ))}
      </div>
      {q.isPending ? (
        <Loading />
      ) : q.error && !q.data ? (
        <Button variant="secondary" onClick={() => void q.refetch()}>
          Повторить загрузку
        </Button>
      ) : list?.length ? (
        <div className="challenge-list">
          {list.map((c) => (
            <ChallengeItem
              key={c.id}
              challenge={c}
              employer={employer}
              userID={me.data!.id}
              locked={locked}
            />
          ))}
        </div>
      ) : (
        <Empty
          title={
            filter === "history"
              ? "Здесь будет история решений"
              : employer
                ? "Предложите первую задачу"
                : "Новые задачи ещё впереди"
          }
          to={!employer ? "/assessments" : undefined}
          link="Проверить свою категорию"
        >
          {filter === "history"
            ? "Решения и комментарии сохраняются после завершения задания."
            : employer
              ? "Выберите специализацию и уровень. Задание автоматически появится у кандидатов этой категории."
              : "Задачи подбираются по подтверждённой специализации и уровню. Проверьте свою категорию или загляните позже."}
        </Empty>
      )}
    </>
  );
}

function ChallengeItem({
  challenge: c,
  employer,
  userID,
  locked,
}: {
  challenge: Challenge;
  employer: boolean;
  userID: string;
  locked: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const draftKey = "challenge:" + userID + ":" + c.id;
  const [answer, setAnswer] = useState(() => {
    try {
      return sessionStorage.getItem(draftKey) || "";
    } catch {
      return "";
    }
  });
  const expired = new Date(c.closes_at).getTime() <= Date.now();
  const submissions = useQuery({
    queryKey: ["challenge-submissions", c.id],
    queryFn: () =>
      api<ChallengeSubmission[]>("/challenges/" + c.id + "/submissions"),
    enabled: employer && open,
    refetchInterval: open ? 15000 : false,
  });
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    try {
      await api("/challenges/" + c.id + "/submissions", "POST", { answer });
      try {
        sessionStorage.removeItem(draftKey);
      } catch {
        /* Draft storage is optional. */
      }
      await queryClient.invalidateQueries({ queryKey: ["challenges"] });
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  return (
    <article className="challenge-item">
      <div className="challenge-meta">
        <span className="eyebrow">{c.company_name}</span>
        <span
          className={
            "tag " +
            (c.submission?.status === "accepted" ? "status-accepted" : "")
          }
        >
          {c.submission
            ? statusLabels[c.submission.status]
            : expired
              ? "Приём завершён"
              : "Открыто"}
        </span>
      </div>
      <h2>{c.title}</h2>
      <div className="challenge-facts">
        <span>
          {specs[c.specialization]} · {grades[c.grade]}
        </span>
        <span>
          <Clock3 size={14} />
          {c.estimated_minutes} минут
        </span>
        <span>До {date(c.closes_at)}</span>
      </div>
      <button
        className="text-link"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
      >
        {open
          ? "Свернуть"
          : employer
            ? `Условие и решения (${c.submission_count})`
            : c.submission
              ? "Моё решение и обратная связь"
              : "Открыть задачу"}
        <ArrowRight size={15} />
      </button>
      {open && (
        <div className="challenge-content">
          <p className="preserve-lines">{c.body}</p>
          <ErrorBox error={error || submissions.error} />
          {employer ? (
            submissions.isPending ? (
              <Loading />
            ) : submissions.error && !submissions.data ? (
              <Button
                variant="secondary"
                onClick={() => void submissions.refetch()}
              >
                Повторить загрузку решений
              </Button>
            ) : submissions.data?.length ? (
              <div className="submission-list">
                {submissions.data.map((s) => (
                  <Review key={s.id} submission={s} challengeID={c.id} />
                ))}
              </div>
            ) : (
              <p className="muted">
                Решений пока нет. Кандидаты увидят задание в своём кабинете.
              </p>
            )
          ) : c.submission ? (
            <div className="own-solution">
              <h3>Ваше решение</h3>
              <p className="preserve-lines">{c.submission.answer}</p>
              <small className="muted">
                Отправлено {date(c.submission.submitted_at)}
              </small>
              {c.submission.feedback ? (
                <div className="feedback">
                  <h3>Обратная связь команды</h3>
                  <p className="preserve-lines">{c.submission.feedback}</p>
                  {c.submission.status === "accepted" && (
                    <p className="text-link">
                      <CheckCircle2 size={16} />
                      Работодатель подтвердил решение
                    </p>
                  )}
                </div>
              ) : (
                <p className="muted">Команда ещё не проверила решение.</p>
              )}
            </div>
          ) : (
            !expired && (
              <form className="form-stack" onSubmit={submit}>
                <Field
                  label="Ваше решение или подход"
                  hint="40–6000 символов. Черновик сохраняется в этой вкладке."
                >
                  <textarea
                    value={answer}
                    rows={6}
                    minLength={40}
                    maxLength={6000}
                    required
                    disabled={locked || busy}
                    onChange={(e) => {
                      setAnswer(e.target.value);
                      try {
                        sessionStorage.setItem(draftKey, e.target.value);
                      } catch {
                        /* Draft storage is optional. */
                      }
                    }}
                    placeholder="Опишите шаги решения, ограничения и проверку результата."
                  />
                </Field>
                <Button disabled={locked || busy}>
                  {busy ? "Отправляем…" : "Отправить решение"}
                  <Send size={16} />
                </Button>
              </form>
            )
          )}
        </div>
      )}
    </article>
  );
}

function Review({
  submission: s,
  challengeID,
}: {
  submission: ChallengeSubmission;
  challengeID: string;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    const data = Object.fromEntries(new FormData(e.currentTarget));
    try {
      await api("/challenge-submissions/" + s.id, "PATCH", data);
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ["challenge-submissions", challengeID],
        }),
        queryClient.invalidateQueries({ queryKey: ["challenges"] }),
        queryClient.invalidateQueries({ queryKey: ["candidates"] }),
        queryClient.invalidateQueries({ queryKey: ["candidate"] }),
      ]);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="submission">
      <div className="challenge-meta">
        <h3>{s.candidate_name}</h3>
        <span className="tag">{statusLabels[s.status]}</span>
      </div>
      <p className="preserve-lines">{s.answer}</p>
      <small className="muted">Отправлено {date(s.submitted_at)}</small>
      <ErrorBox error={error} />
      {s.status === "submitted" ? (
        <form className="form-stack" onSubmit={submit}>
          <Field label="Результат проверки">
            <Select
              label="Результат проверки"
              name="status"
              options={{
                accepted: "Подход подходит",
                needs_work: "Есть что улучшить",
              }}
            />
          </Field>
          <Field
            label="Обратная связь"
            hint="Решение не изменяет грейд кандидата."
          >
            <textarea
              name="feedback"
              rows={3}
              minLength={10}
              maxLength={2000}
              required
              placeholder="Что получилось хорошо и что можно улучшить?"
            />
          </Field>
          <Button disabled={busy} variant="secondary">
            {busy ? "Сохраняем…" : "Отправить обратную связь"}
          </Button>
        </form>
      ) : (
        <div className="feedback">
          <h4>Ваш комментарий</h4>
          <p className="preserve-lines">{s.feedback}</p>
        </div>
      )}
      <Link className="text-link" to={"/candidates/" + s.candidate_id}>
        Профиль кандидата
        <ArrowRight size={15} />
      </Link>
    </section>
  );
}
