import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { ArrowRight, Building2, ExternalLink, MapPin } from "lucide-react";
import {
  api,
  queryClient,
  type CompanyDetails,
  type CompanyProfile,
  workFormats,
} from "../api";
import { Button } from "../components/ui/button";
import { Select } from "../components/ui/select";
import { Avatar, AvatarEditor } from "../components/Avatar";
import { AccountSections, ProfileProgress } from "../components/AccountDetails";
import { CommaInput } from "../components/ProfessionalProfile";
import {
  ErrorBox,
  Field,
  Loading,
  PageTitle,
  Success,
} from "../components/shared";
import { PhoneInput, validatePhone } from "../components/PhoneInput";

const teamSizes: Record<string, string> = {
  "1-10": "1–10 человек",
  "11-50": "11–50 человек",
  "51-200": "51–200 человек",
  "201-500": "201–500 человек",
  "501-1000": "501–1000 человек",
  "1000+": "Более 1000 человек",
};
const emptyDetails: CompanyDetails = {
  tagline: "",
  industry: "",
  legal_name: "",
  inn: "",
  city: "",
  address: "",
  founded_year: 0,
  team_size: "",
  products: "",
  culture: "",
  hiring_process: "",
  technologies: [],
  benefits: [],
  work_formats: [],
  careers_url: "",
  contact_name: "",
  contact_role: "",
  contact_email: "",
  phone: "",
  telegram: "",
};
const detailsData = (v?: Partial<CompanyDetails>): CompanyDetails => ({
  ...emptyDetails,
  ...v,
  technologies: v?.technologies || [],
  benefits: v?.benefits || [],
  work_formats: v?.work_formats || [],
});

function CompanyIdentity({
  company: c,
  hideName = false,
}: {
  company: CompanyProfile;
  hideName?: boolean;
}) {
  const d = detailsData(c.details);
  return (
    <div className="company-identity">
      <Avatar
        name={c.name}
        userId={c.user_id}
        version={c.avatar_version}
        size="portrait"
        company
      />
      <div>
        <span className="eyebrow">Команда и продукт</span>
        {!hideName && <h2>{c.name || "Название вашей компании"}</h2>}
        <p>
          {d.tagline || "Добавьте короткое описание — его увидят кандидаты."}
        </p>
        <div className="identity-meta">
          {d.industry && (
            <span>
              <Building2 size={14} />
              {d.industry}
            </span>
          )}
          {d.city && (
            <span>
              <MapPin size={14} />
              {d.city}
            </span>
          )}
          {d.team_size && <span>{teamSizes[d.team_size]}</span>}
        </div>
      </div>
    </div>
  );
}

export function Company() {
  const q = useQuery({
    queryKey: ["company"],
    queryFn: () => api<CompanyProfile>("/company"),
  });
  const [draft, setDraft] = useState<CompanyProfile>();
  const [error, setError] = useState<unknown>(),
    [saved, setSaved] = useState(false),
    [busy, setBusy] = useState(false);
  const c = draft || q.data;
  const d = detailsData(c?.details);
  const change = <K extends keyof CompanyProfile>(
    key: K,
    value: CompanyProfile[K],
  ) => {
    if (c) {
      setDraft({ ...c, [key]: value });
      setSaved(false);
    }
  };
  const detail = <K extends keyof CompanyDetails>(
    key: K,
    value: CompanyDetails[K],
  ) => change("details", { ...d, [key]: value });
  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!c) return;
    const phoneCheck = validatePhone(d.phone || "");
    if (!phoneCheck.valid) {
      setError(new Error(phoneCheck.error));
      return;
    }
    setBusy(true);
    setSaved(false);
    setError(undefined);
    try {
      await api("/company", "PUT", {
        name: c.name,
        description: c.description,
        website: c.website,
        details: d,
      });
      await Promise.all(
        [
          "company",
          "public-company",
          "vacancies",
          "vacancy",
          "invitations",
          "overview",
        ].map((key) => queryClient.invalidateQueries({ queryKey: [key] })),
      );
      setDraft(undefined);
      setSaved(true);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <PageTitle
        eyebrow="Кабинет работодателя"
        title="Профиль компании"
        action={
          q.data?.name && (
            <Button asChild variant="secondary">
              <Link to={`/companies/${q.data.user_id}`}>
                Вид для кандидата
                <ArrowRight size={16} />
              </Link>
            </Button>
          )
        }
      >
        Помогите кандидатам понять ваш продукт, команду и условия работы.
      </PageTitle>
      {q.data && (
        <details className="account-mobile-navigation">
          <summary>Перейти к разделу компании</summary>
          <AccountSections
            items={[
              ["company-about", "О компании"],
              ["company-product", "Продукт и технологии"],
              ["company-team", "Работа в команде"],
              ["company-contacts", "Контакты"],
              ["company-legal", "Реквизиты"],
            ]}
          />
        </details>
      )}
      <ErrorBox error={q.error || error} />
      {q.isPending ? (
        <Loading />
      ) : (
        c && (
          <div className="profile-layout company-layout">
            <section className="profile-main">
              <CompanyIdentity company={q.data!} />
              <AvatarEditor
                name={q.data!.name}
                userId={q.data!.user_id}
                version={q.data!.avatar_version}
                company
              />
              <form
                className="form-stack"
                onSubmit={submit}
                onChange={() => setSaved(false)}
              >
                <div className="section-heading" id="company-about">
                  <h2>О компании</h2>
                  <span>01 / Основное</span>
                </div>
                <Field label="Название компании">
                  <input
                    required
                    minLength={2}
                    maxLength={150}
                    value={c.name}
                    onChange={(e) => change("name", e.target.value)}
                  />
                </Field>
                <Field
                  label="Коротко о вас"
                  hint="Одно предложение о продукте и его пользе."
                >
                  <input
                    maxLength={160}
                    value={d.tagline}
                    onChange={(e) => detail("tagline", e.target.value)}
                    placeholder="Создаём инструменты для разработчиков и продуктовых команд"
                  />
                </Field>
                <div className="form-grid">
                  <Field label="Отрасль">
                    <input
                      maxLength={100}
                      value={d.industry}
                      onChange={(e) => detail("industry", e.target.value)}
                      placeholder="Информационные технологии"
                    />
                  </Field>
                  <Field label="Город команды">
                    <input
                      maxLength={100}
                      value={d.city}
                      onChange={(e) => detail("city", e.target.value)}
                      placeholder="Москва"
                    />
                  </Field>
                </div>
                <div className="form-grid">
                  <Field label="Размер команды">
                    <Select
                      label="Размер команды"
                      options={teamSizes}
                      emptyLabel="Не указан"
                      value={d.team_size}
                      onValueChange={(v) => detail("team_size", v)}
                    />
                  </Field>
                  <Field label="Год основания">
                    <input
                      type="number"
                      min={1900}
                      max={new Date().getFullYear()}
                      value={d.founded_year || ""}
                      onChange={(e) =>
                        detail("founded_year", Number(e.target.value))
                      }
                    />
                  </Field>
                </div>
                <Field label="Чем занимается компания">
                  <textarea
                    rows={4}
                    maxLength={4000}
                    value={c.description}
                    onChange={(e) => change("description", e.target.value)}
                    placeholder="Для кого вы работаете, какую проблему решаете и чего уже достигли"
                  />
                </Field>
                <div className="form-grid">
                  <Field label="Сайт компании">
                    <input
                      type="url"
                      maxLength={250}
                      value={c.website}
                      onChange={(e) => change("website", e.target.value)}
                      placeholder="https://"
                    />
                  </Field>
                  <Field label="Страница карьеры">
                    <input
                      type="url"
                      maxLength={500}
                      value={d.careers_url}
                      onChange={(e) => detail("careers_url", e.target.value)}
                      placeholder="https://"
                    />
                  </Field>
                </div>
                <div className="section-heading" id="company-product">
                  <h2>Продукт и технологии</h2>
                  <span>02 / Задачи</span>
                </div>
                <Field label="Продукты и направления">
                  <textarea
                    rows={4}
                    maxLength={3000}
                    value={d.products}
                    onChange={(e) => detail("products", e.target.value)}
                    placeholder="Какие продукты развиваете и какие задачи предстоят новым участникам команды"
                  />
                </Field>
                <Field
                  label="Технологии команды"
                  hint="Через запятую. Например: Go, PostgreSQL, React, Kubernetes."
                >
                  <CommaInput
                    value={d.technologies}
                    onChange={(v) => detail("technologies", v)}
                    maxLength={1600}
                  />
                </Field>
                <div className="section-heading" id="company-team">
                  <h2>Работа в команде</h2>
                  <span>03 / Условия</span>
                </div>
                <div className="field">
                  <span>Форматы работы</span>
                  <div
                    className="resume-formats"
                    role="group"
                    aria-label="Форматы работы компании"
                  >
                    {Object.entries(workFormats).map(([key, label]) => (
                      <label className="check" key={key}>
                        <input
                          type="checkbox"
                          checked={d.work_formats.includes(key)}
                          onChange={(e) =>
                            detail(
                              "work_formats",
                              e.target.checked
                                ? [...d.work_formats, key]
                                : d.work_formats.filter((v) => v !== key),
                            )
                          }
                        />
                        {label}
                      </label>
                    ))}
                  </div>
                </div>
                <Field
                  label="Что предлагаете сотрудникам"
                  hint="Через запятую: обучение, ДМС, гибкий график и другие реальные условия."
                >
                  <CommaInput
                    value={d.benefits}
                    onChange={(v) => detail("benefits", v)}
                    maxLength={1500}
                  />
                </Field>
                <Field label="Как устроена работа">
                  <textarea
                    rows={4}
                    maxLength={3000}
                    value={d.culture}
                    onChange={(e) => detail("culture", e.target.value)}
                    placeholder="Расскажите о взаимодействии, процессах, самостоятельности и развитии"
                  />
                </Field>
                <Field label="Процесс найма">
                  <textarea
                    rows={4}
                    maxLength={3000}
                    value={d.hiring_process}
                    onChange={(e) => detail("hiring_process", e.target.value)}
                    placeholder="Этапы знакомства, собеседования и срок обратной связи"
                  />
                </Field>
                <div className="section-heading" id="company-contacts">
                  <h2>Контакты для кандидатов</h2>
                  <span>04 / Связь</span>
                </div>
                <p className="muted resume-editor-hint">
                  Эти контакты видны в профиле компании. Укажите рабочие данные,
                  которыми готовы поделиться с кандидатами.
                </p>
                <div className="form-grid">
                  <Field label="Контактное лицо">
                    <input
                      maxLength={100}
                      value={d.contact_name}
                      onChange={(e) => detail("contact_name", e.target.value)}
                    />
                  </Field>
                  <Field label="Должность">
                    <input
                      maxLength={100}
                      value={d.contact_role}
                      onChange={(e) => detail("contact_role", e.target.value)}
                      placeholder="Рекрутер команды"
                    />
                  </Field>
                </div>
                <div className="form-grid">
                  <Field label="Рабочая почта">
                    <input
                      type="email"
                      maxLength={254}
                      value={d.contact_email}
                      onChange={(e) => detail("contact_email", e.target.value)}
                    />
                  </Field>
                  <Field label="Телефон">
                    <PhoneInput
                      value={d.phone}
                      onChange={(val) => detail("phone", val)}
                      placeholder="+7 (900) 000-00-00"
                    />
                  </Field>
                </div>
                <Field label="Telegram" hint="Имя пользователя, без ссылки.">
                  <input
                    maxLength={32}
                    value={d.telegram}
                    onChange={(e) => detail("telegram", e.target.value)}
                    placeholder="@username"
                  />
                </Field>
                <div className="section-heading" id="company-legal">
                  <h2>Реквизиты</h2>
                  <span>05 / Сведения</span>
                </div>
                <p className="muted resume-editor-hint">
                  Необязательные сведения. Их заполнение не означает проверку
                  компании платформой.
                </p>
                <div className="form-grid">
                  <Field label="Юридическое название">
                    <input
                      maxLength={200}
                      value={d.legal_name}
                      onChange={(e) => detail("legal_name", e.target.value)}
                      placeholder="ООО «Компания»"
                    />
                  </Field>
                  <Field label="ИНН">
                    <input
                      inputMode="numeric"
                      maxLength={12}
                      value={d.inn}
                      onChange={(e) => detail("inn", e.target.value)}
                      pattern="([0-9]{10}|[0-9]{12})"
                    />
                  </Field>
                </div>
                <Field label="Адрес офиса">
                  <input
                    maxLength={300}
                    value={d.address}
                    onChange={(e) => detail("address", e.target.value)}
                  />
                </Field>
                <div className="profile-save-bar">
                  <div>
                    {saved ? (
                      <Success>Профиль компании сохранён</Success>
                    ) : (
                      <span className="muted">
                        Сведения будут доступны кандидатам после сохранения.
                      </span>
                    )}
                  </div>
                  <Button disabled={busy}>
                    {busy ? "Сохраняем…" : "Сохранить изменения"}
                  </Button>
                </div>
                <ErrorBox error={error} />
              </form>
            </section>
            <aside className="profile-aside">
              <AccountSections
                items={[
                  ["company-about", "О компании"],
                  ["company-product", "Продукт и технологии"],
                  ["company-team", "Работа в команде"],
                  ["company-contacts", "Контакты"],
                  ["company-legal", "Реквизиты"],
                ]}
              />
              <ProfileProgress
                items={[
                  { label: "Логотип", done: q.data!.avatar_version > 0 },
                  {
                    label: "Описание",
                    done: !!q.data!.description && !!q.data!.details?.tagline,
                  },
                  {
                    label: "Отрасль и команда",
                    done:
                      !!q.data!.details?.industry &&
                      !!q.data!.details?.team_size,
                  },
                  {
                    label: "Продукт и стек",
                    done:
                      !!q.data!.details?.products &&
                      !!q.data!.details?.technologies?.length,
                  },
                  {
                    label: "Условия работы",
                    done: !!q.data!.details?.work_formats?.length,
                  },
                  {
                    label: "Контакт для связи",
                    done:
                      !!q.data!.details?.contact_email ||
                      !!q.data!.details?.telegram,
                  },
                ]}
              />
              <div className="company-profile-note">
                <h3>Первое знакомство с командой</h3>
                <p>
                  Кандидат сможет открыть этот профиль из вакансии или
                  приглашения и узнать о вас до ответа на предложение.
                </p>
                <Link className="text-link" to={`/companies/${c.user_id}`}>
                  Открыть профиль
                  <ArrowRight size={16} />
                </Link>
              </div>
            </aside>
          </div>
        )
      )}
    </>
  );
}

const external = (href: string, label: string) => (
  <a
    className="text-link"
    href={href}
    target="_blank"
    rel="noopener noreferrer"
  >
    {label}
    <ExternalLink size={15} />
  </a>
);
export function PublicCompany() {
  const { id } = useParams();
  const q = useQuery({
    queryKey: ["public-company", id],
    queryFn: () => api<CompanyProfile>(`/companies/${id}`),
  });
  const c = q.data,
    d = detailsData(c?.details);
  return (
    <>
      <PageTitle
        eyebrow="Компания на платформе"
        title={c?.name || "Профиль компании"}
      >
        Познакомьтесь с командой, продуктом и условиями работы.
      </PageTitle>
      <ErrorBox error={q.error} />
      {q.isPending ? (
        <Loading />
      ) : (
        c && (
          <div className="profile-layout company-public-layout">
            <section className="profile-main">
              <CompanyIdentity company={c} hideName />
              <section className="company-copy">
                <h2>О компании</h2>
                <p className="long-copy">
                  {c.description || "Компания пока не добавила описание."}
                </p>
                <div className="resume-links">
                  {c.website && external(c.website, "Сайт компании")}
                  {d.careers_url &&
                    external(d.careers_url, "Карьера в компании")}
                </div>
              </section>
              {d.products && (
                <section className="company-copy">
                  <h2>Продукты и направления</h2>
                  <p className="long-copy">{d.products}</p>
                </section>
              )}
              {!!d.technologies.length && (
                <section className="company-copy">
                  <h2>Технологии команды</h2>
                  <div className="tags">
                    {d.technologies.map((t) => (
                      <span className="skill" key={t}>
                        {t}
                      </span>
                    ))}
                  </div>
                </section>
              )}
              {!!d.benefits.length && (
                <section className="company-copy">
                  <h2>Условия и возможности</h2>
                  <ul className="company-benefits">
                    {d.benefits.map((b) => (
                      <li key={b}>{b}</li>
                    ))}
                  </ul>
                </section>
              )}
              {d.culture && (
                <section className="company-copy">
                  <h2>Как устроена работа</h2>
                  <p className="long-copy">{d.culture}</p>
                </section>
              )}
              {d.hiring_process && (
                <section className="company-copy">
                  <h2>Процесс найма</h2>
                  <p className="long-copy">{d.hiring_process}</p>
                </section>
              )}
              {(d.legal_name || d.inn) && (
                <section className="company-copy">
                  <h2>Реквизиты</h2>
                  <dl className="resume-preferences">
                    {d.legal_name && (
                      <div>
                        <dt>Юридическое название</dt>
                        <dd>{d.legal_name}</dd>
                      </div>
                    )}
                    {d.inn && (
                      <div>
                        <dt>ИНН</dt>
                        <dd>{d.inn}</dd>
                      </div>
                    )}
                  </dl>
                  <small>
                    Сведения предоставлены компанией. Проверка платформой не
                    проводилась.
                  </small>
                </section>
              )}
            </section>
            <aside className="profile-aside">
              <section className="company-facts">
                <h3>О команде</h3>
                <dl className="resume-preferences">
                  {d.industry && (
                    <div>
                      <dt>Отрасль</dt>
                      <dd>{d.industry}</dd>
                    </div>
                  )}
                  {d.team_size && (
                    <div>
                      <dt>Размер</dt>
                      <dd>{teamSizes[d.team_size]}</dd>
                    </div>
                  )}
                  {d.founded_year > 0 && (
                    <div>
                      <dt>Основана</dt>
                      <dd>{d.founded_year}</dd>
                    </div>
                  )}
                  {d.city && (
                    <div>
                      <dt>Город</dt>
                      <dd>{d.city}</dd>
                    </div>
                  )}
                  {!!d.work_formats.length && (
                    <div>
                      <dt>Формат работы</dt>
                      <dd>
                        {d.work_formats.map((f) => workFormats[f]).join(", ")}
                      </dd>
                    </div>
                  )}
                  {d.address && (
                    <div>
                      <dt>Адрес</dt>
                      <dd>{d.address}</dd>
                    </div>
                  )}
                </dl>
                {!d.industry && !d.team_size && !d.city && (
                  <p className="muted">Команда ещё дополняет свой профиль.</p>
                )}
              </section>
              {(d.contact_name || d.contact_email || d.phone || d.telegram) && (
                <section className="company-facts">
                  <h3>Связаться с командой</h3>
                  {d.contact_name && <strong>{d.contact_name}</strong>}
                  {d.contact_role && <p className="muted">{d.contact_role}</p>}
                  <div className="company-contact-links">
                    {d.contact_email && (
                      <a
                        className="text-link"
                        href={`mailto:${d.contact_email}`}
                      >
                        {d.contact_email}
                      </a>
                    )}
                    {d.phone && (
                      <a
                        className="text-link"
                        href={`tel:${d.phone.replace(/[^+0-9]/g, "")}`}
                      >
                        {d.phone}
                      </a>
                    )}
                    {d.telegram &&
                      external(`https://t.me/${d.telegram}`, `@${d.telegram}`)}
                  </div>
                </section>
              )}
            </aside>
          </div>
        )
      )}
    </>
  );
}
