import { QueryClient } from "@tanstack/react-query";
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false, staleTime: 15000, refetchOnWindowFocus: true },
  },
});
let csrf = "";
export function setCSRF(value: string) {
  csrf = value;
}
export class APIError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  let res: Response;
  try {
    res = await fetch("/api" + path, {
      method,
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/json",
        "X-Requested-With": "fsp",
        ...(csrf ? { "X-CSRF-Token": csrf } : {}),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new APIError(
      "Нет соединения с сервером. Проверьте подключение и повторите действие.",
      0,
    );
  }
  if (!res.ok) {
    const raw = await res.text();
    let message = /^\s*</.test(raw)
      ? "Сервис временно недоступен. Повторите действие через несколько секунд."
      : raw;
    try {
      const data = JSON.parse(raw);
      message = data.detail || "Проверьте заполнение полей";
      if (res.status === 422) {
        const labels: Record<string, string> = {
          email: "Электронная почта",
          password: "Пароль",
          name: "Имя",
          title: "Название",
          message: "Описание предложения",
          salary_min: "Зарплата от",
          salary_max: "Зарплата до",
          answer: "Решение",
          body: "Условие задачи",
          feedback: "Обратная связь",
          description: "Задачи и описание",
          cover_letter: "Сопроводительное письмо",
          specialization: "Направление",
          grade: "Уровень",
          work_format: "Формат работы",
          skills: "Навыки",
          city: "Город",
          consent: "Согласие на контакт",
          phone: "Телефон",
          telegram: "Telegram",
          image: "Изображение",
          institution: "Учебное заведение",
          degree: "Квалификация",
          field: "Направление обучения",
          start_year: "Год начала",
          end_year: "Год окончания",
          issuer: "Организация",
          year: "Год",
          timezone: "Часовой пояс",
          domains: "Отраслевой опыт",
          interests: "Интересующие задачи",
          contact_email: "Рабочая почта",
          founded_year: "Год основания",
          inn: "ИНН",
          estimated_minutes: "Время решения",
        };
        const fields = [
          ...new Set(
            (data.errors || [])
              .map(
                (e: { location?: string }) =>
                  labels[e.location?.split(".").at(-1) || ""] || e.location,
              )
              .filter(Boolean),
          ),
        ];
        message = fields.length
          ? `Проверьте заполнение полей: ${fields.join(", ")}.`
          : "Проверьте заполнение полей и ограничения рядом с ними.";
      }
    } catch {
      /* Non-JSON proxy error. */
    }
    if (res.status === 401 && csrf && path !== "/auth/login") {
      setCSRF("");
      window.dispatchEvent(new Event("fsp:session-expired"));
    }
    throw new APIError(message || "Сервис временно недоступен", res.status);
  }
  return res.json();
}
export const specs: Record<string, string> = {
  backend: "Backend-разработка",
  frontend: "Frontend-разработка",
  analytics: "Аналитика данных",
};
export const grades: Record<string, string> = {
  junior: "Junior",
  middle: "Middle",
  senior: "Senior",
};
export const date = (v: string) => new Date(v).toLocaleDateString("ru-RU");
export const money = (v: number) => new Intl.NumberFormat("ru-RU").format(v);
export const initials = (v: string) =>
  (v.match(/[\p{L}\p{N}]+/gu) || [])
    .slice(0, 2)
    .map((v) => v[0])
    .join("")
    .toUpperCase();
export function plural(n: number, words: [string, string, string]) {
  const last = n % 10,
    hundred = n % 100;
  return hundred >= 11 && hundred <= 14
    ? words[2]
    : last === 1
      ? words[0]
      : last >= 2 && last <= 4
        ? words[1]
        : words[2];
}
export interface User {
  id: string;
  name: string;
  email: string;
  role: "candidate" | "employer";
  verified: boolean;
  csrf: string;
}
export interface Achievement {
  title: string;
  year: number;
  place: number;
  source: string;
}
export interface Profile {
  avatar_version: number;
  contacts?: Contacts;
  user_id: string;
  name: string;
  headline: string;
  city: string;
  about: string;
  skills: string[];
  experience_years: number;
  specialization: string;
  grade: string;
  test_score: number;
  published: boolean;
  fsp_id: string;
  achievements: Achievement[];
  next_grade_change_at: string | null;
  last_practice_at: string | null;
  professional: ProfessionalProfile;
}
export interface WorkExperience {
  company: string;
  role: string;
  start: string;
  end: string;
  current: boolean;
  description: string;
  results: string;
}
export interface PortfolioProject {
  name: string;
  role: string;
  description: string;
  results: string;
  stack: string[];
  repository: string;
  demo: string;
}
export interface ProfessionalProfile {
  timezone: string;
  domains: string[];
  interests: string;
  education: Education[];
  certificates: Certificate[];
  github: string;
  portfolio: string;
  salary_min: number;
  salary_max: number;
  work_formats: string[];
  employment: string;
  availability: string;
  search_status: string;
  relocation: boolean;
  languages: string[];
  experience: WorkExperience[];
  projects: PortfolioProject[];
}
export interface Contacts {
  phone: string;
  telegram: string;
}
export interface Education {
  institution: string;
  degree: string;
  field: string;
  start_year: number;
  end_year: number;
  current: boolean;
}
export interface Certificate {
  title: string;
  issuer: string;
  year: number;
  url: string;
}
export interface CompanyDetails {
  tagline: string;
  industry: string;
  legal_name: string;
  inn: string;
  city: string;
  address: string;
  founded_year: number;
  team_size: string;
  products: string;
  culture: string;
  hiring_process: string;
  technologies: string[];
  benefits: string[];
  work_formats: string[];
  careers_url: string;
  contact_name: string;
  contact_role: string;
  contact_email: string;
  phone: string;
  telegram: string;
}
export interface CompanyProfile {
  user_id: string;
  name: string;
  description: string;
  website: string;
  avatar_version: number;
  details: CompanyDetails;
}
export const employmentTypes: Record<string, string> = {
  full_time: "Полная занятость",
  part_time: "Частичная занятость",
  contract: "Проектная работа",
  internship: "Стажировка",
};
export const availabilityTypes: Record<string, string> = {
  immediately: "Готов выйти сейчас",
  two_weeks: "Через две недели",
  month: "Через месяц",
  negotiable: "По договорённости",
};
export const searchStatuses: Record<string, string> = {
  active: "Активно ищу работу",
  open: "Рассматриваю предложения",
  paused: "Пока не ищу работу",
};
export interface Candidate extends Profile {
  match_score: number;
  reasons: string[];
  email?: string;
  match: Match;
}
export interface Match {
  total: number;
  test: number;
  stack: number;
  fsp: number;
  test_max: number;
  stack_max: number;
  fsp_max: number;
  matched: string[];
  missing: string[];
  reasons: string[];
}
export interface QualificationEvidence {
  source: "assessment" | "demo" | "imported" | "unconfirmed";
  specialization: string;
  grade: string;
  score: number;
  confirmed_at: string | null;
  version: string;
  topics: { topic: string; passed: boolean }[];
  next_grade_change_at: string | null;
  last_practice_at: string | null;
}
export interface Vacancy {
  company_id: string;
  company_avatar_version: number;
  id: string;
  title: string;
  description: string;
  company_name: string;
  company_description: string;
  company_website: string;
  specialization: string;
  grade: string;
  skills: string[];
  work_format: "remote" | "hybrid" | "office";
  city: string;
  salary_min: number;
  salary_max: number;
  status: "draft" | "published" | "closed";
  created_at: string;
  application_count: number;
  my_application: Application | null;
}
export interface Application {
  id: string;
  vacancy_id: string;
  candidate_id: string;
  candidate_name: string;
  candidate_email?: string;
  specialization: string;
  grade: string;
  test_score: number;
  vacancy_title: string;
  company_name: string;
  salary_min: number;
  salary_max: number;
  cover_letter: string;
  status: "new" | "viewed" | "shortlisted" | "rejected" | "withdrawn";
  created_at: string;
  updated_at: string;
}
export const workFormats: Record<string, string> = {
  remote: "Удалённо",
  hybrid: "Гибрид",
  office: "В офисе",
};
export const applicationStatuses: Record<string, string> = {
  new: "Отправлен",
  viewed: "Рассмотрен",
  shortlisted: "В подборке",
  rejected: "Не подошёл",
  withdrawn: "Отозван",
};
export interface Question {
  id: string;
  prompt: string;
  topic: string;
  kind: "number" | "choice";
  options: string[];
}
export interface Attempt {
  id: string;
  specialization: string;
  grade: string;
  questions: Question[];
  started_at: string;
  expires_at: string;
  submitted_at: string | null;
  score: number | null;
  passed: boolean | null;
  template_version: string;
  topics: Record<string, boolean>;
}
export interface Invitation {
  company_id: string;
  company_avatar_version: number;
  id: string;
  candidate_id: string;
  candidate_name: string;
  company_name: string;
  company_email: string;
  candidate_email?: string;
  title: string;
  message: string;
  salary_min: number;
  salary_max: number;
  status: "sent" | "viewed" | "accepted" | "declined";
  created_at: string;
}
export interface Config {
  demo: boolean;
  grade_cooldown_days: number;
  test_minutes: number;
  test_questions: number;
  test_pass_score: number;
  test_version: string;
}

export interface ChallengeSubmission {
  id: string;
  candidate_id: string;
  candidate_name: string;
  answer: string;
  status: "submitted" | "accepted" | "needs_work";
  feedback: string;
  submitted_at: string;
  reviewed_at: string | null;
}
export interface Challenge {
  id: string;
  title: string;
  body: string;
  company_name: string;
  specialization: string;
  grade: string;
  estimated_minutes: number;
  created_at: string;
  closes_at: string;
  submission_count: number;
  submission: ChallengeSubmission | null;
}
export interface ChallengeFeed {
  challenges: Challenge[];
  next_submission_at: string | null;
}
