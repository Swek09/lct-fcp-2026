import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  ArrowRight,
  Download,
  Eye,
  EyeOff,
  ShieldCheck,
  Trophy,
} from "lucide-react";
import {
  api,
  type Config,
  date,
  grades,
  type Profile,
  type ProfessionalProfile,
  queryClient,
  specs,
} from "../api";
import { Button } from "../components/ui/button";
import { Avatar, AvatarEditor } from "../components/Avatar";
import {
  AccountSections,
  EducationEditor,
  PersonalDetailsEditor,
  ProfileProgress,
} from "../components/AccountDetails";
import { EvidencePassport } from "../components/Evidence";
import {
  ProfessionalEditor,
  professionalData,
} from "../components/ProfessionalProfile";
import {
  ErrorBox,
  Field,
  Loading,
  PageTitle,
  Success,
} from "../components/shared";
import { PhoneInput, validatePhone } from "../components/PhoneInput";
export function CandidateProfile() {
  const q = useQuery({
    queryKey: ["profile"],
    queryFn: () => api<Profile>("/profile"),
  });
  const cfg = useQuery({
    queryKey: ["config"],
    queryFn: () => api<Config>("/config"),
  });
  const [error, setError] = useState<unknown>();
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [professional, setProfessional] = useState<ProfessionalProfile>();
  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setBusy(true);
    setMessage("");
    setError(undefined);
    const f = Object.fromEntries(new FormData(e.currentTarget));
    const phone = String(f.phone || "").trim();
    const phoneCheck = validatePhone(phone);
    if (!phoneCheck.valid) {
      setError(new Error(phoneCheck.error));
      setBusy(false);
      return;
    }
    try {
      await api("/profile", "PUT", {
        name: String(f.name || ""),
        headline: String(f.headline || ""),
        city: String(f.city || ""),
        about: String(f.about || ""),
        skills: String(f.skills)
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean),
        experience_years: Number(f.experience_years),
        published: f.published === "on",
        professional: professionalData(professional || q.data?.professional),
        contacts: {
          phone,
          telegram: String(f.telegram || ""),
        },
      });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["profile"] }),
        queryClient.invalidateQueries({ queryKey: ["me"] }),
      ]);
      setMessage("Изменения сохранены");
      setProfessional(undefined);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  const fsp = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    try {
      await api(
        "/profile/fsp",
        "POST",
        Object.fromEntries(new FormData(e.currentTarget)),
      );
      await queryClient.invalidateQueries({ queryKey: ["profile"] });
      setMessage("Достижения обновлены");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <PageTitle
        eyebrow="Личный кабинет"
        title="Ваш профиль"
        action={
          <Button asChild variant="secondary">
            <a href="/api/profile/pdf" download>
              <Download size={16} />
              PDF-профиль
            </a>
          </Button>
        }
      >
        Покажите, что вы умеете. Следующий шаг сделает работодатель.
      </PageTitle>
      {q.data && (
        <details className="account-mobile-navigation">
          <summary>Перейти к разделу профиля</summary>
          <AccountSections
            items={[
              ["about", "О вас"],
              ["preferences", "Пожелания к работе"],
              ["portfolio", "Портфолио"],
              ["experience", "Опыт и проекты"],
              ["education", "Образование"],
              ["certificates", "Сертификаты"],
              ["contacts", "Контакты"],
              ["visibility", "Видимость"],
            ]}
          />
        </details>
      )}
      <ErrorBox error={q.error || error} />
      {message && <Success>{message}</Success>}
      {q.isPending ? (
        <Loading />
      ) : (
        q.data && (
          <div className="profile-layout">
            <section className="profile-main">
              <div className="profile-summary">
                <Avatar
                  name={q.data.name}
                  userId={q.data.user_id}
                  version={q.data.avatar_version}
                  size="portrait"
                />
                <div>
                  <h2>{q.data.name}</h2>
                  <p>
                    {q.data.headline || "Добавьте профессиональный заголовок"}
                  </p>
                  <span className="muted">
                    {q.data.city || "Город не указан"}
                  </span>
                </div>
              </div>
              <AvatarEditor
                name={q.data.name}
                userId={q.data.user_id}
                version={q.data.avatar_version}
              />
              <EvidencePassport />
              <form
                className="form-stack"
                onSubmit={submit}
                onChange={() => setMessage("")}
              >
                <div className="section-heading" id="about">
                  <h2>О вас</h2>
                  <span>01 / Профиль</span>
                </div>
                <div className="form-grid">
                  <Field label="Имя и фамилия">
                    <input
                      name="name"
                      defaultValue={q.data.name}
                      required
                      minLength={2}
                      maxLength={100}
                    />
                  </Field>
                  <Field label="Город">
                    <input
                      name="city"
                      defaultValue={q.data.city}
                      maxLength={100}
                      placeholder="Например, Москва"
                    />
                  </Field>
                </div>
                <Field label="Профессиональный заголовок">
                  <input
                    name="headline"
                    defaultValue={q.data.headline}
                    maxLength={150}
                    placeholder="Backend-разработчик · Go"
                  />
                </Field>
                <Field
                  label="Навыки"
                  hint="Через запятую. Эти сведения видны как ваше самоописание; категорию подтверждает тест."
                >
                  <input
                    name="skills"
                    defaultValue={q.data.skills.join(", ")}
                    placeholder="Go, PostgreSQL, Redis"
                  />
                </Field>
                <Field label="Опыт в годах">
                  <input
                    className="short-input"
                    name="experience_years"
                    type="number"
                    min={0}
                    max={60}
                    defaultValue={q.data.experience_years}
                  />
                </Field>
                <Field label="О себе">
                  <textarea
                    name="about"
                    rows={5}
                    defaultValue={q.data.about}
                    maxLength={4000}
                    placeholder="Над чем работали и какие задачи интересны сейчас?"
                  />
                </Field>
                <PersonalDetailsEditor
                  value={professionalData(professional || q.data.professional)}
                  onChange={(value) => {
                    setMessage("");
                    setProfessional(value);
                  }}
                />
                <ProfessionalEditor
                  value={professionalData(professional || q.data.professional)}
                  onChange={(value) => {
                    setMessage("");
                    setProfessional(value);
                  }}
                />
                <EducationEditor
                  value={professionalData(professional || q.data.professional)}
                  onChange={(value) => {
                    setMessage("");
                    setProfessional(value);
                  }}
                />
                <div className="section-heading" id="contacts">
                  <h2>Контакты</h2>
                  <span>08 / Связь</span>
                </div>
                <p className="muted resume-editor-hint">
                  Телефон и Telegram откроются только компании, чьё приглашение
                  вы примете или на чью вакансию откликнетесь с согласием на
                  контакт.
                </p>
                <div className="form-grid">
                  <Field label="Телефон">
                    <PhoneInput
                      name="phone"
                      defaultValue={q.data.contacts?.phone || ""}
                      placeholder="+7 (900) 000-00-00"
                    />
                  </Field>
                  <Field label="Telegram" hint="Имя пользователя, без ссылки.">
                    <input
                      name="telegram"
                      defaultValue={q.data.contacts?.telegram || ""}
                      maxLength={32}
                      placeholder="@username"
                    />
                  </Field>
                </div>
                <div className="section-heading" id="visibility">
                  <h2>Видимость</h2>
                  <span>09 / Приватность</span>
                </div>
                <label className="check publication">
                  <input
                    type="checkbox"
                    name="published"
                    defaultChecked={q.data.published}
                  />
                  <span>
                    <strong>Разрешаю публикацию профиля работодателям</strong>
                    <small>
                      Имя, резюме и результаты будут доступны после
                      подтверждения уровня. Почта откроется компании, чьё
                      приглашение вы примете или на чью вакансию откликнетесь с
                      согласием.
                    </small>
                  </span>
                </label>
                <div className="profile-save">
                  <span>
                    {message || "Сохраните изменения перед уходом со страницы"}
                  </span>
                  <Button disabled={busy}>
                    {busy ? "Сохраняем…" : "Сохранить профиль"}
                  </Button>
                </div>
              </form>
            </section>
            <aside className="profile-aside">
              <AccountSections
                items={[
                  ["about", "О вас"],
                  ["preferences", "Пожелания к работе"],
                  ["portfolio", "Портфолио"],
                  ["experience", "Опыт и проекты"],
                  ["education", "Образование"],
                  ["certificates", "Сертификаты"],
                  ["contacts", "Контакты"],
                  ["visibility", "Видимость"],
                ]}
              />
              <ProfileProgress
                items={[
                  { label: "Фото профиля", done: q.data.avatar_version > 0 },
                  {
                    label: "О себе",
                    done: !!q.data.about && !!q.data.headline,
                  },
                  {
                    label: "Опыт или проекты",
                    done:
                      !!q.data.professional?.experience?.length ||
                      !!q.data.professional?.projects?.length,
                  },
                  {
                    label: "Условия работы",
                    done: !!q.data.professional?.work_formats?.length,
                  },
                  {
                    label: "Образование или курсы",
                    done:
                      !!q.data.professional?.education?.length ||
                      !!q.data.professional?.certificates?.length,
                  },
                  {
                    label: "Контакты",
                    done:
                      !!q.data.contacts?.phone || !!q.data.contacts?.telegram,
                  },
                ]}
              />
              <div className="panel verification">
                <ShieldCheck size={26} />
                <span className="eyebrow">Подтверждённый уровень</span>
                <h2>
                  {q.data.grade ? grades[q.data.grade] : "Пока не определён"}
                </h2>
                <p>
                  {q.data.specialization
                    ? specs[q.data.specialization]
                    : "Пройдите тест, чтобы получить категорию и появиться в поиске."}
                </p>
                {q.data.grade && (
                  <div className="score-line">
                    <span>Результат тестирования</span>
                    <strong>
                      {q.data.test_score}
                      <small>/100</small>
                    </strong>
                  </div>
                )}
                <Button
                  asChild
                  variant={q.data.grade ? "secondary" : "default"}
                >
                  <Link to="/assessments">
                    {q.data.grade ? "Результаты и тесты" : "Подтвердить навыки"}
                    <ArrowRight size={16} />
                  </Link>
                </Button>
                {q.data.next_grade_change_at && (
                  <small>
                    Следующая смена уровня — {date(q.data.next_grade_change_at)}
                  </small>
                )}
              </div>
              <div className="visibility-state">
                {q.data.published && q.data.grade ? (
                  <Eye size={19} />
                ) : (
                  <EyeOff size={19} />
                )}
                <p>
                  {q.data.published && q.data.grade
                    ? "Работодатели могут найти ваш профиль"
                    : q.data.published
                      ? "Профиль появится в поиске после теста"
                      : "Профиль скрыт из поиска"}
                </p>
              </div>
              <div className="panel practice-summary">
                <h3>Практика навыков</h3>
                <p className="muted">
                  {q.data.last_practice_at
                    ? `Последнее решение подтверждено работодателем ${date(q.data.last_practice_at)}.`
                    : "Решайте короткие задачи от команд и получайте обратную связь."}
                </p>
                <Link className="text-link" to="/challenges">
                  Короткие задания
                  <ArrowRight size={16} />
                </Link>
              </div>
              <div className="panel">
                <div className="inline-heading">
                  <Trophy size={20} />
                  <h3>Достижения ФСП</h3>
                </div>
                {q.data.achievements.length ? (
                  q.data.achievements.map((a, i) => (
                    <div className="achievement" key={i}>
                      <span>{String(a.place).padStart(2, "0")}</span>
                      <div>
                        <strong>{a.title}</strong>
                        <small>
                          {a.year} · {a.place}-е место · демо
                        </small>
                      </div>
                    </div>
                  ))
                ) : (
                  <p className="muted">
                    Пока без истории ФСП. Вы можете подтвердить навыки тестом и
                    получать предложения.
                  </p>
                )}
                {cfg.data?.demo ? (
                  <form className="form-stack" onSubmit={fsp}>
                    <Field
                      label="Демонстрационный ФСП ID"
                      hint="DEMO-1001 или DEMO-1002. Пустое поле удалит связь."
                    >
                      <input
                        name="id"
                        defaultValue={q.data.fsp_id}
                        placeholder="DEMO-1001"
                      />
                    </Field>
                    <Button variant="secondary" size="sm" disabled={busy}>
                      Обновить связь
                    </Button>
                    <small>
                      Реестр ФСП ещё не подключён. Здесь используются тестовые
                      достижения.
                    </small>
                  </form>
                ) : (
                  <p className="muted">Подключение реестра ФСП готовится.</p>
                )}
              </div>
            </aside>
          </div>
        )
      )}
    </>
  );
}
