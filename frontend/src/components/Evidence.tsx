import { useQuery } from "@tanstack/react-query";
import { Check, CircleHelp, ShieldCheck } from "lucide-react";
import {
  api,
  date,
  type QualificationEvidence,
  type Match,
  grades,
  specs,
} from "../api";
import { ErrorBox, Loading } from "./shared";

export function EvidencePassport({ candidateId }: { candidateId?: string }) {
  const q = useQuery({
    queryKey: ["evidence", candidateId || "me"],
    queryFn: () =>
      api<QualificationEvidence>(
        candidateId
          ? `/candidates/${candidateId}/evidence`
          : "/profile/evidence",
      ),
  });
  return (
    <section className="evidence-passport" aria-label="Паспорт квалификации">
      <div className="section-heading">
        <h2>Паспорт квалификации</h2>
        <ShieldCheck size={20} />
      </div>
      <ErrorBox error={q.error} />
      {q.isPending ? (
        <Loading />
      ) : (
        q.data && (
          <>
            <div className="passport-stamp">
              <span>
                {specs[q.data.specialization] || "Направление не выбрано"}
              </span>
              <strong>
                {grades[q.data.grade] || "Уровень не подтверждён"}
              </strong>
            </div>
            {q.data.source === "assessment" ? (
              <>
                <p>
                  Подтверждено {date(q.data.confirmed_at!)} · тест{" "}
                  {q.data.version}
                </p>
                <div className="competency-grid">
                  {q.data.topics.map((t) => (
                    <div key={t.topic} className={t.passed ? "is-passed" : ""}>
                      {t.passed ? (
                        <Check size={16} />
                      ) : (
                        <CircleHelp size={16} />
                      )}
                      <span>{t.topic}</span>
                      <small>{t.passed ? "Засчитано" : "Нужно усилить"}</small>
                    </div>
                  ))}
                </div>
              </>
            ) : (
              <p className="muted">
                {q.data.source === "demo"
                  ? "Демонстрационный профиль: категория и баллы заданы в тестовом наборе. Реальная попытка появится здесь после прохождения теста."
                  : q.data.source === "imported"
                    ? "Категория импортирована. Детализация теста недоступна."
                    : "Пройдите тестирование, чтобы получить категорию и подтверждения по темам."}
              </p>
            )}
            {q.data.grade && (
              <dl className="passport-footer">
                <div>
                  <dt>Результат теста</dt>
                  <dd className="passport-score">
                    {q.data.score}
                    <span>/100</span>
                  </dd>
                </div>
                <div>
                  <dt>Практика от работодателя</dt>
                  <dd>
                    {q.data.last_practice_at
                      ? date(q.data.last_practice_at)
                      : "Ещё не подтверждена"}
                  </dd>
                </div>
              </dl>
            )}
            <small className="passport-note">
              Тест подтверждает специализацию и грейд. Перечень технологий и
              стаж указаны в резюме кандидата.
            </small>
          </>
        )
      )}
    </section>
  );
}

export function MatchBreakdown({ match }: { match: Match }) {
  return (
    <div className="match-breakdown">
      <p className="muted">
        Индекс подбора — {match.total}/100. Он помогает сравнить профили и не
        является вероятностью успешного найма.
      </p>
      {[
        {
          label: "Подтверждённый тест",
          value: match.test,
          max: match.test_max,
        },
        { label: "Стек из резюме", value: match.stack, max: match.stack_max },
        { label: "Достижения ФСП", value: match.fsp, max: match.fsp_max },
      ].map((p) => (
        <div className="score-component" key={p.label}>
          <span>{p.label}</span>
          <div className="score-track">
            <span
              style={{ width: `${p.max ? (p.value / p.max) * 100 : 0}%` }}
            />
          </div>
          <strong>{p.max ? `${p.value}/${p.max}` : "Не задан"}</strong>
        </div>
      ))}
      {match.matched.length > 0 && (
        <p>
          <Check size={15} />
          Совпали: {match.matched.join(", ")}
        </p>
      )}
      {match.missing.length > 0 && (
        <p className="missing-skills">
          <CircleHelp size={15} />
          Уточнить на встрече: {match.missing.join(", ")}
        </p>
      )}
    </div>
  );
}
