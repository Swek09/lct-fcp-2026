import { useQueries } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import { ArrowLeft, ArrowRight, ShieldCheck } from "lucide-react";
import { api, type Candidate, date, grades, specs } from "../api";
import { MatchBreakdown } from "../components/Evidence";
import { SaveCandidate } from "../components/SaveCandidate";
import { Empty, ErrorBox, Loading, PageTitle } from "../components/shared";
import { Button } from "../components/ui/button";

export function Compare() {
  const [params] = useSearchParams();
  const ids = [
    ...new Set((params.get("ids") || "").split(",").filter(Boolean)),
  ].slice(0, 3);
  const context = new URLSearchParams();
  for (const key of ["need", "skills"])
    if (params.get(key)) context.set(key, params.get(key)!);
  const qs = useQueries({
    queries: ids.map((id) => ({
      queryKey: ["candidate", id, context.toString()],
      queryFn: () =>
        api<Candidate>(`/candidates/${encodeURIComponent(id)}?${context}`),
    })),
  });
  return (
    <>
      <Link to="/candidates" className="text-link back">
        <ArrowLeft size={16} />К подборке
      </Link>
      <PageTitle
        eyebrow="Решение на основе подтверждений"
        title="Сравнить кандидатов"
      >
        До трёх специалистов рядом. Индекс подбора, сильные стороны и вопросы,
        которые стоит уточнить на встрече.
      </PageTitle>
      {ids.length < 2 ? (
        <Empty
          title="Выберите минимум двух кандидатов"
          to="/candidates"
          link="Открыть подборку"
        >
          Отметьте специалистов в подборке или избранном и нажмите «Сравнить».
        </Empty>
      ) : qs.some((q) => q.isPending) ? (
        <Loading />
      ) : (
        <>
          {qs.map((q, i) => (
            <ErrorBox key={ids[i]} error={q.error} />
          ))}
          <div
            className="compare-scroll"
            role="region"
            aria-label="Таблица сравнения кандидатов"
            tabIndex={0}
          >
            <table className="compare-table">
              <thead>
                <tr>
                  <th scope="col">Основание</th>
                  {qs.map((q, i) => (
                    <th scope="col" key={ids[i]}>
                      {q.data ? (
                        <>
                          <Link to={`/candidates/${q.data.user_id}?${context}`}>
                            {q.data.name}
                          </Link>
                          <p>{q.data.headline}</p>
                          <SaveCandidate id={q.data.user_id} />
                        </>
                      ) : (
                        "Профиль недоступен"
                      )}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {[
                  {
                    label: "Подтверждённая категория",
                    render: (c: Candidate) => (
                      <>
                        <ShieldCheck size={16} />
                        {specs[c.specialization]}
                        <strong>{grades[c.grade]}</strong>
                      </>
                    ),
                  },
                  {
                    label: "Результат теста",
                    render: (c: Candidate) => (
                      <>
                        <strong className="compare-score">
                          {c.test_score}
                          <small>/100</small>
                        </strong>
                        <small>
                          Для демонстрационных профилей баллы заданы в наборе
                          данных.
                        </small>
                      </>
                    ),
                  },
                  {
                    label: "Соответствие запросу",
                    render: (c: Candidate) => (
                      <MatchBreakdown match={c.match} />
                    ),
                  },
                  {
                    label: "Стек из резюме",
                    render: (c: Candidate) => (
                      <div className="tags">
                        {c.skills.map((s) => (
                          <span className="skill" key={s}>
                            {s}
                          </span>
                        ))}
                      </div>
                    ),
                  },
                  {
                    label: "Практика от работодателя",
                    render: (c: Candidate) =>
                      c.last_practice_at
                        ? date(c.last_practice_at)
                        : "Ещё не подтверждена",
                  },
                  {
                    label: "История ФСП",
                    render: (c: Candidate) =>
                      c.achievements.length ? (
                        <>
                          {c.achievements.length} достижений
                          <small>Демонстрационные данные</small>
                        </>
                      ) : (
                        "Без истории ФСП"
                      ),
                  },
                  {
                    label: "Опыт и город",
                    render: (c: Candidate) => (
                      <>
                        {c.experience_years} лет · {c.city || "Город не указан"}
                        <small>Указано кандидатом</small>
                      </>
                    ),
                  },
                  {
                    label: "Следующий шаг",
                    render: (c: Candidate) => (
                      <Button asChild size="sm">
                        <Link to={`/candidates/${c.user_id}?${context}`}>
                          Открыть и пригласить
                          <ArrowRight size={16} />
                        </Link>
                      </Button>
                    ),
                  },
                ].map((row) => (
                  <tr key={row.label}>
                    <th scope="row">{row.label}</th>
                    {qs.map((q, i) => (
                      <td key={ids[i]}>
                        {q.data ? (
                          <div className="compare-cell">
                            {row.render(q.data)}
                          </div>
                        ) : (
                          "—"
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </>
  );
}
