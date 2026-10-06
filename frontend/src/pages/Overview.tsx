import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import {
  ArrowRight,
  Bookmark,
  BriefcaseBusiness,
  Check,
  ChevronRight,
  Inbox,
  Search,
  ShieldCheck,
} from "lucide-react";
import {
  api,
  type Profile,
  type Invitation,
  date,
  grades,
  money,
  specs,
} from "../api";
import { useUser } from "../App";
import { Button } from "../components/ui/button";
import { Empty, ErrorBox, Loading } from "../components/shared";
import { EvidencePassport } from "../components/Evidence";

interface OverviewData {
  invitations: Record<string, number>;
  applications: Record<string, number>;
  categories: { specialization: string; grade: string; count: number }[];
  candidate_count: number;
  published_vacancies: number;
  saved_candidates: number;
}
export function Overview() {
  const me = useUser();
  const employer = me.data?.role === "employer";
  const navigate = useNavigate();
  const [need, setNeed] = useState("");
  const q = useQuery({
    queryKey: ["overview"],
    queryFn: () => api<OverviewData>("/overview"),
    refetchInterval: 15000,
  });
  const profile = useQuery({
    queryKey: ["profile"],
    queryFn: () => api<Profile>("/profile"),
    enabled: !employer,
  });
  const company = useQuery({
    queryKey: ["company"],
    queryFn: () => api<{ name: string; description: string }>("/company"),
    enabled: employer,
  });
  const invitations = useQuery({
    queryKey: ["invitations"],
    queryFn: () => api<Invitation[]>("/invitations"),
  });
  if (q.isPending) return <Loading />;
  if (!q.data)
    return (
      <>
        <ErrorBox error={q.error} />
        <Button onClick={() => void q.refetch()}>Повторить загрузку</Button>
      </>
    );
  const data = q.data;
  const p = profile.data;
  const nextStep =
    !p?.headline || !p?.skills.length ? 0 : !p.grade ? 1 : !p.published ? 2 : 3;
  const inviteTotal = Object.values(data.invitations).reduce(
    (a, b) => a + b,
    0,
  );
  return (
    <div className="overview-page">
      <header className="overview-heading">
        <div>
          <h1>
            {employer
              ? "Соберите сильную команду."
              : `Ваши навыки. Новые возможности.`}
          </h1>
          <p>
            {employer
              ? "Подтверждённая квалификация → точный выбор → предложение с зарплатой."
              : "Подтвердите уровень один раз. Получайте предложения от команд, которым нужен ваш опыт."}
          </p>
        </div>
      </header>
      <ErrorBox error={q.error || profile.error || company.error} />
      {employer ? (
        <>
          <div className="hiring-workbench">
            <section className="hiring-start">
              <span className="section-index">
                01 / Найти своего специалиста
              </span>
              <h2>
                Какая задача стоит
                <br />
                перед командой?
              </h2>
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  navigate(`/candidates?${new URLSearchParams({ need })}`);
                }}
              >
                <label htmlFor="overview-need">
                  Направление, уровень и технологии
                </label>
                <div>
                  <input
                    id="overview-need"
                    value={need}
                    onChange={(e) => setNeed(e.target.value)}
                    placeholder="Middle backend · Go, PostgreSQL, Redis"
                    maxLength={2000}
                    required
                  />
                  <Button type="submit">
                    <Search size={18} />
                    Подобрать
                  </Button>
                </div>
              </form>
              <div className="need-examples">
                <span>Быстрый старт</span>
                {[
                  {
                    label: "Go-разработчик",
                    text: "middle backend Go PostgreSQL Redis",
                  },
                  {
                    label: "React-разработчик",
                    text: "middle frontend React TypeScript",
                  },
                  {
                    label: "Аналитик данных",
                    text: "middle аналитик SQL Python",
                  },
                ].map((x) => (
                  <button
                    key={x.label}
                    onClick={() =>
                      navigate(
                        `/candidates?${new URLSearchParams({ need: x.text })}`,
                      )
                    }
                  >
                    {x.label}
                    <ChevronRight size={14} />
                  </button>
                ))}
              </div>
              <small>
                Ищем по категории из теста. Совпадение стека и достижения
                помогают выбрать внутри неё.
              </small>
            </section>
            <aside className="hiring-pulse">
              <p className="eyebrow">Ваша работа с кандидатами</p>
              <h2>
                От интереса
                <br />к знакомству.
              </h2>
              <ol className="hiring-stages">
                {[
                  {
                    label: "Сохранено для сравнения",
                    count: data.saved_candidates,
                    to: "/shortlist",
                    icon: Bookmark,
                  },
                  {
                    label: "Приглашения ждут решения",
                    count:
                      (data.invitations.sent || 0) +
                      (data.invitations.viewed || 0),
                    to: "/invitations",
                    icon: Inbox,
                  },
                  {
                    label: "Готовы начать общение",
                    count: data.invitations.accepted || 0,
                    to: "/invitations",
                    icon: Check,
                  },
                ].map(({ label, count, to, icon: Icon }) => (
                  <li key={label}>
                    <Link to={to}>
                      <Icon size={17} />
                      <span>{label}</span>
                      <strong>{count}</strong>
                      <ChevronRight size={15} />
                    </Link>
                  </li>
                ))}
              </ol>
              <small>
                По вашим приглашениям. Принятые предложения открывают контакты.
              </small>
            </aside>
          </div>
          <section className="overview-section">
            <div className="section-heading">
              <div>
                <p className="eyebrow">02 / Выбрать по подтверждениям</p>
                <h2>Кого можно найти сейчас</h2>
              </div>
              <span className="muted">
                {data.candidate_count} опубликованных профилей
              </span>
            </div>
            <div className="category-browser">
              {Object.entries(specs).map(([s, label]) => (
                <div key={s}>
                  <h3>{label}</h3>
                  {Object.entries(grades).map(([g, title]) => {
                    const count =
                      data.categories.find(
                        (c) => c.specialization === s && c.grade === g,
                      )?.count || 0;
                    return (
                      <Link
                        key={g}
                        to={`/candidates?specialization=${s}&grade=${g}`}
                      >
                        <span>{title}</span>
                        <strong>{count}</strong>
                        <ArrowRight size={15} />
                      </Link>
                    );
                  })}
                </div>
              ))}
            </div>
          </section>
          <div className="overview-action-line">
            <BriefcaseBusiness size={20} />
            <div>
              <strong>
                {data.published_vacancies} опубликованных вакансий ·{" "}
                {data.applications.new || 0} новых откликов
              </strong>
              <p>Отклики дополняют основной сценарий приглашений.</p>
            </div>
            <Link className="text-link" to="/applications">
              Рассмотреть отклики
              <ArrowRight size={16} />
            </Link>
          </div>
        </>
      ) : (
        <>
          <section className="candidate-route">
            <div>
              <p className="section-index">Ваш следующий шаг</p>
              <h2>
                {nextStep === 0
                  ? "Покажите, что вы умеете."
                  : nextStep === 1
                    ? "Подтвердите свой уровень."
                    : nextStep === 2
                      ? "Станьте видимым командам."
                      : "Вы в банке кандидатов."}
              </h2>
              <p>
                {nextStep === 0
                  ? "Добавьте стек и коротко расскажите о своём опыте. Это поможет работодателю понять ваш профиль."
                  : nextStep === 1
                    ? "Выберите специализацию и предполагаемый грейд. Персональный вариант теста определит вашу категорию."
                    : nextStep === 2
                      ? "Квалификация уже подтверждена. Разрешите публикацию, чтобы работодатели могли найти и пригласить вас."
                      : "Работодатели видят вашу подтверждённую категорию. Контакты открываются только после вашего решения."}
              </p>
              <Button asChild>
                <Link
                  to={
                    nextStep === 1
                      ? "/assessments"
                      : nextStep === 3
                        ? "/invitations"
                        : "/profile"
                  }
                >
                  {nextStep === 1
                    ? "Пройти тест"
                    : nextStep === 3
                      ? "Посмотреть предложения"
                      : "Открыть профиль"}
                  <ArrowRight size={16} />
                </Link>
              </Button>
            </div>
            <ol className="onboarding-route">
              {[
                {
                  title: "Профиль и стек",
                  done: !!p?.headline && !!p?.skills.length,
                  to: "/profile",
                },
                {
                  title: "Подтверждённая категория",
                  done: !!p?.grade,
                  to: "/assessments",
                },
                {
                  title: "Видимость работодателям",
                  done: !!p?.published && !!p?.grade,
                  to: "/profile",
                },
              ].map((s, i) => (
                <li key={s.title} className={s.done ? "done" : ""}>
                  <Link to={s.to}>
                    <span>
                      {s.done ? (
                        <Check size={17} />
                      ) : (
                        String(i + 1).padStart(2, "0")
                      )}
                    </span>
                    <div>
                      <strong>{s.title}</strong>
                      <small>{s.done ? "Готово" : "Следующий шаг"}</small>
                    </div>
                    <ChevronRight size={16} />
                  </Link>
                </li>
              ))}
            </ol>
          </section>
          {p && (
            <div className="candidate-overview-meta">
              <span>
                <ShieldCheck size={18} />
                {p.grade
                  ? `${specs[p.specialization]} · ${grades[p.grade]}`
                  : "Категория ещё не определена"}
              </span>
              <span>
                {p.published && p.grade
                  ? "Профиль виден работодателям"
                  : "Профиль скрыт из поиска"}
              </span>
              <Link className="text-link" to="/challenges">
                Поддержать навыки практикой
                <ArrowRight size={16} />
              </Link>
            </div>
          )}
        </>
      )}
      <section className="overview-section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              {employer
                ? "03 / Начать разговор"
                : "Работодатели проявляют инициативу"}
            </p>
            <h2>
              {employer ? "Последние приглашения" : "Предложения для вас"}
            </h2>
          </div>
          <Link className="text-link" to="/invitations">
            Все {inviteTotal}
            <ArrowRight size={16} />
          </Link>
        </div>
        <ErrorBox error={invitations.error} />
        {invitations.isPending ? (
          <Loading />
        ) : invitations.data?.length ? (
          <div className="overview-invitations">
            {invitations.data.slice(0, 3).map((i) => (
              <Link to="/invitations" key={i.id}>
                <div>
                  <small>{employer ? i.candidate_name : i.company_name}</small>
                  <strong>{i.title}</strong>
                  <span>
                    {money(i.salary_min)}–{money(i.salary_max)} ₽{" "}
                    <small>в месяц</small>
                  </span>
                </div>
                <span className={`status ${i.status}`}>
                  {
                    {
                      sent: "Отправлено",
                      viewed: "Просмотрено",
                      accepted: "Принято",
                      declined: "Отклонено",
                    }[i.status]
                  }
                </span>
                <ArrowRight size={18} />
              </Link>
            ))}
          </div>
        ) : (
          <Empty
            title={
              employer
                ? "Начните знакомство с первым кандидатом"
                : "Первое предложение ещё впереди"
            }
            to={employer ? "/candidates" : "/vacancies"}
            link={employer ? "Подобрать кандидатов" : "Посмотреть вакансии"}
          >
            {employer
              ? "Укажите задачи и зарплату: кандидат увидит условия до начала общения."
              : "Проверьте публикацию профиля. Пока работодатель ищет вас, можно познакомиться с открытыми вакансиями."}
          </Empty>
        )}
      </section>
      {!employer && <EvidencePassport />}
    </div>
  );
}
