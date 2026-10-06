import { useEffect, type CSSProperties, type ReactNode } from "react";
import { useQueries } from "@tanstack/react-query";
import { Avatar } from "../components/Avatar";
import { Link, useSearchParams } from "react-router-dom";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Columns2,
  Minus,
  Plus,
  ShieldCheck,
  X,
} from "lucide-react";
import {
  api,
  type Candidate,
  type QualificationEvidence,
  date,
  grades,
  plural,
  specs,
  workFormats,
  employmentTypes,
  availabilityTypes,
  searchStatuses,
} from "../api";
import {
  professionalData,
  salaryExpectation,
} from "../components/ProfessionalProfile";
import {
  comparisonQuery,
  comparisonReturnTo,
  useComparison,
  validComparisonId,
} from "../components/Comparison";
import { SaveCandidate } from "../components/SaveCandidate";
import { ErrorBox, Loading, PageTitle } from "../components/shared";
import { Button } from "../components/ui/button";

type Criterion = {
  id: string;
  label: string;
  hint?: string;
  value: (
    candidate: Candidate,
    evidence?: QualificationEvidence,
  ) => string | number;
  render: (candidate: Candidate, evidence?: QualificationEvidence) => ReactNode;
};
type Group = { label: string; rows: Criterion[] };
const normalize = (skill: string) => skill.trim().toLocaleLowerCase("ru-RU");
function evidenceLabel(evidence?: QualificationEvidence) {
  if (!evidence) return "Подтверждение недоступно";
  return {
    assessment: "Пройден тест",
    demo: "Демопрофиль",
    imported: "Импортирована категория",
    unconfirmed: "Не подтверждено",
  }[evidence.source];
}
const number = (value: number, max = 100) => (
  <span className="comparison-number">
    {value}
    <small>/{max}</small>
  </span>
);

export function Compare() {
  const [params, setParams] = useSearchParams();
  const comparison = useComparison();
  const idString = [
    ...new Set((params.get("ids") || "").split(",").filter(validComparisonId)),
  ]
    .slice(0, 3)
    .join(",");
  const ids = idString ? idString.split(",") : [];
  const context = comparisonQuery(params);
  const returnTo = comparisonReturnTo(params.get("from"));
  const comparePath = `/compare?${params}`;
  const onlyDifferences = params.get("differences") === "true";
  const queries = useQueries({
    queries: ids.map((id) => ({
      queryKey: ["candidate", id, context],
      queryFn: () =>
        api<Candidate>(`/candidates/${encodeURIComponent(id)}?${context}`),
    })),
  });
  const evidenceQueries = useQueries({
    queries: ids.map((id, index) => ({
      queryKey: ["evidence", id],
      queryFn: () =>
        api<QualificationEvidence>(
          `/candidates/${encodeURIComponent(id)}/evidence`,
        ),
      enabled: !!queries[index]?.data,
    })),
  });
  const candidates = queries.flatMap((query) =>
    query.data ? [query.data] : [],
  );
  const canCompare = candidates.length >= 2;
  const summaries = ids.map(
    (id, index) =>
      queries[index]?.data ||
      comparison.candidates.find((candidate) => candidate.user_id === id) || {
        user_id: id,
        name: "Кандидат",
        grade: "",
        specialization: "",
      },
  );
  const summaryString = JSON.stringify(
    summaries.map(({ user_id, name, grade, specialization }) => ({
      user_id,
      name,
      grade,
      specialization,
    })),
  );
  useEffect(() => {
    comparison.synchronize(JSON.parse(summaryString), context, returnTo);
  }, [comparison.synchronize, summaryString, context, returnTo]);
  const remove = (id: string) => {
    comparison.remove(id);
    const next = new URLSearchParams(params);
    const remaining = ids.filter((value) => value !== id);
    if (remaining.length) next.set("ids", remaining.join(","));
    else next.delete("ids");
    setParams(next, { replace: true });
  };
  const requestedSkills = [
    ...new Map(
      candidates
        .flatMap((candidate) => [
          ...candidate.match.matched,
          ...candidate.match.missing,
        ])
        .map((skill) => [normalize(skill), skill]),
    ).values(),
  ].map(
    (skill) =>
      candidates
        .flatMap((candidate) => candidate.skills)
        .find((name) => normalize(name) === normalize(skill)) || skill,
  );
  const evidenceFor = (candidate: Candidate) =>
    evidenceQueries[ids.indexOf(candidate.user_id)]?.data;
  const differing = (row: Criterion) =>
    new Set(
      candidates.map((candidate) =>
        row.value(candidate, evidenceFor(candidate)),
      ),
    ).size > 1;
  const groups: Group[] = [
    {
      label: "Квалификация",
      rows: [
        {
          id: "specialization",
          label: "Направление",
          value: (c) => c.specialization,
          render: (c) => specs[c.specialization] || "Не указано",
        },
        {
          id: "grade",
          label: "Уровень",
          value: (c) => c.grade,
          render: (c) => (
            <span className="tag grade">{grades[c.grade] || "Не указан"}</span>
          ),
        },
        {
          id: "score",
          label: "Результат теста",
          hint: "Внутри одной категории",
          value: (c) => c.test_score,
          render: (c) => number(c.test_score),
        },
        {
          id: "source",
          label: "Основание категории",
          value: (_, evidence) =>
            `${evidence?.source}:${evidence?.confirmed_at ? date(evidence.confirmed_at) : ""}`,
          render: (_, evidence) => (
            <div className="comparison-source">
              <span>
                <ShieldCheck size={15} />
                {evidenceLabel(evidence)}
              </span>
              {evidence?.confirmed_at && (
                <small>{date(evidence.confirmed_at)}</small>
              )}
              {evidence?.source === "demo" && (
                <small>Синтетические данные</small>
              )}
            </div>
          ),
        },
      ],
    },
    {
      label: "Соответствие подбору",
      rows: [
        {
          id: "match",
          label: "Индекс подбора",
          hint: "Тест, стек и история ФСП",
          value: (c) => c.match_score,
          render: (c) => (
            <div className="comparison-match">
              {number(c.match_score)}
              <span>
                Тест {c.match.test}/{c.match.test_max}
                {c.match.stack_max > 0 &&
                  ` · Стек ${c.match.stack}/${c.match.stack_max}`}
                {c.match.fsp_max > 0 &&
                  ` · ФСП ${c.match.fsp}/${c.match.fsp_max}`}
              </span>
            </div>
          ),
        },
      ],
    },
    {
      label: "Условия работы",
      rows: [
        {
          id: "salary",
          label: "Зарплатные ожидания",
          hint: "В месяц, до налогов",
          value: (c) => salaryExpectation(c.professional),
          render: (c) => salaryExpectation(c.professional),
        },
        {
          id: "formats",
          label: "Формат работы",
          value: (c) =>
            professionalData(c.professional)
              .work_formats.slice()
              .sort()
              .join(","),
          render: (c) =>
            professionalData(c.professional)
              .work_formats.map((v) => workFormats[v])
              .join(", ") || "Не указан",
        },
        {
          id: "employment",
          label: "Занятость",
          value: (c) => c.professional.employment || "",
          render: (c) =>
            employmentTypes[c.professional.employment] || "Не указана",
        },
        {
          id: "availability",
          label: "Готовность выйти",
          value: (c) => c.professional.availability || "",
          render: (c) =>
            availabilityTypes[c.professional.availability] || "Не указана",
        },
        {
          id: "search-status",
          label: "Поиск работы",
          value: (c) => c.professional.search_status || "",
          render: (c) =>
            searchStatuses[c.professional.search_status] || "Не указан",
        },
        {
          id: "relocation",
          label: "Переезд",
          value: (c) => String(c.professional.relocation),
          render: (c) =>
            c.professional.relocation ? "Готов к переезду" : "Не указан",
        },
        {
          id: "languages",
          label: "Языки",
          value: (c) => professionalData(c.professional).languages.join(","),
          render: (c) =>
            professionalData(c.professional).languages.join(", ") ||
            "Не указаны",
        },
      ],
    },
    ...(requestedSkills.length
      ? [
          {
            label: "Нужные технологии",
            rows: requestedSkills.map((skill): Criterion => ({
              id: `skill-${normalize(skill)}`,
              label: skill,
              hint: "Из текущего запроса",
              value: (c) =>
                String(
                  c.match.matched.some(
                    (item) => normalize(item) === normalize(skill),
                  ),
                ),
              render: (c) =>
                c.match.matched.some(
                  (item) => normalize(item) === normalize(skill),
                ) ? (
                  <span className="comparison-skill present">
                    <Check size={16} />
                    Есть в резюме
                  </span>
                ) : (
                  <span className="comparison-skill missing">
                    <Minus size={16} />
                    Не указан
                  </span>
                ),
            })),
          },
        ]
      : []),
    {
      label: "Опыт и практика",
      rows: [
        {
          id: "experience",
          label: "Опыт работы",
          hint: "Указан кандидатом",
          value: (c) => c.experience_years,
          render: (c) =>
            `${c.experience_years} ${plural(c.experience_years, ["год", "года", "лет"])}`,
        },
        {
          id: "stack",
          label: "Весь стек",
          hint: "Из резюме",
          value: (c) => c.skills.map(normalize).sort().join(","),
          render: (c) => (
            <div className="tags">
              {c.skills.length ? (
                c.skills.map((skill) => (
                  <span className="skill" key={skill}>
                    {skill}
                  </span>
                ))
              ) : (
                <span className="muted">Не указан</span>
              )}
            </div>
          ),
        },
        {
          id: "practice",
          label: "Практика от работодателя",
          value: (c) => c.last_practice_at || "",
          render: (c) =>
            c.last_practice_at ? (
              <span className="comparison-skill present">
                <Check size={16} />
                {date(c.last_practice_at)}
              </span>
            ) : (
              <span className="muted">Ещё не подтверждена</span>
            ),
        },
        {
          id: "fsp",
          label: "Достижения ФСП",
          value: (c) =>
            JSON.stringify(
              c.achievements
                .map((a) => [a.title, a.year, a.place, a.source])
                .sort(),
            ),
          render: (c) =>
            c.achievements.length ? (
              <div className="comparison-source">
                <strong>
                  {c.achievements.length}{" "}
                  {plural(c.achievements.length, [
                    "достижение",
                    "достижения",
                    "достижений",
                  ])}
                </strong>
                <ul className="comparison-achievements">
                  {c.achievements.slice(0, 3).map((achievement, index) => (
                    <li key={index}>
                      {achievement.title}
                      <small>
                        {achievement.year} · {achievement.place}-е место
                      </small>
                    </li>
                  ))}
                </ul>
                {c.achievements.length > 3 && (
                  <small>Ещё {c.achievements.length - 3} в профиле</small>
                )}
                {c.achievements.some(
                  (achievement) => achievement.source === "demo",
                ) && <small>Есть демонстрационные данные</small>}
              </div>
            ) : (
              <span className="muted">Без истории ФСП</span>
            ),
        },
        {
          id: "city",
          label: "Город",
          value: (c) => c.city.trim().toLocaleLowerCase("ru-RU"),
          render: (c) => c.city || <span className="muted">Не указан</span>,
        },
      ],
    },
  ];
  const visibleGroups = groups
    .map((group) => ({
      ...group,
      rows: group.rows.filter(
        (row) => !onlyDifferences || !canCompare || differing(row),
      ),
    }))
    .filter((group) => group.rows.length);
  const loading = queries.some((query) => query.isPending);

  return (
    <div className="comparison-page">
      <Link to={returnTo} className="text-link back">
        <ArrowLeft size={16} />
        {returnTo.startsWith("/shortlist") ? "К избранному" : "К подборке"}
      </Link>
      <PageTitle eyebrow="Выбор специалиста" title="Сравнение кандидатов" />
      {!ids.length ? (
        <section className="comparison-empty">
          <Columns2 size={28} />
          <h2>С кем начнём сравнение?</h2>
          <p>
            Добавьте двух или трёх специалистов кнопкой «К сравнению» в
            подборке. Имена появятся в панели внизу экрана.
          </p>
          <Button asChild>
            <Link to={returnTo}>
              Выбрать кандидатов
              <ArrowRight size={16} />
            </Link>
          </Button>
        </section>
      ) : (
        <>
          <div className="comparison-controls">
            <div className="comparison-count">
              <Columns2 size={18} />
              <strong>{ids.length} из 3 кандидатов</strong>
              {ids.length < 3 && (
                <Link className="text-link" to={returnTo}>
                  <Plus size={15} />
                  Добавить
                </Link>
              )}
            </div>
            <label className="comparison-differences">
              <input
                type="checkbox"
                checked={onlyDifferences && canCompare}
                disabled={!canCompare || loading}
                onChange={(event) => {
                  const next = new URLSearchParams(params);
                  if (event.target.checked) next.set("differences", "true");
                  else next.delete("differences");
                  setParams(next, { replace: true });
                }}
              />
              <span>Только различия</span>
            </label>
          </div>
          {ids.length === 1 && (
            <div className="comparison-guidance">
              <span>
                Первый кандидат на месте.{" "}
                <Link to={returnTo}>Добавьте второго</Link>, чтобы сравнить их
                по одной строке.
              </span>
            </div>
          )}
          {context && (
            <div className="comparison-context">
              <span>Ваш запрос</span>
              <strong>
                {params.get("need") ||
                  [
                    specs[params.get("specialization") || ""],
                    grades[params.get("grade") || ""],
                    params
                      .get("skills")
                      ?.split(",")
                      .map((skill) => skill.trim())
                      .join(", "),
                    params.get("fsp") === "true" ? "Достижения ФСП" : "",
                  ]
                    .filter(Boolean)
                    .join(" · ")}
              </strong>
            </div>
          )}
          {loading ? (
            <Loading />
          ) : (
            <>
              <p className="comparison-mobile-hint">
                Листайте таблицу в сторону. Критерии и имена останутся на виду.
              </p>
              <div
                id="candidate-comparison"
                className="compare-scroll"
                data-columns={ids.length}
                role="region"
                aria-label="Сравнение кандидатов по критериям"
                tabIndex={0}
              >
                <table
                  className="compare-table"
                  style={
                    { "--comparison-columns": ids.length } as CSSProperties
                  }
                >
                  <caption className="sr-only">
                    Сравнение {ids.length} кандидатов. Каждый столбец —
                    кандидат; строки — общие критерии.
                  </caption>
                  <colgroup>
                    <col className="comparison-label-column" />
                    {ids.map((id) => (
                      <col key={id} />
                    ))}
                  </colgroup>
                  <thead>
                    <tr>
                      <th scope="col">
                        <span className="comparison-corner">Критерии</span>
                      </th>
                      {queries.map((query, index) => (
                        <th scope="col" key={ids[index]}>
                          <div className="comparison-person">
                            <button
                              type="button"
                              className="comparison-remove"
                              aria-label={`Убрать из сравнения: ${summaries[index].name}`}
                              onClick={() => remove(ids[index])}
                            >
                              <X size={17} />
                            </button>
                            <Avatar
                              name={query.data?.name || summaries[index].name}
                              userId={ids[index]}
                              version={query.data?.avatar_version}
                            />
                            {query.data ? (
                              <>
                                <Link
                                  to={`/candidates/${ids[index]}?${context}`}
                                  state={{ returnTo: comparePath }}
                                >
                                  {query.data.name}
                                </Link>
                                <p>
                                  {query.data.headline ||
                                    specs[query.data.specialization]}
                                </p>
                                <div className="comparison-person-actions">
                                  <SaveCandidate id={ids[index]} />
                                  <Link
                                    className="text-link"
                                    to={`/candidates/${ids[index]}?${context}`}
                                    state={{ returnTo: comparePath }}
                                  >
                                    Профиль
                                    <ArrowRight size={14} />
                                  </Link>
                                </div>
                              </>
                            ) : (
                              <>
                                <strong>{summaries[index].name}</strong>
                                <ErrorBox error={query.error} />
                                <Button
                                  size="sm"
                                  variant="secondary"
                                  onClick={() => void query.refetch()}
                                >
                                  Повторить
                                </Button>
                              </>
                            )}
                          </div>
                        </th>
                      ))}
                    </tr>
                  </thead>
                  {visibleGroups.map((group) => (
                    <tbody key={group.label}>
                      <tr className="comparison-group">
                        <th colSpan={ids.length + 1} scope="rowgroup">
                          {group.label}
                        </th>
                      </tr>
                      {group.rows.map((row) => (
                        <tr key={row.id} data-different={differing(row)}>
                          <th scope="row">
                            <span>{row.label}</span>
                            {row.hint && <small>{row.hint}</small>}
                          </th>
                          {queries.map((query, index) => (
                            <td key={ids[index]}>
                              {query.data ? (
                                <div className="compare-cell">
                                  {row.id === "source" &&
                                  evidenceQueries[index].isPending ? (
                                    <span className="muted">
                                      Загружаем подтверждение…
                                    </span>
                                  ) : (
                                    row.render(
                                      query.data,
                                      evidenceQueries[index].data,
                                    )
                                  )}
                                </div>
                              ) : (
                                <span className="muted">Недоступно</span>
                              )}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  ))}
                  <tfoot>
                    <tr>
                      <th scope="row">Следующий шаг</th>
                      {queries.map((query, index) => (
                        <td key={ids[index]}>
                          {query.data && (
                            <Button asChild size="sm" variant="secondary">
                              <Link
                                to={`/candidates/${ids[index]}?${context}`}
                                state={{ returnTo: comparePath }}
                              >
                                Профиль и предложение
                                <ArrowRight size={15} />
                              </Link>
                            </Button>
                          )}
                        </td>
                      ))}
                    </tr>
                  </tfoot>
                </table>
                {onlyDifferences && canCompare && !visibleGroups.length && (
                  <div className="comparison-no-differences">
                    <Check size={20} />
                    <p>По этим критериям профили совпадают.</p>
                    <button
                      type="button"
                      className="text-link"
                      onClick={() => {
                        const next = new URLSearchParams(params);
                        next.delete("differences");
                        setParams(next, { replace: true });
                      }}
                    >
                      Показать все критерии
                    </button>
                  </div>
                )}
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
}
