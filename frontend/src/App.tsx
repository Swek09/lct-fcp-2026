import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Navigate,
  NavLink,
  Outlet,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";
import {
  ArrowUpRight,
  Building2,
  Bookmark,
  BriefcaseBusiness,
  LayoutDashboard,
  FileCheck2,
  ChevronRight,
  ClipboardCheck,
  FileUser,
  Lightbulb,
  Inbox,
  LogOut,
  Menu,
  Search,
  ShieldCheck,
  X,
} from "lucide-react";
import {
  api,
  APIError,
  type Invitation,
  type Config,
  queryClient,
  setCSRF,
  type User,
} from "./api";
import { Auth, ResetPassword, Verify } from "./pages/Auth";
import { CandidateProfile } from "./pages/Profile";
import { Assessments } from "./pages/Assessments";
import { SearchPage, CandidateDetail } from "./pages/Search";
import { Invitations } from "./pages/Invitations";
import { Company, PublicCompany } from "./pages/Company";
import { Avatar } from "./components/Avatar";
import { Challenges } from "./pages/Challenges";
import { Overview } from "./pages/Overview";
import { Vacancies, VacancyDetail, VacancyEditor } from "./pages/Vacancies";
import { Applications } from "./pages/Applications";
import { Compare } from "./pages/Compare";
import {
  ComparisonDock,
  ComparisonProvider,
  clearComparisonSession,
} from "./components/Comparison";
import { ErrorBox, Loading } from "./components/shared";
import { Button } from "./components/ui/button";
export function useUser() {
  return useQuery({
    queryKey: ["me"],
    queryFn: async () => {
      const u = await api<User>("/auth/me");
      setCSRF(u.csrf);
      return u;
    },
  });
}
function Layout() {
  const me = useUser();
  const avatar = useQuery({
    queryKey: ["avatar"],
    queryFn: () => api<{ version: number }>("/avatar"),
    enabled: !!me.data,
  });
  const navigate = useNavigate();
  const location = useLocation();
  const [menu, setMenu] = useState(false);
  const [error, setError] = useState<unknown>();
  const config = useQuery({
    queryKey: ["config"],
    queryFn: () => api<Config>("/config"),
  });
  const invitations = useQuery({
    queryKey: ["invitations"],
    queryFn: () => api<Invitation[]>("/invitations"),
    enabled: !!me.data,
    refetchInterval: 15000,
  });
  const newInvitations =
    me.data?.role === "candidate"
      ? invitations.data?.filter((i) => i.status === "sent").length || 0
      : 0;
  useEffect(() => {
    const expired = () => {
      if (me.data) clearComparisonSession(me.data.id);
      queryClient.clear();
      navigate("/login", { replace: true, state: { sessionExpired: true } });
    };
    window.addEventListener("fsp:session-expired", expired);
    return () => window.removeEventListener("fsp:session-expired", expired);
  }, [navigate, me.data?.id]);
  useEffect(() => {
    if (!menu) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    document.querySelector<HTMLAnchorElement>(".sidebar nav a")?.focus();
    const escape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMenu(false);
        document.querySelector<HTMLButtonElement>(".mobile-menu")?.focus();
      }
      if (event.key === "Tab") {
        const controls = Array.from(
          document.querySelectorAll<HTMLElement>(
            ".mobile-menu, .sidebar a, .sidebar button",
          ),
        );
        const first = controls[0],
          last = controls.at(-1);
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last?.focus();
        }
        if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first?.focus();
        }
      }
    };
    const resized = () => {
      if (window.innerWidth > 760) setMenu(false);
    };
    window.addEventListener("keydown", escape);
    window.addEventListener("resize", resized);
    return () => {
      document.body.style.overflow = previous;
      window.removeEventListener("keydown", escape);
      window.removeEventListener("resize", resized);
    };
  }, [menu]);
  useEffect(() => {
    setMenu(false);
    document.getElementById("main")?.focus({ preventScroll: true });
    window.scrollTo({ top: 0 });
  }, [location.pathname]);
  if (me.isPending) return <Loading />;
  if (
    !me.data &&
    me.error &&
    !(me.error instanceof APIError && me.error.status === 401)
  )
    return (
      <div className="connection-error">
        <ErrorBox error={me.error} />
        <Button onClick={() => void me.refetch()}>Повторить загрузку</Button>
      </div>
    );
  if (!me.data) return <Navigate to="/login" replace />;
  const user = me.data;
  const links =
    user.role === "employer"
      ? [
          { to: "/overview", icon: LayoutDashboard, label: "Обзор" },
          { to: "/candidates", icon: Search, label: "Поиск кандидатов" },
          { to: "/shortlist", icon: Bookmark, label: "Избранное" },
          { to: "/invitations", icon: Inbox, label: "Приглашения" },
          { to: "/vacancies", icon: BriefcaseBusiness, label: "Вакансии" },
          { to: "/applications", icon: FileCheck2, label: "Отклики" },
          { to: "/challenges", icon: Lightbulb, label: "Задачи команды" },
          { to: "/company", icon: Building2, label: "Компания" },
        ]
      : [
          { to: "/overview", icon: LayoutDashboard, label: "Обзор" },
          { to: "/profile", icon: FileUser, label: "Мой профиль" },
          {
            to: "/assessments",
            icon: ClipboardCheck,
            label: "Подтвердить навыки",
          },
          { to: "/invitations", icon: Inbox, label: "Предложения" },
          { to: "/vacancies", icon: BriefcaseBusiness, label: "Вакансии" },
          { to: "/applications", icon: FileCheck2, label: "Мои отклики" },
          { to: "/challenges", icon: Lightbulb, label: "Короткие задания" },
        ];
  const logout = async () => {
    try {
      await api("/auth/logout", "POST");
      setCSRF("");
      clearComparisonSession(user.id);
      queryClient.clear();
      navigate("/login");
    } catch (e) {
      setError(e);
    }
  };
  return (
    <ComparisonProvider key={user.id} userId={user.id}>
      <div className="app-shell">
        <a className="skip-link" href="#main">
          К содержимому
        </a>
        <button
          className="mobile-menu"
          aria-label="Меню"
          aria-expanded={menu}
          aria-controls="main-navigation"
          onClick={() => setMenu(!menu)}
        >
          {menu ? <X /> : <Menu />}
        </button>
        {menu && (
          <button
            className="menu-backdrop"
            aria-label="Закрыть меню"
            onClick={() => setMenu(false)}
          />
        )}
        <aside
          id="main-navigation"
          role={menu ? "dialog" : undefined}
          aria-modal={menu ? true : undefined}
          aria-label={menu ? "Навигация" : undefined}
          className={"sidebar " + (menu ? "is-open" : "")}
        >
          <NavLink to="/" className="brand">
            <img src="/brand/fsp-symbol.svg" alt="ФСП" />
            <div>
              Карьера<span>Федерация спортивного программирования</span>
            </div>
          </NavLink>
          <div className="workspace-label">
            {user.role === "employer" ? "Работодатель" : "Участник платформы"}
          </div>
          <nav aria-label="Главная навигация">
            {links.map(({ to, icon: Icon, label }) => (
              <NavLink key={to} to={to}>
                <Icon size={19} />
                <span>{label}</span>
                {to === "/invitations" && newInvitations > 0 && (
                  <span
                    className="nav-count"
                    aria-label={`${newInvitations} новых приглашений`}
                  >
                    {newInvitations}
                  </span>
                )}
                <ChevronRight size={14} />
              </NavLink>
            ))}
          </nav>
          <div className="sidebar-note">
            <ShieldCheck size={22} />
            <p>
              Навыки говорят
              <br />
              <strong>громче резюме.</strong>
            </p>
            <span>От достижений — к возможностям</span>
          </div>
          <div className="sidebar-bottom">
            <div className="account">
              <Avatar
                name={user.name}
                userId={user.id}
                version={avatar.data?.version}
                size="small"
                company={user.role === "employer"}
              />
              <div>
                <strong>{user.name}</strong>
                <span>
                  {user.role === "employer"
                    ? "Кабинет работодателя"
                    : "Кабинет кандидата"}
                </span>
              </div>
              <button
                className="icon-button"
                title="Выйти"
                aria-label="Выйти"
                onClick={logout}
              >
                <LogOut size={18} />
              </button>
            </div>
          </div>
        </aside>
        <div className="workspace">
          <header className="topbar">
            <span>
              Рабочее пространство <span className="separator">/</span>{" "}
              {links.find((l) => location.pathname.startsWith(l.to))?.label ||
                (location.pathname === "/compare"
                  ? "Сравнение кандидатов"
                  : location.pathname.startsWith("/companies/")
                    ? "Компания"
                    : "Карьера")}
            </span>
            <div>
              <span className="live-dot" />
              {config.data?.demo
                ? "Демонстрационный стенд"
                : "Рабочее пространство"}
              <NavLink to="/invitations" aria-label="Открыть приглашения">
                <ArrowUpRight size={16} />
              </NavLink>
            </div>
          </header>
          <main id="main" tabIndex={-1}>
            <ErrorBox error={error} />
            <Outlet />
          </main>
          <footer className="footer">
            <span>ФСП · Карьера</span>
            <span>Профессиональный путь начинается с навыков</span>
            <span>2026</span>
          </footer>
          {user.role === "employer" && <ComparisonDock />}
        </div>
      </div>
    </ComparisonProvider>
  );
}
function Home() {
  const me = useUser();
  if (me.isPending) return <Loading />;
  return <Navigate to={me.data ? "/overview" : "/login"} replace />;
}
function Role({
  role,
  children,
}: {
  role: User["role"];
  children: React.ReactNode;
}) {
  const me = useUser();
  return me.data?.role === role ? <>{children}</> : <Navigate to="/" replace />;
}
export function App() {
  return (
    <Routes>
      <Route path="/login" element={<Auth />} />
      <Route path="/register" element={<Auth register />} />
      <Route path="/verify" element={<Verify />} />
      <Route path="/reset" element={<ResetPassword />} />
      <Route element={<Layout />}>
        <Route index element={<Home />} />
        <Route path="/overview" element={<Overview />} />
        <Route path="/vacancies" element={<Vacancies />} />
        <Route path="/companies/:id" element={<PublicCompany />} />
        <Route
          path="/vacancies/new"
          element={
            <Role role="employer">
              <VacancyEditor />
            </Role>
          }
        />
        <Route path="/vacancies/:id" element={<VacancyDetail />} />
        <Route
          path="/vacancies/:id/edit"
          element={
            <Role role="employer">
              <VacancyDetail edit />
            </Role>
          }
        />
        <Route path="/applications" element={<Applications />} />
        <Route
          path="/shortlist"
          element={
            <Role role="employer">
              <SearchPage saved />
            </Role>
          }
        />
        <Route
          path="/compare"
          element={
            <Role role="employer">
              <Compare />
            </Role>
          }
        />
        <Route
          path="/profile"
          element={
            <Role role="candidate">
              <CandidateProfile />
            </Role>
          }
        />
        <Route
          path="/assessments"
          element={
            <Role role="candidate">
              <Assessments />
            </Role>
          }
        />
        <Route
          path="/candidates"
          element={
            <Role role="employer">
              <SearchPage />
            </Role>
          }
        />
        <Route
          path="/candidates/:id"
          element={
            <Role role="employer">
              <CandidateDetail />
            </Role>
          }
        />
        <Route path="/invitations" element={<Invitations />} />
        <Route path="/challenges" element={<Challenges />} />
        <Route
          path="/company"
          element={
            <Role role="employer">
              <Company />
            </Role>
          }
        />
        <Route
          path="*"
          element={
            <div className="empty">
              <h1>Страница не найдена</h1>
              <Button asChild>
                <NavLink to="/">В кабинет</NavLink>
              </Button>
            </div>
          }
        />
      </Route>
    </Routes>
  );
}
