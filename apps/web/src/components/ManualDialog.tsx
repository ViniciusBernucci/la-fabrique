import { type ReactNode, useEffect, useRef, useState } from "react";

export type Section = { id: string; title: string; when?: string; body: ReactNode };

/** Pequena tabela "campo → o que significa", usada nas seções dos manuais. */
export function Fields({ rows }: { rows: [ReactNode, ReactNode][] }) {
  return (
    <dl className="manual-fields">
      {rows.map(([name, text], i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: lista estática do manual, nunca reordenada.
        <div key={i}>
          <dt>{name}</dt>
          <dd>{text}</dd>
        </div>
      ))}
    </dl>
  );
}

/** Botão que abre um manual operacional em janela, com índice lateral. */
export function ManualDialog({
  buttonLabel,
  label,
  title,
  sections,
}: {
  buttonLabel: string;
  label: string;
  title: string;
  sections: Section[];
}) {
  const [open, setOpen] = useState(false);
  const closeRef = useRef<HTMLButtonElement>(null);
  const opener = useRef<HTMLButtonElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    closeRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        opener.current?.focus();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  function close() {
    setOpen(false);
    opener.current?.focus();
  }

  function jump(id: string) {
    bodyRef.current?.querySelector(`#manual-${id}`)?.scrollIntoView?.({ block: "start" });
  }

  return (
    <>
      <button
        ref={opener}
        type="button"
        className="secondary-action"
        aria-haspopup="dialog"
        onClick={() => setOpen(true)}
      >
        {buttonLabel}
      </button>
      {open && (
        <div className="dialog-backdrop">
          <div
            className="manual panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="manual-title"
          >
            <header className="manual-head">
              <div>
                <p className="page-label">{label}</p>
                <h2 id="manual-title">{title}</h2>
              </div>
              <button ref={closeRef} type="button" className="secondary-action" onClick={close}>
                Fechar
              </button>
            </header>
            <div className="manual-layout">
              <nav className="manual-toc" aria-label="Seções do manual">
                <ol>
                  {sections.map((s) => (
                    <li key={s.id}>
                      <button type="button" className="link-button" onClick={() => jump(s.id)}>
                        {s.title}
                      </button>
                    </li>
                  ))}
                </ol>
              </nav>
              <div className="manual-body" ref={bodyRef}>
                {sections.map((s) => (
                  <section key={s.id} id={`manual-${s.id}`} aria-labelledby={`manual-h-${s.id}`}>
                    <h3 id={`manual-h-${s.id}`}>{s.title}</h3>
                    {s.when && <p className="manual-when">{s.when}</p>}
                    {s.body}
                  </section>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
