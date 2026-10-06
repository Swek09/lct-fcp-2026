import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import { ArrowRight, Mail } from "lucide-react";
import {
  api,
  type Application,
  applicationStatuses,
  date,
  grades,
  money,
  queryClient,
  specs,
} from "../api";
import { useUser } from "../App";
import { Button } from "../components/ui/button";
import { Empty, ErrorBox, Loading, PageTitle } from "../components/shared";

export function Applications() {
  const employer = useUser().data?.role === "employer";
  const [params] = useSearchParams();
  const [filter, setFilter] = useState("all");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState<unknown>();
  const q = useQuery({
    queryKey: ["applications", params.toString()],
    queryFn: () => api<Application[]>("/applications?" + params),
    refetchInterval: 15000,
  });
  const change = async (id: string, status: string) => {
    setBusy(id);
    setError(undefined);
    try {
      await api(`/applications/${id}`, "PATCH", { status });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["applications"] }),
        queryClient.invalidateQueries({ queryKey: ["vacancies"] }),
        queryClient.invalidateQueries({ queryKey: ["vacancy"] }),
        queryClient.invalidateQueries({ queryKey: ["candidate"] }),
      ]);
    } catch (e) {
      setError(e);
    } finally {
      setBusy("");
    }
  };
  const rows =
    q.data?.filter((a) => filter === "all" || a.status === filter) || [];
  return (
    <>
      <PageTitle
        eyebrow={employer ? "Обращения кандидатов" : "Ваши обращения к компаниям"}
        title={employer ? "Входящие отклики" : "Мои отклики"}
      >
        {employer
          ? "Смотрите подтверждённую категорию и сопроводительное письмо. Контакты переданы кандидатом только вашей компании."
          : "Решение компании видно здесь. Отзыв отклика прекращает доступ этой компании к контактам по этому отклику."}
      </PageTitle>
      <div className="tabs" aria-label="Статус откликов">
        {["all", ...Object.keys(applicationStatuses)].map((k) => (
          <button
            key={k}
            aria-pressed={k === filter}
            className={k === filter ? "active" : ""}
            onClick={() => setFilter(k)}
          >
            {k === "all" ? "Все" : applicationStatuses[k]}
            {q.data && (
              <span>
                {k === "all"
                  ? q.data.length
                  : q.data.filter((a) => a.status === k).length}
              </span>
            )}
          </button>
        ))}
      </div>
      <ErrorBox error={error || q.error} />
      {q.isPending ? (
        <Loading />
      ) : q.error && !q.data ? (
        <Button onClick={() => void q.refetch()}>Повторить загрузку</Button>
      ) : rows.length ? (
        <div className="application-list">
          {rows.map((a) => (
            <article className="application-row" key={a.id}>
              <div className="application-top">
                <div>
                  <p className="eyebrow">
                    {employer ? a.vacancy_title : a.company_name}
                  </p>
                  <h2>{employer ? a.candidate_name : a.vacancy_title}</h2>
                  <p className="muted">
                    {employer
                      ? `${specs[a.specialization]} · ${grades[a.grade]} · тест ${a.test_score}/100`
                      : `${money(a.salary_min)}–${money(a.salary_max)} ₽ в месяц`}
                  </p>
                </div>
                <span className={`status ${a.status}`}>
                  {applicationStatuses[a.status]}
                </span>
              </div>
              {a.cover_letter && <p className="long-copy">{a.cover_letter}</p>}
              <div className="application-bottom">
                <small>{date(a.created_at)}</small>
                {employer ? (
                  <>
                    <Link
                      className="text-link"
                      to={`/candidates/${a.candidate_id}`}
                    >
                      Профиль
                      <ArrowRight size={15} />
                    </Link>
                    {a.candidate_email && (
                      <a
                        className="text-link"
                        href={`mailto:${a.candidate_email}`}
                      >
                        <Mail size={15} />
                        Написать
                      </a>
                    )}
                    {["new", "viewed"].includes(a.status) && (
                      <>
                        <Button
                          size="sm"
                          variant="secondary"
                          disabled={busy === a.id}
                          onClick={() => void change(a.id, "shortlisted")}
                        >
                          В подборку
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={busy === a.id}
                          onClick={() => void change(a.id, "rejected")}
                        >
                          Не подходит
                        </Button>
                      </>
                    )}
                    {a.status === "new" && (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={busy === a.id}
                        onClick={() => void change(a.id, "viewed")}
                      >
                        Отметить рассмотренным
                      </Button>
                    )}
                  </>
                ) : (
                  <>
                    <Link
                      className="text-link"
                      to={`/vacancies/${a.vacancy_id}`}
                    >
                      Вакансия
                      <ArrowRight size={15} />
                    </Link>
                    {a.status !== "withdrawn" && (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={busy === a.id}
                        onClick={() => void change(a.id, "withdrawn")}
                      >
                        Отозвать отклик
                      </Button>
                    )}
                  </>
                )}
              </div>
              {a.status === "withdrawn" && (
                <small>Доступ к контактам по этому отклику отозван.</small>
              )}
            </article>
          ))}
        </div>
      ) : (
        <Empty
          title={
            filter === "all"
              ? "Откликов пока нет"
              : "Нет откликов с этим статусом"
          }
          to={employer ? "/vacancies" : "/vacancies"}
          link={employer ? "Открыть вакансии" : "Посмотреть вакансии"}
        >
          {employer
            ? "Опубликуйте вакансию с задачами и зарплатой или найдите кандидатов и пригласите их напрямую."
            : "Вы можете проявить инициативу сами. Это не мешает получать предложения от работодателей."}
        </Empty>
      )}
    </>
  );
}
