import { useEffect, useState, type ReactNode } from "react";
import { AlertCircle, ArrowRight, CheckCircle2, Loader2 } from "lucide-react";
import { Link } from "react-router-dom";
export function ErrorBox({ error }: { error: unknown }) {
  return error ? (
    <div className="notice error" role="alert">
      <AlertCircle size={18} />
      <span>{error instanceof Error ? error.message : String(error)}</span>
    </div>
  ) : null;
}
export function Success({ children }: { children: ReactNode }) {
  return (
    <div className="notice success" role="status">
      <CheckCircle2 size={18} />
      <span>{children}</span>
    </div>
  );
}
export function Loading() {
  const [online, setOnline] = useState(navigator.onLine);
  useEffect(() => {
    const update = () => setOnline(navigator.onLine);
    window.addEventListener("online", update);
    window.addEventListener("offline", update);
    return () => {
      window.removeEventListener("online", update);
      window.removeEventListener("offline", update);
    };
  }, []);
  if (!online)
    return (
      <div className="notice" role="status">
        <AlertCircle size={18} />
        <span>
          Нет подключения к сети. Загрузка продолжится автоматически, когда
          связь восстановится.
        </span>
      </div>
    );
  return (
    <div className="loading" role="status">
      <Loader2 className="spin" size={20} /> Загружаем данные…
    </div>
  );
}
export function PageTitle({
  title,
  children,
  action,
}: {
  eyebrow?: string;
  title: string;
  children?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <header className="page-heading">
      <div>
        <h1>{title}</h1>
        {children && <p className="subtitle">{children}</p>}
      </div>
      {action}
    </header>
  );
}
export function Empty({
  title,
  children,
  to,
  link,
}: {
  title: string;
  children: ReactNode;
  to?: string;
  link?: string;
}) {
  return (
    <div className="empty">
      <span className="empty-symbol">[ &nbsp; ]</span>
      <h2>{title}</h2>
      <p>{children}</p>
      {to && (
        <Link className="text-link" to={to}>
          {link}
          <ArrowRight size={16} />
        </Link>
      )}
    </div>
  );
}
export function Field({
  label,
  children,
  hint,
}: {
  label: string;
  children: ReactNode;
  hint?: string;
}) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
      {hint && <small>{hint}</small>}
    </label>
  );
}
