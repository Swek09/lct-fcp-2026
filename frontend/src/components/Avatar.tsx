import { useEffect, useRef, useState, type ChangeEvent } from "react";
import { Camera, Trash2 } from "lucide-react";
import { api, initials, queryClient } from "../api";
import { Button } from "./ui/button";
import { ErrorBox, Success } from "./shared";

export function Avatar({
  name,
  userId,
  version = 0,
  size = "",
  company = false,
}: {
  name: string;
  userId: string;
  version?: number;
  size?: string;
  company?: boolean;
}) {
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [userId, version]);
  return (
    <span className={`avatar ${size} ${company ? "company-avatar" : ""}`}>
      {version > 0 && !failed ? (
        <img
          src={`/api/avatars/${encodeURIComponent(userId)}?v=${version}`}
          alt={company ? `Логотип ${name}` : `Фото ${name}`}
          onError={() => setFailed(true)}
        />
      ) : (
        <span aria-hidden="true">
          {initials(name || (company ? "Компания" : "Профиль"))}
        </span>
      )}
    </span>
  );
}

async function prepareImage(file: File, company: boolean): Promise<string> {
  if (!["image/jpeg", "image/png", "image/webp"].includes(file.type))
    throw new Error("Выберите фото в формате JPEG, PNG или WebP.");
  if (file.size > 8 * 1024 * 1024)
    throw new Error("Размер файла должен быть не больше 8 МБ.");
  const url = URL.createObjectURL(file);
  try {
    const img = new Image();
    img.src = url;
    await img.decode();
    if (!img.naturalWidth || !img.naturalHeight)
      throw new Error("Не удалось прочитать изображение.");
    const canvas = document.createElement("canvas");
    canvas.width = canvas.height = 512;
    const ctx = canvas.getContext("2d");
    if (!ctx) throw new Error("Браузер не поддерживает обработку изображения.");
    ctx.fillStyle = "#1B1C21";
    ctx.fillRect(0, 0, 512, 512);
    const scale = company
      ? Math.min(464 / img.naturalWidth, 464 / img.naturalHeight)
      : Math.max(512 / img.naturalWidth, 512 / img.naturalHeight);
    const w = img.naturalWidth * scale,
      h = img.naturalHeight * scale;
    ctx.drawImage(img, (512 - w) / 2, (512 - h) / 2, w, h);
    return canvas.toDataURL("image/jpeg", 0.9).split(",")[1];
  } finally {
    URL.revokeObjectURL(url);
  }
}

export function AvatarEditor({
  version,
  company = false,
}: {
  name: string;
  userId: string;
  version: number;
  company?: boolean;
}) {
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(),
    [saved, setSaved] = useState("");
  const [preview, setPreview] = useState("");
  const refresh = () =>
    Promise.all(
      [
        "avatar",
        "profile",
        "company",
        "candidates",
        "candidate",
        "vacancies",
        "vacancy",
        "invitations",
        "public-company",
        "comparison-candidates",
      ].map((key) => queryClient.invalidateQueries({ queryKey: [key] })),
    );
  const choose = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setError(undefined);
    setSaved("");
    setBusy(true);
    try {
      setPreview(await prepareImage(file, company));
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  const upload = async () => {
    setBusy(true);
    setError(undefined);
    try {
      await api("/avatar", "POST", { image: preview });
      await refresh();
      setPreview("");
      setSaved(company ? "Логотип сохранён" : "Фото сохранено");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  const remove = async () => {
    setBusy(true);
    setSaved("");
    setError(undefined);
    try {
      await api("/avatar", "DELETE");
      await refresh();
      setPreview("");
      setSaved("Изображение удалено");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="avatar-editor">
      {preview && (
        <div className="avatar-editor-preview">
          <span
            className={`avatar portrait ${company ? "company-avatar" : ""}`}
          >
            <img
              src={`data:image/jpeg;base64,${preview}`}
              alt="Предпросмотр изображения"
            />
          </span>
        </div>
      )}
      <div className="avatar-editor-body">
        <strong>{company ? "Логотип компании" : "Фото профиля"}</strong>
        <p>
          {company
            ? "Логотип сохраняется целиком, с отступами."
            : "Фото будет обрезано по центру в квадрат."}{" "}
          JPEG, PNG или WebP, до 8 МБ.
        </p>
        <input
          ref={input}
          type="file"
          accept="image/jpeg,image/png,image/webp"
          onChange={choose}
          className="visually-hidden"
          aria-label={
            company ? "Выбрать логотип компании" : "Выбрать фото профиля"
          }
        />
        <div className="avatar-editor-actions">
          <Button
            type="button"
            variant="secondary"
            size="sm"
            disabled={busy}
            onClick={() => input.current?.click()}
          >
            <Camera size={16} />
            {preview
              ? "Выбрать другое"
              : company
                ? "Загрузить логотип"
                : "Загрузить фото"}
          </Button>
          {preview && (
            <>
              <Button type="button" size="sm" disabled={busy} onClick={upload}>
                {busy ? "Сохраняем…" : "Сохранить изображение"}
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={busy}
                onClick={() => {
                  setPreview("");
                  setError(undefined);
                }}
              >
                Отмена
              </Button>
            </>
          )}
          {!preview && version > 0 && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={busy}
              onClick={remove}
            >
              <Trash2 size={15} />
              Удалить
            </Button>
          )}
        </div>
        <ErrorBox error={error} />
        {saved && <Success>{saved}</Success>}
      </div>
    </div>
  );
}
