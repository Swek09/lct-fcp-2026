import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Bookmark, BookmarkCheck } from "lucide-react";
import { api, type Candidate, queryClient } from "../api";
import { Button } from "./ui/button";
import { ErrorBox } from "./shared";

export function useShortlist() {
  return useQuery({
    queryKey: ["shortlist"],
    queryFn: () => api<Candidate[]>("/shortlist"),
  });
}
export function SaveCandidate({ id }: { id: string }) {
  const q = useShortlist();
  const saved = q.data?.some((c) => c.user_id === id) || false;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const toggle = async () => {
    setBusy(true);
    setError(undefined);
    try {
      await api(`/shortlist/${id}`, saved ? "DELETE" : "POST");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["shortlist"] }),
        queryClient.invalidateQueries({ queryKey: ["candidates"] }),
      ]);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="save-candidate">
      <Button
        variant="ghost"
        size="sm"
        disabled={busy || q.isPending || !!q.error}
        aria-pressed={saved}
        onClick={() => void toggle()}
      >
        {saved ? <BookmarkCheck size={17} /> : <Bookmark size={17} />}
        <span>{busy ? "Сохраняем…" : saved ? "В избранном" : "Сохранить"}</span>
      </Button>
      <ErrorBox error={error || q.error} />
    </div>
  );
}
