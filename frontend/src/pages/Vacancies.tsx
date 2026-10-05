import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Link,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import {
  ArrowLeft,
  ArrowRight,
  BriefcaseBusiness,
  MapPin,
  Plus,
  ShieldCheck,
} from "lucide-react";
import {
  api,
  type Vacancy,
  type Profile,
  applicationStatuses,
  date,
  grades,
  money,
  queryClient,
  specs,
  workFormats,
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

const vacancyStatuses: Record<string, string> = {
  draft: "Черновик",
  published: "Опубликована",
  closed: "Закрыта",
};
export function Vacancies() {
  const me = useUser();
  const employer = me.data?.role === "employer";
  const [params, setParams] = useSearchParams();
  const [tab, setTab] = useState("all");
  const q = useQuery({
    queryKey: ["vacancies", params.toString()],
    queryFn: () =>
      api<{ vacancies: Vacancy[]; total: number }>("/vacancies?" + params),
  });
  const update = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    value ? next.set(key, value) : next.delete(key);
    next.delete("page");
    setParams(next);
  };
  const rows =
    q.data?.vacancies.filter(
      (v) => !employer || tab === "all" || v.status === tab,
    ) || [];
  return (
    <>
      <PageTitle
        eyebrow={employer ? "Найм в вашу команду" : "Открытые возможности"}
        title={employer ? "Вакансии команды" : "Условия известны заранее."}
        action={
          employer && (
            <Button asChild>
              <Link to="/vacancies/new">
                <Plus size={18} />
                Создать вакансию
              </Link>
            </Button>
          )
        }
      >
        {employer
          ? "Обратный подбор остаётся главным. Публикация вакансии даёт кандидатам возможность проявить инициативу."
          : "Работодатели могут пригласить вас сами. Здесь вы также можете выбрать команду и отправить отклик."}
      </PageTitle>
      {employer ? (
        <div className="tabs" aria-label="Статус вакансий">
          {["all", "published", "draft", "closed"].map((t) => (
            <button
              key={t}
              className={tab === t ? "active" : ""}
              aria-pressed={tab === t}
              onClick={() => setTab(t)}
            >
              {t === "all" ? "Все" : vacancyStatuses[t]}
            </button>
          ))}
        </div>
      ) : (
        <div className="filters">
          <label>
            <span>Направление</span>
            <Select
              label="Направление"
              options={specs}
              emptyLabel="Все направления"
              value={params.get("specialization") || ""}
              onValueChange={(value) => update("specialization", value)}
            />
          </label>
          <label>
            <span>Уровень</span>
            <Select
              label="Уровень"
              options={grades}
              emptyLabel="Все уровни"
              value={params.get("grade") || ""}
              onValueChange={(value) => update("grade", value)}
            />
          </label>
          <label>
            <span>Формат</span>
            <Select
              label="Формат"
              options={workFormats}
              emptyLabel="Любой формат"
              value={params.get("work_format") || ""}
              onValueChange={(value) => update("work_format", value)}
            />
          </label>
        </div>
      )}
      <ErrorBox error={q.error} />
      {q.isPending ? (
        <Loading />
      ) : q.error && !q.data ? (
        <Button onClick={() => void q.refetch()}>Повторить загрузку</Button>
      ) : rows.length ? (
        <div className="vacancy-list">
          {rows.map((v) => (
            <article className="vacancy-row" key={v.id}>
              <div className="vacancy-company">
                <BriefcaseBusiness size={18} />
                <span>{v.company_name}</span>
                <small>{date(v.created_at)}</small>
              </div>
              <div className="vacancy-row-heading">
                <div>
                  <Link to={`/vacancies/${v.id}`}>
                    <h2>{v.title}</h2>
                  </Link>
                  <p>
                    {specs[v.specialization]} · {grades[v.grade]}
                  </p>
                </div>
                <div className="vacancy-salary">
                  <strong>
                    {money(v.salary_min)}–{money(v.salary_max)} ₽
                  </strong>
                  <small>в месяц, до вычета налогов</small>
                </div>
              </div>
              <p className="vacancy-excerpt">
                {v.description.length > 180
                  ? v.description.slice(0, 180) + "…"
                  : v.description}
              </p>
              <div className="vacancy-row-footer">
                <div className="tags">
                  <span className="tag">
                    <MapPin size={13} />
                    {workFormats[v.work_format]}
                    {v.city ? ` · ${v.city}` : ""}
                  </span>
                  {v.skills.map((s) => (
                    <span className="skill" key={s}>
                      {s}
                    </span>
                  ))}
                </div>
                <Link className="text-link" to={`/vacancies/${v.id}`}>
                  {employer
                    ? `${vacancyStatuses[v.status]} · ${v.application_count} откликов`
                    : v.my_application
                      ? applicationStatuses[v.my_application.status]
                      : "Посмотреть условия"}
                  <ArrowRight size={16} />
                </Link>
              </div>
            </article>
          ))}
        </div>
      ) : (
        <Empty
          title={
            employer
              ? "Создайте первую вакансию"
              : "Пока нет вакансий по этим условиям"
          }
          to={employer ? "/vacancies/new" : undefined}
          link="Создать вакансию"
        >
          {employer
            ? "Расскажите о задачах, укажите стек и зарплату. Сначала сохранится черновик."
            : "Попробуйте расширить фильтры. Ваш профиль остаётся доступен для приглашений, если вы разрешили публикацию."}
        </Empty>
      )}
      {q.data && q.data.total > 20 && (
        <div className="pagination">
          <Button
            variant="secondary"
            disabled={Number(params.get("page") || 1) <= 1}
            onClick={() => {
              const n = new URLSearchParams(params);
              n.set("page", String(Number(params.get("page") || 1) - 1));
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
              n.set("page", String(Number(params.get("page") || 1) + 1));
              setParams(n);
            }}
          >
            Далее
          </Button>
        </div>
      )}
    </>
  );
}

export function VacancyEditor({ vacancy }: { vacancy?: Vacancy }) {
  const navigate = useNavigate();
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const [format, setFormat] = useState(vacancy?.work_format || "remote");
  async function save(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    const f = Object.fromEntries(new FormData(e.currentTarget));
    try {
      const payload = {
        ...f,
        skills: String(f.skills)
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean),
        salary_min: Number(f.salary_min),
        salary_max: Number(f.salary_max),
      };
      const r = await api<{ id?: string }>(
        vacancy ? `/vacancies/${vacancy.id}` : "/vacancies",
        vacancy ? "PUT" : "POST",
        payload,
      );
      await queryClient.invalidateQueries({ queryKey: ["vacancies"] });
      await queryClient.invalidateQueries({ queryKey: ["vacancy"] });
      navigate(`/vacancies/${vacancy?.id || r.id}`, { replace: true });
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <Link
        className="text-link back"
        to={vacancy ? `/vacancies/${vacancy.id}` : "/vacancies"}
      >
        <ArrowLeft size={16} />К вакансиям
      </Link>
      <PageTitle
        eyebrow="Работодатель"
        title={
          vacancy ? "Редактировать вакансию" : "Новая возможность в команде"
        }
      >
        Укажите реальные задачи и условия. Контакты откликнувшегося кандидата
        будут доступны вашей компании.
      </PageTitle>
      <ErrorBox error={error} />
      <form className="vacancy-editor" onSubmit={save}>
        <div className="form-stack">
          <Field label="Название вакансии">
            <input
              name="title"
              defaultValue={vacancy?.title}
              minLength={4}
              maxLength={160}
              required
              placeholder="Go-разработчик в команду платежей"
            />
          </Field>
          <Field
            label="Задачи, команда и ожидания"
            hint="От 40 символов. Опишите, над чем предстоит работать, требования и условия."
          >
            <textarea
              name="description"
              defaultValue={vacancy?.description}
              rows={9}
              minLength={40}
              maxLength={6000}
              required
            />
          </Field>
          <Field label="Стек через запятую">
            <input
              name="skills"
              defaultValue={vacancy?.skills.join(", ")}
              placeholder="Go, PostgreSQL, Redis"
              maxLength={500}
            />
          </Field>
        </div>
        <aside className="form-stack">
          <div className="salary-fields">
            <Field label="Направление">
              <Select
                label="Направление"
                options={specs}
                name="specialization"
                defaultValue={vacancy?.specialization || "backend"}
              />
            </Field>
            <Field label="Уровень">
              <Select
                label="Уровень"
                options={grades}
                name="grade"
                defaultValue={vacancy?.grade || "middle"}
              />
            </Field>
          </div>
          <Field label="Формат работы">
            <Select
              label="Формат работы"
              options={workFormats}
              name="work_format"
              value={format}
              onValueChange={(value) =>
                setFormat(value as Vacancy["work_format"])
              }
            />
          </Field>
          <Field
            label="Город"
            hint={
              format === "remote"
                ? "Необязательно для удалённой работы"
                : "Город работы команды"
            }
          >
            <input
              name="city"
              defaultValue={vacancy?.city}
              required={format !== "remote"}
              maxLength={100}
            />
          </Field>
          <div className="salary-fields">
            <Field label="Зарплата от, ₽">
              <input
                name="salary_min"
                type="number"
                min={1}
                max={10000000}
                defaultValue={vacancy?.salary_min}
                required
              />
            </Field>
            <Field label="До, ₽">
              <input
                name="salary_max"
                type="number"
                min={1}
                max={10000000}
                defaultValue={vacancy?.salary_max}
                required
              />
            </Field>
          </div>
          <small>
            В месяц, до вычета налогов. Зарплатная вилка обязательна и видна
            кандидату до отклика.
          </small>
          <Button disabled={busy}>
            {busy
              ? "Сохраняем…"
              : vacancy
                ? "Сохранить изменения"
                : "Сохранить черновик"}
            <ArrowRight size={16} />
          </Button>
          {!vacancy && (
            <small>Публикация — отдельный шаг после проверки черновика.</small>
          )}
        </aside>
      </form>
    </>
  );
}

export function VacancyDetail({ edit = false }: { edit?: boolean }) {
  const { id } = useParams();
  const me = useUser();
  const employer = me.data?.role === "employer";
  const q = useQuery({
    queryKey: ["vacancy", id],
    queryFn: () => api<Vacancy>(`/vacancies/${id}`),
  });
  const profile = useQuery({
    queryKey: ["profile"],
    queryFn: () => api<Profile>("/profile"),
    enabled: !employer,
  });
  const [error, setError] = useState<unknown>();
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const mutate = async (path: string, method: string, body: unknown) => {
    setBusy(true);
    setError(undefined);
    setMessage("");
    try {
      const r = await api<{ message: string }>(path, method, body);
      setMessage(r.message);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["vacancy"] }),
        queryClient.invalidateQueries({ queryKey: ["vacancies"] }),
        queryClient.invalidateQueries({ queryKey: ["applications"] }),
      ]);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  if (q.isPending) return <Loading />;
  if (!q.data)
    return (
      <>
        <ErrorBox error={q.error} />
        <Button onClick={() => void q.refetch()}>Повторить загрузку</Button>
      </>
    );
  const v = q.data;
  if (edit)
    return employer ? (
      <VacancyEditor key={v.id} vacancy={v} />
    ) : (
      <Empty
        title="Редактирование недоступно"
        to="/vacancies"
        link="К вакансиям"
      >
        Вакансию редактирует компания, которая её разместила.
      </Empty>
    );
  const application = v.my_application;
  return (
    <>
      <Link className="text-link back" to="/vacancies">
        <ArrowLeft size={16} />К вакансиям
      </Link>
      <PageTitle
        eyebrow={v.company_name}
        title={v.title}
        action={
          employer && (
            <Button asChild variant="secondary">
              <Link to={`/vacancies/${v.id}/edit`}>Редактировать</Link>
            </Button>
          )
        }
      >
        {specs[v.specialization]} · {grades[v.grade]} ·{" "}
        {workFormats[v.work_format]}
        {v.city ? ` · ${v.city}` : ""}
      </PageTitle>
      <div className="vacancy-detail-layout">
        <section>
          <div className="vacancy-pay">
            <span>Зарплатная вилка</span>
            <strong>
              {money(v.salary_min)}–{money(v.salary_max)} ₽
            </strong>
            <small>В месяц, до вычета налогов</small>
          </div>
          <div className="section-heading">
            <h2>Над чем предстоит работать</h2>
          </div>
          <p className="long-copy">{v.description}</p>
          <div className="tags">
            {v.skills.map((s) => (
              <span className="skill" key={s}>
                {s}
              </span>
            ))}
          </div>
          <div className="section-heading">
            <h2>О компании</h2>
          </div>
          <p className="long-copy">
            {v.company_description || "Компания пока не добавила описание."}
          </p>
          {/^https?:\/\//i.test(v.company_website) && (
            <a
              className="text-link"
              href={v.company_website}
              target="_blank"
              rel="noreferrer"
            >
              Сайт компании
              <ArrowRight size={16} />
            </a>
          )}
        </section>
        <aside className="vacancy-action">
          <ErrorBox error={error || q.error} />
          {message && <Success>{message}</Success>}
          {employer ? (
            <>
              <span className={`status ${v.status}`}>
                {vacancyStatuses[v.status]}
              </span>
              <h2>{v.application_count} откликов</h2>
              <Button asChild variant="secondary">
                <Link to={`/applications?vacancy_id=${v.id}`}>
                  Рассмотреть отклики
                  <ArrowRight size={16} />
                </Link>
              </Button>
              {v.status !== "published" && (
                <Button
                  disabled={busy}
                  onClick={() =>
                    void mutate(`/vacancies/${v.id}`, "PATCH", {
                      status: "published",
                    })
                  }
                >
                  {v.status === "draft"
                    ? "Опубликовать вакансию"
                    : "Открыть снова"}
                </Button>
              )}
              {v.status === "published" && (
                <Button
                  disabled={busy}
                  variant="secondary"
                  onClick={() =>
                    void mutate(`/vacancies/${v.id}`, "PATCH", {
                      status: "closed",
                    })
                  }
                >
                  Закрыть приём откликов
                </Button>
              )}
              <Link
                className="text-link"
                to={`/candidates?${new URLSearchParams({ specialization: v.specialization, grade: v.grade, need: `${v.grade} ${v.specialization} ${v.skills.join(" ")}` })}`}
              >
                Подобрать кандидатов по этой вакансии
                <ArrowRight size={16} />
              </Link>
            </>
          ) : application && application.status !== "withdrawn" ? (
            <>
              <ShieldCheck size={24} />
              <h2>
                Отклик {applicationStatuses[application.status].toLowerCase()}
              </h2>
              <p className="muted">
                Вы передали профиль и почту компании {v.company_name}. В разделе
                «Мои отклики» можно следить за решением или отозвать доступ.
              </p>
              <Button asChild variant="secondary">
                <Link to="/applications">
                  Мои отклики
                  <ArrowRight size={16} />
                </Link>
              </Button>
            </>
          ) : v.status !== "published" ? (
            <>
              <h2>Приём откликов завершён</h2>
              <p>
                Выберите другую возможность. Работодатель также может пригласить
                вас напрямую.
              </p>
            </>
          ) : !profile.data?.grade ? (
            <>
              <h2>Сначала подтвердите уровень</h2>
              <p>
                Категория определяется тестом. После тестирования вы сможете
                отправить отклик.
              </p>
              <ErrorBox error={profile.error} />
              <Button asChild>
                <Link to="/assessments">
                  Подтвердить навыки
                  <ArrowRight size={16} />
                </Link>
              </Button>
            </>
          ) : (
            <form
              className="form-stack"
              onSubmit={(e) => {
                e.preventDefault();
                const f = new FormData(e.currentTarget);
                void mutate(`/vacancies/${v.id}/applications`, "POST", {
                  cover_letter: f.get("cover_letter"),
                  consent: f.get("consent") === "on",
                });
              }}
            >
              <h2>Познакомиться с командой</h2>
              {(profile.data.specialization !== v.specialization ||
                profile.data.grade !== v.grade) && (
                <p className="muted">
                  Ваша категория отличается от указанной в вакансии. Компания
                  увидит ваш подтверждённый уровень.
                </p>
              )}
              <Field
                label="Несколько слов о себе"
                hint="Необязательно. Расскажите, чем вам интересны задачи команды."
              >
                <textarea name="cover_letter" rows={5} maxLength={3000} />
              </Field>
              <label className="check">
                <input type="checkbox" name="consent" required />
                <span>
                  Разрешаю передать мой профиль и электронную почту компании{" "}
                  {v.company_name} для рассмотрения отклика.
                </span>
              </label>
              <Button disabled={busy}>
                {busy ? "Отправляем…" : "Отправить отклик"}
                <ArrowRight size={16} />
              </Button>
              <small>
                Отклик можно отозвать. Другие компании не получат доступ к вашим
                контактам.
              </small>
            </form>
          )}
        </aside>
      </div>
    </>
  );
}
