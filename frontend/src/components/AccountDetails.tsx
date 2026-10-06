import { Plus, Trash2 } from "lucide-react";
import { type ProfessionalProfile } from "../api";
import { CommaInput } from "./ProfessionalProfile";
import { Field } from "./shared";
import { Button } from "./ui/button";

export function PersonalDetailsEditor({
  value: d,
  onChange,
}: {
  value: ProfessionalProfile;
  onChange: (value: ProfessionalProfile) => void;
}) {
  const set = <K extends keyof ProfessionalProfile>(
    key: K,
    value: ProfessionalProfile[K],
  ) => onChange({ ...d, [key]: value });
  return (
    <>
      <div className="form-grid">
        <Field
          label="Часовой пояс"
          hint="Помогает команде планировать совместную работу."
        >
          <input
            value={d.timezone}
            onChange={(e) => set("timezone", e.target.value)}
            maxLength={80}
            placeholder="Europe/Moscow"
            list="timezones"
          />
          <datalist id="timezones">
            <option value="Europe/Moscow" />
            <option value="Europe/Kaliningrad" />
            <option value="Asia/Yekaterinburg" />
            <option value="Asia/Novosibirsk" />
            <option value="Asia/Vladivostok" />
          </datalist>
        </Field>
        <Field
          label="Отраслевой опыт"
          hint="Через запятую: финтех, образование, e-commerce."
        >
          <CommaInput
            value={d.domains}
            onChange={(v) => set("domains", v)}
            maxLength={700}
          />
        </Field>
      </div>
      <Field
        label="Какие задачи интересны"
        hint="С какими продуктами и задачами хотите работать дальше?"
      >
        <textarea
          rows={3}
          maxLength={2000}
          value={d.interests}
          onChange={(e) => set("interests", e.target.value)}
          placeholder="Например, развивать API продуктовой команды и оптимизировать работу с данными"
        />
      </Field>
    </>
  );
}

export function EducationEditor({
  value: d,
  onChange,
}: {
  value: ProfessionalProfile;
  onChange: (value: ProfessionalProfile) => void;
}) {
  const year = new Date().getFullYear();
  return (
    <>
      <div className="section-heading" id="education">
        <h2>Образование</h2>
        <span>06 / Обучение</span>
      </div>
      {!d.education.length && (
        <p className="muted resume-editor-hint">
          Добавьте университет, колледж или программу обучения. Этот раздел
          необязателен.
        </p>
      )}
      {d.education.map((item, index) => {
        const set = (patch: Partial<typeof item>) =>
          onChange({
            ...d,
            education: d.education.map((e, i) =>
              i === index ? { ...e, ...patch } : e,
            ),
          });
        return (
          <fieldset className="resume-editor-record" key={index}>
            <legend>Образование {index + 1}</legend>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="record-remove"
              aria-label={`Удалить образование ${index + 1}`}
              onClick={() =>
                onChange({
                  ...d,
                  education: d.education.filter((_, i) => i !== index),
                })
              }
            >
              <Trash2 size={15} />
              Удалить
            </Button>
            <Field label="Учебное заведение">
              <input
                value={item.institution}
                onChange={(e) => set({ institution: e.target.value })}
                required
                maxLength={200}
              />
            </Field>
            <div className="form-grid">
              <Field label="Степень или квалификация">
                <input
                  value={item.degree}
                  onChange={(e) => set({ degree: e.target.value })}
                  maxLength={100}
                  placeholder="Бакалавр"
                />
              </Field>
              <Field label="Направление обучения">
                <input
                  value={item.field}
                  onChange={(e) => set({ field: e.target.value })}
                  maxLength={150}
                  placeholder="Программная инженерия"
                />
              </Field>
            </div>
            <div className="form-grid">
              <Field label="Год начала">
                <input
                  type="number"
                  min={1950}
                  max={year}
                  value={item.start_year || ""}
                  onChange={(e) => set({ start_year: Number(e.target.value) })}
                />
              </Field>
              <Field label="Год окончания">
                <input
                  type="number"
                  min={item.start_year || 1950}
                  max={year + 10}
                  disabled={item.current}
                  value={item.end_year || ""}
                  onChange={(e) => set({ end_year: Number(e.target.value) })}
                />
              </Field>
            </div>
            <label className="check">
              <input
                type="checkbox"
                checked={item.current}
                onChange={(e) =>
                  set({
                    current: e.target.checked,
                    ...(e.target.checked ? { end_year: 0 } : {}),
                  })
                }
              />
              Учусь сейчас
            </label>
          </fieldset>
        );
      })}
      <Button
        type="button"
        variant="secondary"
        size="sm"
        disabled={d.education.length >= 8}
        onClick={() =>
          onChange({
            ...d,
            education: [
              ...d.education,
              {
                institution: "",
                degree: "",
                field: "",
                start_year: 0,
                end_year: 0,
                current: false,
              },
            ],
          })
        }
      >
        <Plus size={16} />
        Добавить образование
      </Button>
      <div className="section-heading" id="certificates">
        <h2>Сертификаты и курсы</h2>
        <span>07 / Развитие</span>
      </div>
      {!d.certificates.length && (
        <p className="muted resume-editor-hint">
          Укажите значимые курсы и сертификаты. Ссылка на документ поможет
          работодателю проверить сведения.
        </p>
      )}
      {d.certificates.map((item, index) => {
        const set = (patch: Partial<typeof item>) =>
          onChange({
            ...d,
            certificates: d.certificates.map((c, i) =>
              i === index ? { ...c, ...patch } : c,
            ),
          });
        return (
          <fieldset className="resume-editor-record" key={index}>
            <legend>Сертификат {index + 1}</legend>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="record-remove"
              aria-label={`Удалить сертификат ${index + 1}`}
              onClick={() =>
                onChange({
                  ...d,
                  certificates: d.certificates.filter((_, i) => i !== index),
                })
              }
            >
              <Trash2 size={15} />
              Удалить
            </Button>
            <Field label="Название курса или сертификата">
              <input
                value={item.title}
                onChange={(e) => set({ title: e.target.value })}
                required
                maxLength={200}
              />
            </Field>
            <div className="form-grid">
              <Field label="Организация">
                <input
                  value={item.issuer}
                  onChange={(e) => set({ issuer: e.target.value })}
                  maxLength={150}
                />
              </Field>
              <Field label="Год получения">
                <input
                  type="number"
                  min={1950}
                  max={year}
                  value={item.year || ""}
                  onChange={(e) => set({ year: Number(e.target.value) })}
                />
              </Field>
            </div>
            <Field label="Ссылка на документ">
              <input
                type="url"
                value={item.url}
                onChange={(e) => set({ url: e.target.value })}
                maxLength={500}
                placeholder="https://"
              />
            </Field>
          </fieldset>
        );
      })}
      <Button
        type="button"
        variant="secondary"
        size="sm"
        disabled={d.certificates.length >= 12}
        onClick={() =>
          onChange({
            ...d,
            certificates: [
              ...d.certificates,
              { title: "", issuer: "", year: 0, url: "" },
            ],
          })
        }
      >
        <Plus size={16} />
        Добавить сертификат
      </Button>
    </>
  );
}

export function AccountSections({ items }: { items: [string, string][] }) {
  return (
    <nav className="account-sections" aria-label="Разделы профиля">
      <span className="eyebrow">В этом профиле</span>
      {items.map(([id, label], i) => (
        <button
          key={id}
          type="button"
          onClick={() => {
            document
              .querySelector(".account-mobile-navigation")
              ?.removeAttribute("open");
            document
              .getElementById(id)
              ?.scrollIntoView({ behavior: "auto", block: "start" });
          }}
        >
          <span>{String(i + 1).padStart(2, "0")}</span>
          {label}
        </button>
      ))}
    </nav>
  );
}

export function ProfileProgress({
  items,
}: {
  items: { label: string; done: boolean }[];
}) {
  const completed = items.filter((i) => i.done).length;
  return (
    <section className="profile-progress">
      <div className="inline-heading">
        <h3>Полнота профиля</h3>
        <span>
          {completed}/{items.length}
        </span>
      </div>
      <progress
        value={completed}
        max={items.length}
        aria-label="Заполненные разделы профиля"
      />
      <ul>
        {items.map((i) => (
          <li key={i.label}>
            <span
              className={`completion-dot ${i.done ? "done" : ""}`}
              aria-hidden="true"
            />
            {i.label}
            <small>{i.done ? "Готово" : "Добавить"}</small>
          </li>
        ))}
      </ul>
      <p>Заполняйте только сведения, которыми готовы поделиться.</p>
    </section>
  );
}
