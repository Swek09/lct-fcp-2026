import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { api, queryClient } from "../api";
import { Button } from "../components/ui/button";
import {
  ErrorBox,
  Field,
  Loading,
  PageTitle,
  Success,
} from "../components/shared";
export function Company() {
  const q = useQuery({
    queryKey: ["company"],
    queryFn: () =>
      api<{ name: string; description: string; website: string }>("/company"),
  });
  const [error, setError] = useState<unknown>();
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setBusy(true);
    setSaved(false);
    setError(undefined);
    try {
      await api(
        "/company",
        "PUT",
        Object.fromEntries(new FormData(e.currentTarget)),
      );
      await queryClient.invalidateQueries({ queryKey: ["company"] });
      setSaved(true);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <PageTitle eyebrow="Кабинет работодателя" title="Ваша компания">
        Расскажите, с кем предстоит работать кандидату.
      </PageTitle>
      <ErrorBox error={q.error || error} />
      {q.isPending ? (
        <Loading />
      ) : (
        q.data && (
          <form className="panel form-stack narrow" onSubmit={submit}>
            <Field label="Название компании">
              <input
                name="name"
                defaultValue={q.data.name}
                required
                minLength={2}
                maxLength={150}
              />
            </Field>
            <Field label="Чем занимается команда">
              <textarea
                name="description"
                rows={5}
                defaultValue={q.data.description}
                maxLength={4000}
              />
            </Field>
            <Field label="Сайт">
              <input
                name="website"
                type="url"
                placeholder="https://"
                defaultValue={q.data.website}
                maxLength={250}
              />
            </Field>
            {saved && <Success>Профиль компании сохранён</Success>}
            <Button disabled={busy}>
              {busy ? "Сохраняем…" : "Сохранить изменения"}
            </Button>
          </form>
        )
      )}
    </>
  );
}
