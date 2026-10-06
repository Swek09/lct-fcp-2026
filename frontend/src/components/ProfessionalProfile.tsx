import { useEffect, useState } from "react";
import { ExternalLink, Plus, Trash2 } from "lucide-react";
import {
  type ProfessionalProfile,
  type WorkExperience,
  type PortfolioProject,
  type Profile,
  employmentTypes,
  availabilityTypes,
  searchStatuses,
  workFormats,
  money,
} from "../api";
import { Field } from "./shared";
import { Button } from "./ui/button";
import { Select } from "./ui/select";

const empty: ProfessionalProfile = {
  timezone: "",
  domains: [],
  interests: "",
  education: [],
  certificates: [],
  github: "",
  portfolio: "",
  salary_min: 0,
  salary_max: 0,
  work_formats: [],
  employment: "",
  availability: "",
  search_status: "",
  relocation: false,
  languages: [],
  experience: [],
  projects: [],
};
export const professionalData = (
  value?: Partial<ProfessionalProfile>,
): ProfessionalProfile => ({
  ...empty,
  ...value,
  work_formats: value?.work_formats || [],
  languages: value?.languages || [],
  experience: value?.experience || [],
  projects: value?.projects || [],
  domains: value?.domains || [],
  education: value?.education || [],
  certificates: value?.certificates || [],
});
const list = (value: string) =>
  value
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
export const salaryExpectation = (value?: Partial<ProfessionalProfile>) => {
  const d = professionalData(value);
  return d.salary_min && d.salary_max
    ? `${money(d.salary_min)}–${money(d.salary_max)} ₽`
    : d.salary_min
      ? `От ${money(d.salary_min)} ₽`
      : d.salary_max
        ? `До ${money(d.salary_max)} ₽`
        : "Не указаны";
};
const monthLabel = (value: string) =>
  value
    ? new Date(`${value}-01T12:00:00`).toLocaleDateString("ru-RU", {
        month: "short",
        year: "numeric",
      })
    : "";
export function CommaInput({
  value,
  onChange,
  maxLength = 500,
}: {
  value: string[];
  onChange: (value: string[]) => void;
  maxLength?: number;
}) {
  const [raw, setRaw] = useState(value.join(", "));
  const key = value.join(", ");
  useEffect(() => {
    if (list(raw).join(", ") !== key) setRaw(key);
  }, [key]);
  return (
    <input
      value={raw}
      maxLength={maxLength}
      onChange={(e) => {
        setRaw(e.target.value);
        onChange(list(e.target.value));
      }}
    />
  );
}
export function ProfessionalEditor({
  value,
  onChange,
}: {
  value: ProfessionalProfile;
  onChange: (value: ProfessionalProfile) => void;
}) {
  const d = professionalData(value);
  const set = <K extends keyof ProfessionalProfile>(
    key: K,
    value: ProfessionalProfile[K],
  ) => onChange({ ...d, [key]: value });
  const work = (index: number, change: Partial<WorkExperience>) =>
    set(
      "experience",
      d.experience.map((item, i) =>
        i === index ? { ...item, ...change } : item,
      ),
    );
  const project = (index: number, change: Partial<PortfolioProject>) =>
    set(
      "projects",
      d.projects.map((item, i) =>
        i === index ? { ...item, ...change } : item,
      ),
    );
  return (
    <>
      <div className="section-heading" id="preferences">
        <h2>Пожелания к работе</h2>
        <span>02 / Условия</span>
      </div>
      <div className="form-grid">
        <Field label="Поиск работы">
          <Select
            label="Поиск работы"
            value={d.search_status}
            options={searchStatuses}
            emptyLabel="Не указано"
            onValueChange={(v) => set("search_status", v)}
          />
        </Field>
        <Field label="Когда готовы выйти">
          <Select
            label="Когда готовы выйти"
            value={d.availability}
            options={availabilityTypes}
            emptyLabel="Не указано"
            onValueChange={(v) => set("availability", v)}
          />
        </Field>
        <Field label="Занятость">
          <Select
            label="Занятость"
            value={d.employment}
            options={employmentTypes}
            emptyLabel="Не указана"
            onValueChange={(v) => set("employment", v)}
          />
        </Field>
        <Field label="Языки" hint="Например: русский — родной, английский — B2">
          <CommaInput
            value={d.languages}
            onChange={(v) => set("languages", v)}
          />
        </Field>
      </div>
      <div className="field">
        <span>Формат работы</span>
        <div className="resume-formats" role="group" aria-label="Формат работы">
          {Object.entries(workFormats).map(([key, label]) => (
            <label className="check" key={key}>
              <input
                type="checkbox"
                checked={d.work_formats.includes(key)}
                onChange={(e) =>
                  set(
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
      <div className="form-grid">
        <Field label="Ожидаемая зарплата от, ₽">
          <input
            type="number"
            min={0}
            max={10000000}
            value={d.salary_min || ""}
            onChange={(e) => set("salary_min", Number(e.target.value))}
            placeholder="Не указана"
          />
        </Field>
        <Field label="До, ₽">
          <input
            type="number"
            min={d.salary_min || 0}
            max={10000000}
            value={d.salary_max || ""}
            onChange={(e) => set("salary_max", Number(e.target.value))}
            placeholder="Не указана"
          />
        </Field>
      </div>
      <small>
        В месяц, до вычета налогов. Можно указать только нижнюю или верхнюю
        границу.
      </small>
      <label className="check">
        <input
          type="checkbox"
          checked={d.relocation}
          onChange={(e) => set("relocation", e.target.checked)}
        />
        Готов к переезду
      </label>
      <div className="section-heading">
        <h2 id="portfolio">Ссылки на работы</h2>
        <span>03 / Портфолио</span>
      </div>
      <div className="form-grid">
        <Field label="GitHub / GitLab">
          <input
            type="url"
            value={d.github}
            onChange={(e) => set("github", e.target.value)}
            maxLength={500}
            placeholder="https://github.com/username"
          />
        </Field>
        <Field label="Сайт или портфолио">
          <input
            type="url"
            value={d.portfolio}
            onChange={(e) => set("portfolio", e.target.value)}
            maxLength={500}
            placeholder="https://example.com"
          />
        </Field>
      </div>
      <small>
        Эти ссылки будут видны работодателям вместе с опубликованным резюме. Не
        добавляйте ссылки, если хотите скрыть указанные на них контакты.
      </small>
      <div className="section-heading">
        <h2 id="experience">Опыт работы</h2>
        <span>04 / Роли и результаты</span>
      </div>
      <p className="muted resume-editor-hint">
        Опишите свой вклад: над чем работали, за что отвечали и чего достигли.
        Можно добавить учебный или волонтёрский опыт.
      </p>
      {d.experience.map((item, index) => (
        <fieldset className="resume-entry" key={index}>
          <legend>Место работы {index + 1}</legend>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="resume-delete"
            aria-label={`Удалить место работы ${index + 1}`}
            onClick={() =>
              set(
                "experience",
                d.experience.filter((_, i) => i !== index),
              )
            }
          >
            <Trash2 size={16} />
            Удалить
          </Button>
          <div className="form-grid">
            <Field label="Компания или организация">
              <input
                required
                value={item.company}
                onChange={(e) => work(index, { company: e.target.value })}
                maxLength={150}
              />
            </Field>
            <Field label="Ваша роль">
              <input
                required
                value={item.role}
                onChange={(e) => work(index, { role: e.target.value })}
                maxLength={150}
                placeholder="Backend-разработчик"
              />
            </Field>
            <Field label="Начало работы">
              <input
                type="month"
                required
                value={item.start}
                onChange={(e) => work(index, { start: e.target.value })}
              />
            </Field>
            <Field label="Окончание">
              <input
                type="month"
                required={!item.current}
                disabled={item.current}
                min={item.start}
                value={item.end}
                onChange={(e) => work(index, { end: e.target.value })}
              />
            </Field>
          </div>
          <label className="check">
            <input
              type="checkbox"
              checked={item.current}
              onChange={(e) =>
                work(index, {
                  current: e.target.checked,
                  end: e.target.checked ? "" : item.end,
                })
              }
            />
            Работаю сейчас
          </label>
          <Field label="Задачи и ответственность">
            <textarea
              rows={3}
              value={item.description}
              onChange={(e) => work(index, { description: e.target.value })}
              maxLength={3000}
            />
          </Field>
          <Field label="Результаты">
            <textarea
              rows={2}
              value={item.results}
              onChange={(e) => work(index, { results: e.target.value })}
              maxLength={2000}
              placeholder="Например, сократил время обработки запроса с 300 до 80 мс"
            />
          </Field>
        </fieldset>
      ))}
      <Button
        type="button"
        variant="secondary"
        className="resume-add"
        disabled={d.experience.length >= 8}
        onClick={() =>
          set("experience", [
            ...d.experience,
            {
              company: "",
              role: "",
              start: "",
              end: "",
              current: false,
              description: "",
              results: "",
            },
          ])
        }
      >
        <Plus size={16} />
        Добавить место работы
      </Button>
      <div className="section-heading">
        <h2>Проекты</h2>
        <span>05 / Примеры работ</span>
      </div>
      {d.projects.map((item, index) => (
        <fieldset className="resume-entry" key={index}>
          <legend>Проект {index + 1}</legend>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="resume-delete"
            aria-label={`Удалить проект ${index + 1}`}
            onClick={() =>
              set(
                "projects",
                d.projects.filter((_, i) => i !== index),
              )
            }
          >
            <Trash2 size={16} />
            Удалить
          </Button>
          <div className="form-grid">
            <Field label="Название проекта">
              <input
                required
                value={item.name}
                onChange={(e) => project(index, { name: e.target.value })}
                maxLength={150}
              />
            </Field>
            <Field label="Роль в проекте">
              <input
                required
                value={item.role}
                onChange={(e) => project(index, { role: e.target.value })}
                maxLength={150}
              />
            </Field>
          </div>
          <Field label="Что делает проект">
            <textarea
              required
              rows={3}
              value={item.description}
              onChange={(e) => project(index, { description: e.target.value })}
              maxLength={3000}
            />
          </Field>
          <Field label="Ваш вклад и результат">
            <textarea
              rows={2}
              value={item.results}
              onChange={(e) => project(index, { results: e.target.value })}
              maxLength={2000}
            />
          </Field>
          <Field label="Технологии проекта" hint="Через запятую">
            <CommaInput
              value={item.stack}
              onChange={(v) => project(index, { stack: v })}
              maxLength={1000}
            />
          </Field>
          <div className="form-grid">
            <Field label="Репозиторий">
              <input
                type="url"
                value={item.repository}
                onChange={(e) => project(index, { repository: e.target.value })}
                maxLength={500}
                placeholder="https://github.com/username/project"
              />
            </Field>
            <Field label="Демонстрация проекта">
              <input
                type="url"
                value={item.demo}
                onChange={(e) => project(index, { demo: e.target.value })}
                maxLength={500}
                placeholder="https://example.com"
              />
            </Field>
          </div>
        </fieldset>
      ))}
      <Button
        type="button"
        variant="secondary"
        className="resume-add"
        disabled={d.projects.length >= 8}
        onClick={() =>
          set("projects", [
            ...d.projects,
            {
              name: "",
              role: "",
              description: "",
              results: "",
              stack: [],
              repository: "",
              demo: "",
            },
          ])
        }
      >
        <Plus size={16} />
        Добавить проект
      </Button>
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
export function ProfessionalDetails({ profile }: { profile: Profile }) {
  const d = professionalData(profile.professional);
  const preferences = [
    ["Поиск работы", searchStatuses[d.search_status]],
    [
      "Зарплатные ожидания",
      d.salary_min || d.salary_max
        ? `${salaryExpectation(d)} в месяц, до налогов`
        : "",
    ],
    ["Формат", d.work_formats.map((v) => workFormats[v]).join(", ")],
    ["Занятость", employmentTypes[d.employment]],
    ["Готовность выйти", availabilityTypes[d.availability]],
    ["Переезд", d.relocation ? "Готов к переезду" : ""],
    ["Языки", d.languages.join(", ")],
    ["Часовой пояс", d.timezone],
    ["Отраслевой опыт", d.domains.join(", ")],
  ].filter(([, value]) => value);
  return (
    <div className="resume-details">
      {preferences.length > 0 && (
        <>
          <div className="section-heading">
            <h2>Пожелания к работе</h2>
          </div>
          <dl className="resume-preferences">
            {preferences.map(([label, value]) => (
              <div key={label}>
                <dt>{label}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
        </>
      )}
      {d.interests && (
        <section className="resume-interests">
          <h2>Интересующие задачи</h2>
          <p>{d.interests}</p>
        </section>
      )}
      {(d.github || d.portfolio) && (
        <>
          <div className="section-heading">
            <h2>Портфолио</h2>
          </div>
          <div className="resume-links">
            {d.github && external(d.github, "GitHub / GitLab")}
            {d.portfolio && external(d.portfolio, "Сайт и портфолио")}
          </div>
        </>
      )}
      {d.experience.length > 0 && (
        <>
          <div className="section-heading">
            <h2>Опыт работы</h2>
            <span>Со слов кандидата</span>
          </div>
          <ol className="resume-timeline">
            {d.experience.map((item, index) => (
              <li key={index}>
                <h3>{item.role}</h3>
                <div className="resume-position">
                  <strong>{item.company}</strong>
                  <span>
                    {monthLabel(item.start)} —{" "}
                    {item.current ? "по настоящее время" : monthLabel(item.end)}
                  </span>
                </div>
                {item.description && <p>{item.description}</p>}
                {item.results && (
                  <p className="resume-results">
                    <strong>Результат</strong>
                    {item.results}
                  </p>
                )}
              </li>
            ))}
          </ol>
        </>
      )}
      {d.projects.length > 0 && (
        <>
          <div className="section-heading">
            <h2>Проекты</h2>
            <span>Со слов кандидата</span>
          </div>
          <ol className="resume-timeline">
            {d.projects.map((item, index) => (
              <li key={index}>
                <h3>{item.name}</h3>
                <span className="muted">{item.role}</span>
                <p>{item.description}</p>
                {item.results && (
                  <p className="resume-results">
                    <strong>Вклад и результат</strong>
                    {item.results}
                  </p>
                )}
                <div className="tags">
                  {item.stack.map((s) => (
                    <span className="skill" key={s}>
                      {s}
                    </span>
                  ))}
                </div>
                <div className="resume-links">
                  {item.repository && external(item.repository, "Репозиторий")}
                  {item.demo && external(item.demo, "Посмотреть проект")}
                </div>
              </li>
            ))}
          </ol>
        </>
      )}
      {d.education.length > 0 && (
        <section>
          <div className="section-heading">
            <h2>Образование</h2>
            <span>Со слов кандидата</span>
          </div>
          <ol className="resume-timeline">
            {d.education.map((e, i) => (
              <li key={i}>
                <h3>{e.institution}</h3>
                <p>{[e.degree, e.field].filter(Boolean).join(" · ")}</p>
                <small>
                  {e.start_year || ""}
                  {e.start_year ? " — " : ""}
                  {e.current ? "по настоящее время" : e.end_year || ""}
                </small>
              </li>
            ))}
          </ol>
        </section>
      )}
      {d.certificates.length > 0 && (
        <section>
          <div className="section-heading">
            <h2>Сертификаты и курсы</h2>
            <span>Со слов кандидата</span>
          </div>
          <ol className="resume-timeline">
            {d.certificates.map((c, i) => (
              <li key={i}>
                <h3>{c.title}</h3>
                <p>{[c.issuer, c.year || ""].filter(Boolean).join(" · ")}</p>
                {c.url && external(c.url, "Документ")}
              </li>
            ))}
          </ol>
        </section>
      )}
    </div>
  );
}
