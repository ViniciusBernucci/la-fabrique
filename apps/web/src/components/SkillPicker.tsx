import type { Skill } from "../lib/api";

/**
 * Escolha das skills de um agente a partir do catálogo. Skills já gravadas no agente que não estão
 * no catálogo continuam marcadas (com aviso) para não serem perdidas ao salvar.
 */
export function SkillPicker({
  catalog,
  value,
  onChange,
}: {
  catalog: Skill[];
  value: string[];
  onChange: (skills: string[]) => void;
}) {
  const names = new Set(catalog.map((s) => s.name));
  const orphans = value.filter((v) => !names.has(v));
  const toggle = (name: string, on: boolean) =>
    onChange(on ? [...value, name] : value.filter((v) => v !== name));

  return (
    <fieldset className="skill-picker">
      <legend>Skills</legend>
      {catalog.length === 0 && orphans.length === 0 && (
        <p className="muted">Nenhuma skill no catálogo. Importe ou crie em Skills.</p>
      )}
      {[
        ...catalog.map((s) => ({ name: s.name, hint: s.description, orphan: false })),
        ...orphans.map((name) => ({ name, hint: "fora do catálogo", orphan: true })),
      ].map((s) => (
        <label key={s.name} className="skill-picker-item">
          <input
            type="checkbox"
            checked={value.includes(s.name)}
            onChange={(e) => toggle(s.name, e.target.checked)}
          />
          <span>
            <strong>{s.name}</strong>
            {s.hint && <small className={s.orphan ? "form-error" : "muted"}>{s.hint}</small>}
          </span>
        </label>
      ))}
    </fieldset>
  );
}
