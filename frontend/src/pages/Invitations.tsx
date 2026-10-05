import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ArrowRight, Building2, Check, Clock3, Mail, X } from "lucide-react";
import { api, date, type Invitation, money, queryClient } from "../api";
import { useUser } from "../App";
import { Button } from "../components/ui/button";
import { Empty, ErrorBox, Loading, PageTitle } from "../components/shared";
const statuses: Record<string, string> = {
  sent: "Отправлено",
  viewed: "Просмотрено",
  accepted: "Принято",
  declined: "Отклонено",
};
export function Invitations() {
  const me = useUser();
  const candidate = me.data?.role === "candidate";
  const q = useQuery({
    queryKey: ["invitations"],
    queryFn: () => api<Invitation[]>("/invitations"),
    refetchInterval: 15000,
  });
  const [filter, setFilter] = useState("all");
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState("");
  const [expanded, setExpanded] = useState<string[]>([]);
  const action = async (id: string, status: string) => {
    setBusy(id);
    setError(undefined);
    try {
      await api("/invitations/" + id, "PATCH", { status });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["invitations"] }),
        queryClient.invalidateQueries({ queryKey: ["candidate"] }),
        queryClient.invalidateQueries({ queryKey: ["candidates"] }),
      ]);
    } catch (e) {
      setError(e);
    } finally {
      setBusy("");
    }
  };
  const expand = (i: Invitation) => {
    setExpanded((v) =>
      v.includes(i.id) ? v.filter((id) => id !== i.id) : [...v, i.id],
    );
    if (candidate && i.status === "sent") void action(i.id, "viewed");
  };
  const list = q.data?.filter(
    (i) =>
      filter === "all" ||
      (filter === "active"
        ? ["sent", "viewed"].includes(i.status)
        : i.status === filter),
  );
  return (
    <>
      <PageTitle
        eyebrow={candidate ? "Входящие приглашения" : "Контакт с кандидатами"}
        title={candidate ? "Возможности для вас" : "Ваши приглашения"}
      >
        {candidate
          ? "Команды уже сделали первый шаг. Выберите предложение, которое вам подходит."
          : "Все предложения и решения кандидатов в одном месте."}
      </PageTitle>
      <div className="tabs" role="group" aria-label="Фильтр приглашений">
        {[
          ["all", "Все"],
          ["active", "В ожидании"],
          ["accepted", "Принятые"],
          ["declined", "Отклонённые"],
        ].map(([k, v]) => (
          <button
            className={filter === k ? "active" : ""}
            key={k}
            aria-pressed={filter === k}
            onClick={() => setFilter(k)}
          >
            {v}
            {k === "all" && <span>{q.data?.length || 0}</span>}
          </button>
        ))}
      </div>
      <ErrorBox error={q.error || error} />
      {q.isPending ? (
        <Loading />
      ) : q.error && !q.data ? (
        <Button variant="secondary" onClick={() => void q.refetch()}>
          Повторить загрузку
        </Button>
      ) : list?.length ? (
        <div className="invitation-list">
          {list.map((i) => (
            <article className="invitation" key={i.id}>
              <div className="invitation-top">
                <div className="company-mark">
                  <Building2 size={22} />
                </div>
                <div>
                  <span className="eyebrow">
                    {candidate ? i.company_name : i.candidate_name}
                  </span>
                  <h2>{i.title}</h2>
                </div>
                <span className={"tag status-" + i.status}>
                  {statuses[i.status]}
                </span>
              </div>
              <div className="salary">
                {money(i.salary_min)} — {money(i.salary_max)} ₽
                <span>в месяц, до вычета налогов</span>
              </div>
              <div className="invitation-bottom">
                <span>
                  <Clock3 size={14} />
                  {date(i.created_at)}
                </span>
                <button
                  className="text-link"
                  onClick={() => expand(i)}
                  aria-expanded={expanded.includes(i.id)}
                >
                  {expanded.includes(i.id) ? "Свернуть" : "Подробнее"}
                  <ArrowRight size={15} />
                </button>
              </div>
              {expanded.includes(i.id) && (
                <div className="invitation-body">
                  <p>{i.message}</p>
                  <a className="text-link" href={"mailto:" + i.company_email}>
                    <Mail size={16} />
                    {i.company_email}
                  </a>
                  {!candidate && i.candidate_email && (
                    <div className="notice success">
                      <Check size={18} />
                      <span>
                        Кандидат принял приглашение:{" "}
                        <a href={"mailto:" + i.candidate_email}>
                          {i.candidate_email}
                        </a>
                      </span>
                    </div>
                  )}
                  {candidate && ["sent", "viewed"].includes(i.status) && (
                    <>
                      <p className="muted">
                        Принимая приглашение, вы открываете этой компании свою
                        электронную почту.
                      </p>
                      <div className="actions">
                        <Button
                          disabled={busy === i.id}
                          onClick={() => action(i.id, "accepted")}
                        >
                          <Check size={16} />
                          Принять приглашение
                        </Button>
                        <Button
                          variant="secondary"
                          disabled={busy === i.id}
                          onClick={() => action(i.id, "declined")}
                        >
                          <X size={16} />
                          Отклонить
                        </Button>
                      </div>
                    </>
                  )}
                  {!candidate && (
                    <Link
                      className="text-link"
                      to={"/candidates/" + i.candidate_id}
                    >
                      Профиль кандидата
                      <ArrowRight size={16} />
                    </Link>
                  )}
                </div>
              )}
            </article>
          ))}
        </div>
      ) : (
        <Empty
          title={
            filter !== "all"
              ? "В этой группе пока нет приглашений"
              : candidate
                ? "Предложения ещё впереди"
                : "Здесь пока нет приглашений"
          }
          to={
            filter !== "all"
              ? undefined
              : candidate
                ? "/profile"
                : "/candidates"
          }
          link={candidate ? "Проверить профиль" : "Найти кандидатов"}
        >
          {filter !== "all"
            ? "Выберите «Все», чтобы посмотреть остальные предложения."
            : candidate
              ? "Подтвердите уровень и разрешите публикацию профиля. Тогда компании смогут вас найти."
              : "Откройте профиль подходящего специалиста и отправьте предложение с зарплатой."}
        </Empty>
      )}
    </>
  );
}
