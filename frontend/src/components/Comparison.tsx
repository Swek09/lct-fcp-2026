import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { ArrowRight, Check, Plus, X } from "lucide-react";
import { type Candidate, grades } from "../api";
import { Avatar } from "./Avatar";
import { Button } from "./ui/button";

export const COMPARISON_LIMIT = 3;
export const comparisonContextKeys = [
  "need",
  "skills",
  "specialization",
  "grade",
  "fsp",
];
type SelectedCandidate = Pick<
  Candidate,
  "user_id" | "name" | "grade" | "specialization"
> & { avatar_version?: number };
type Selection = {
  candidates: SelectedCandidate[];
  query: string;
  returnTo: string;
};
const emptySelection: Selection = {
  candidates: [],
  query: "",
  returnTo: "/candidates",
};
const storageKey = (userId: string) => `fsp:comparison:${userId}`;
export const validComparisonId = (id: unknown): id is string =>
  typeof id === "string" &&
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(id);
export const comparisonReturnTo = (value: string | null) =>
  value && /^\/(candidates|shortlist)(\?|$)/.test(value)
    ? value
    : "/candidates";
export function comparisonQuery(params: URLSearchParams) {
  const query = new URLSearchParams();
  for (const key of comparisonContextKeys)
    if (params.get(key)) query.set(key, params.get(key)!);
  return query.toString();
}
const summary = (candidate: SelectedCandidate): SelectedCandidate => ({
  user_id: candidate.user_id,
  name: candidate.name,
  grade: candidate.grade,
  specialization: candidate.specialization,
  avatar_version: candidate.avatar_version,
});
export function clearComparisonSession(userId: string) {
  try {
    sessionStorage.removeItem(storageKey(userId));
  } catch {
    /* Selection still works without storage. */
  }
}
function restore(userId: string): Selection {
  try {
    const value = JSON.parse(
      sessionStorage.getItem(storageKey(userId)) || "null",
    );
    if (!value || !Array.isArray(value.candidates)) return emptySelection;
    const candidates = value.candidates.filter(
      (candidate: SelectedCandidate) =>
        candidate &&
        validComparisonId(candidate.user_id) &&
        typeof candidate.name === "string" &&
        typeof candidate.grade === "string" &&
        typeof candidate.specialization === "string",
    );
    return {
      candidates: candidates
        .filter(
          (candidate: SelectedCandidate, index: number) =>
            candidates.findIndex(
              (item: SelectedCandidate) => item.user_id === candidate.user_id,
            ) === index,
        )
        .slice(0, COMPARISON_LIMIT)
        .map(summary),
      query: comparisonQuery(
        new URLSearchParams(typeof value.query === "string" ? value.query : ""),
      ),
      returnTo: comparisonReturnTo(value.returnTo),
    };
  } catch {
    return emptySelection;
  }
}
type ComparisonState = Selection & {
  toggle: (candidate: Candidate, query?: string, returnTo?: string) => void;
  remove: (id: string) => void;
  clear: () => void;
  setContext: (query: string, returnTo: string) => void;
  synchronize: (
    candidates: SelectedCandidate[],
    query: string,
    returnTo: string,
  ) => void;
};
const ComparisonContext = createContext<ComparisonState | null>(null);
export function ComparisonProvider({
  userId,
  children,
}: {
  userId: string;
  children: ReactNode;
}) {
  const [selection, setSelection] = useState<Selection>(() => restore(userId));
  useEffect(() => {
    try {
      sessionStorage.setItem(storageKey(userId), JSON.stringify(selection));
    } catch {
      /* Private browsing can disable storage. */
    }
  }, [selection, userId]);
  const toggle = useCallback(
    (candidate: Candidate, query?: string, returnTo?: string) => {
      setSelection((previous) => {
        const exists = previous.candidates.some(
          (item) => item.user_id === candidate.user_id,
        );
        if (!exists && previous.candidates.length >= COMPARISON_LIMIT)
          return previous;
        return {
          candidates: exists
            ? previous.candidates.filter(
                (item) => item.user_id !== candidate.user_id,
              )
            : [...previous.candidates, summary(candidate)],
          query: query ?? previous.query,
          returnTo: returnTo ? comparisonReturnTo(returnTo) : previous.returnTo,
        };
      });
    },
    [],
  );
  const remove = useCallback(
    (id: string) =>
      setSelection((previous) => ({
        ...previous,
        candidates: previous.candidates.filter((item) => item.user_id !== id),
      })),
    [],
  );
  const clear = useCallback(() => setSelection(emptySelection), []);
  const setContext = useCallback(
    (query: string, returnTo: string) =>
      setSelection((previous) =>
        previous.query === query && previous.returnTo === returnTo
          ? previous
          : { ...previous, query, returnTo: comparisonReturnTo(returnTo) },
      ),
    [],
  );
  const synchronize = useCallback(
    (candidates: SelectedCandidate[], query: string, returnTo: string) => {
      setSelection((previous) => {
        const next = {
          candidates: candidates.map(summary),
          query,
          returnTo: comparisonReturnTo(returnTo),
        };
        return JSON.stringify(next) === JSON.stringify(previous)
          ? previous
          : next;
      });
    },
    [],
  );
  return (
    <ComparisonContext.Provider
      value={{ ...selection, toggle, remove, clear, setContext, synchronize }}
    >
      {children}
    </ComparisonContext.Provider>
  );
}
export function useComparison() {
  const state = useContext(ComparisonContext);
  if (!state) throw new Error("ComparisonProvider is missing");
  return state;
}
export function CompareCandidate({
  candidate,
  query,
  returnTo,
}: {
  candidate: Candidate;
  query?: string;
  returnTo?: string;
}) {
  const comparison = useComparison();
  const selected = comparison.candidates.some(
    (item) => item.user_id === candidate.user_id,
  );
  const full = !selected && comparison.candidates.length >= COMPARISON_LIMIT;
  return (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      className="compare-toggle"
      aria-pressed={selected}
      aria-label={`${selected ? "Убрать из сравнения" : "Добавить к сравнению"}: ${candidate.name}`}
      aria-controls="candidate-comparison"
      aria-describedby={full ? "comparison-limit" : undefined}
      title={
        full
          ? "Можно сравнить троих. Уберите кандидата из панели, чтобы добавить другого."
          : undefined
      }
      disabled={full}
      onClick={() => comparison.toggle(candidate, query, returnTo)}
    >
      {selected ? <Check size={16} /> : <Plus size={16} />}
      {selected ? "В сравнении" : "К сравнению"}
    </Button>
  );
}
export function ComparisonDock() {
  const comparison = useComparison();
  const location = useLocation();
  const navigate = useNavigate();
  const count = comparison.candidates.length;
  const hidden = location.pathname === "/compare";
  const compare = () => {
    const params = new URLSearchParams(comparison.query);
    params.set(
      "ids",
      comparison.candidates.map((candidate) => candidate.user_id).join(","),
    );
    params.set("from", comparison.returnTo);
    navigate(`/compare?${params}`);
  };
  return (
    <>
      <span
        className="sr-only"
        role="status"
        aria-live="polite"
        aria-atomic="true"
      >
        {count
          ? `Для сравнения выбрано ${count} из ${COMPARISON_LIMIT}. ${count === 1 ? "Выберите ещё одного кандидата." : "Можно открыть сравнение."}`
          : "Список сравнения пуст."}
      </span>
      <span id="comparison-limit" className="sr-only">
        Можно сравнить троих. Уберите кандидата из панели, чтобы добавить
        другого.
      </span>
      {!!count && !hidden && (
        <section
          id="candidate-comparison"
          className="comparison-dock"
          aria-label="Выбранные кандидаты"
        >
          <ul className="comparison-slots">
            {comparison.candidates.map((candidate) => (
              <li key={candidate.user_id}>
                <Avatar
                  name={candidate.name}
                  userId={candidate.user_id}
                  version={candidate.avatar_version}
                  size="small"
                />
                <span className="comparison-slot-name">
                  <strong title={candidate.name}>{candidate.name}</strong>
                  <small>{grades[candidate.grade] || "Профиль"}</small>
                </span>
                <button
                  type="button"
                  className="comparison-remove"
                  aria-label={`Убрать из сравнения: ${candidate.name}`}
                  onClick={() => comparison.remove(candidate.user_id)}
                >
                  <X size={16} />
                </button>
              </li>
            ))}
            {Array.from({ length: COMPARISON_LIMIT - count }, (_, index) => (
              <li
                className="comparison-slot-empty"
                key={`empty-${index}`}
                aria-hidden="true"
              >
                <Plus size={16} />
                <span>Ещё кандидат</span>
              </li>
            ))}
          </ul>
          <div className="comparison-dock-actions">
            <Button type="button" disabled={count < 2} onClick={compare}>
              Сравнить {count}
              <ArrowRight size={16} />
            </Button>
          </div>
        </section>
      )}
    </>
  );
}
