import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Link,
  useLocation,
  useNavigate,
  useSearchParams,
} from "react-router-dom";
import { ArrowRight, Check, Fingerprint } from "lucide-react";
import { api, type Config, queryClient, setCSRF, type User } from "../api";
import { Button } from "../components/ui/button";
import { ErrorBox, Field, Success } from "../components/shared";
function AuthShell({ children }: { children: ReactNode }) {
  return (
    <div className="auth-page">
      <section className="auth-story">
        <Link className="brand" to="/login">
          <img src="/brand/fsp-symbol.svg" alt="ФСП" />
          <div>
            Карьера<span>Федерация спортивного программирования</span>
          </div>
        </Link>
        <div className="auth-statement">
          <p className="eyebrow">Подтверждено делом</p>
          <h1>
            Ваши навыки.
            <br />
            Ваш следующий
            <br />
            <span>большой шаг.</span>
          </h1>
          <p>
            Работодатели находят вас по тому, что вы умеете. Подтвердите уровень
            и получайте предложения с открытой зарплатой.
          </p>
          <div className="journey">
            <div>
              <span>01</span>Профиль
            </div>
            <ArrowRight size={16} />
            <div>
              <span>02</span>Навыки
            </div>
            <ArrowRight size={16} />
            <div>
              <span>03</span>Предложение
            </div>
          </div>
        </div>
        <span className="auth-foot">
          Спортивный опыт. Профессиональное будущее.
        </span>
      </section>
      <section className="auth-form-panel">{children}</section>
    </div>
  );
}
export function Auth({ register = false }: { register?: boolean }) {
  const location = useLocation();
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [role, setRole] = useState("candidate");
  const [forgot, setForgot] = useState(false);
  const navigate = useNavigate();
  const config = useQuery({
    queryKey: ["config"],
    queryFn: () => api<Config>("/config"),
  });
  useEffect(() => {
    setMessage("");
    setError(undefined);
    setForgot(false);
  }, [register]);
  const login = async (email: string, password: string) => {
    const u = await api<User>("/auth/login", "POST", { email, password });
    setCSRF(u.csrf);
    queryClient.clear();
    queryClient.setQueryData(["me"], u);
    navigate("/overview");
  };
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(undefined);
    setBusy(true);
    const data = Object.fromEntries(new FormData(e.currentTarget));
    try {
      if (forgot) {
        const r = await api<{ message: string }>("/auth/forgot", "POST", {
          email: data.email,
        });
        setMessage(r.message);
      } else if (register) {
        const r = await api<{ message: string }>("/auth/register", "POST", {
          ...data,
          role,
          consent: data.consent === "on",
        });
        setMessage(r.message);
      } else await login(String(data.email), String(data.password));
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  async function demo(role: string) {
    setBusy(true);
    setError(undefined);
    try {
      await login(role + "@demo.fsp", "DemoPass2026!");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  return (
    <AuthShell>
      <div className="auth-form">
        <Fingerprint className="auth-mark" size={32} />
        <p className="eyebrow">
          {register ? "Начнём знакомство" : "Рады видеть вас"}
        </p>
        <h2>
          {forgot
            ? "Восстановить доступ"
            : register
              ? "Создать аккаунт"
              : "Войти в платформу"}
        </h2>
        <p className="muted">
          {register
            ? "Один шаг к новым профессиональным возможностям."
            : "Продолжите свой профессиональный путь."}
        </p>
        <ErrorBox error={error} />
        {!register && location.state?.sessionExpired && (
          <div className="notice" role="status">
            Сессия завершилась. Войдите снова, чтобы продолжить.
          </div>
        )}
        {message ? (
          <>
            <Success>{message}</Success>
            <p className="muted">Проверьте входящие письма и папку «Спам».</p>
            {config.data?.demo && (
              <a
                className="text-link"
                href="http://localhost:8025"
                target="_blank"
                rel="noreferrer"
              >
                Открыть почту демонстрационного стенда <ArrowRight size={16} />
              </a>
            )}
            <Button
              variant="secondary"
              onClick={() => {
                setMessage("");
                navigate("/login");
              }}
            >
              Вернуться ко входу
            </Button>
          </>
        ) : (
          <form onSubmit={submit} className="form-stack">
            {register && (
              <>
                <div className="segmented" aria-label="Тип аккаунта">
                  <button
                    type="button"
                    className={role === "candidate" ? "active" : ""}
                    onClick={() => setRole("candidate")}
                  >
                    Ищу возможности
                  </button>
                  <button
                    type="button"
                    className={role === "employer" ? "active" : ""}
                    onClick={() => setRole("employer")}
                  >
                    Ищу специалистов
                  </button>
                </div>
                <Field label="Ваше имя">
                  <input
                    name="name"
                    autoComplete="name"
                    required
                    minLength={2}
                    maxLength={100}
                    placeholder="Александр Морозов"
                  />
                </Field>
              </>
            )}
            <Field label="Электронная почта">
              <input
                name="email"
                type="email"
                autoComplete="email"
                required
                maxLength={254}
                placeholder="you@company.ru"
              />
            </Field>
            {!forgot && (
              <Field label="Пароль" hint={register ? "От 10 символов" : " "}>
                <input
                  name="password"
                  type="password"
                  autoComplete={register ? "new-password" : "current-password"}
                  required
                  minLength={register ? 10 : 1}
                  maxLength={72}
                  placeholder="Введите пароль"
                />
              </Field>
            )}
            {register && (
              <label className="check">
                <input type="checkbox" name="consent" required />
                <span>
                  Согласен на обработку данных аккаунта и профиля для работы
                  платформы. Публикацию профиля разрешу отдельно в настройках.
                </span>
              </label>
            )}
            <Button disabled={busy}>
              {busy
                ? "Подождите…"
                : forgot
                  ? "Отправить ссылку"
                  : register
                    ? "Создать аккаунт"
                    : "Войти"}
              <ArrowRight size={17} />
            </Button>
            {!register && !forgot && (
              <button
                className="text-link center"
                type="button"
                onClick={() => setForgot(true)}
              >
                Не помню пароль
              </button>
            )}
          </form>
        )}
        <p className="auth-switch">
          {register ? "Уже есть аккаунт?" : "Впервые здесь?"}{" "}
          <Link to={register ? "/login" : "/register"}>
            {register ? "Войти" : "Зарегистрироваться"}
          </Link>
        </p>
        {config.data?.demo && !register && !forgot && (
          <div className="demo-block">
            <span className="eyebrow">Посмотреть, как это работает</span>
            <div className="demo-actions">
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() => demo("candidate")}
              >
                Я кандидат
              </Button>
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() => demo("employer")}
              >
                Я работодатель
              </Button>
            </div>
            <small>Демоаккаунты · синтетические профили и достижения</small>
          </div>
        )}
      </div>
    </AuthShell>
  );
}
export function Verify() {
  const [params] = useSearchParams();
  const [error, setError] = useState<unknown>();
  const [message, setMessage] = useState("");
  const done = useRef(false);
  useEffect(() => {
    if (done.current) return;
    done.current = true;
    api<{ message: string }>("/auth/verify", "POST", {
      token: params.get("token") || "",
    })
      .then((v) => setMessage(v.message))
      .catch(setError);
  }, [params]);
  return (
    <AuthShell>
      <div className="auth-form">
        <Check size={36} />
        <h2>Подтверждение почты</h2>
        <ErrorBox error={error} />
        {message ? (
          <Success>{message}</Success>
        ) : (
          !error && <p>Проверяем ссылку…</p>
        )}
        <Button asChild>
          <Link to="/login">Перейти ко входу</Link>
        </Button>
      </div>
    </AuthShell>
  );
}
export function ResetPassword() {
  const [params] = useSearchParams();
  const [error, setError] = useState<unknown>();
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    const f = new FormData(e.currentTarget);
    try {
      const r = await api<{ message: string }>("/auth/reset", "POST", {
        token: params.get("token") || "",
        password: f.get("password"),
      });
      setMessage(r.message);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <AuthShell>
      <div className="auth-form">
        <h2>Новый пароль</h2>
        <ErrorBox error={error} />
        {message ? (
          <>
            <Success>{message}</Success>
            <Link to="/login" className="text-link">
              Войти
            </Link>
          </>
        ) : (
          <form className="form-stack" onSubmit={submit}>
            <Field label="Новый пароль">
              <input
                type="password"
                name="password"
                autoComplete="new-password"
                minLength={10}
                maxLength={72}
                required
              />
            </Field>
            <Button disabled={busy}>Сохранить пароль</Button>
          </form>
        )}
      </div>
    </AuthShell>
  );
}
