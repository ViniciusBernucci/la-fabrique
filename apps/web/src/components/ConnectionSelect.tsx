import { Link } from "react-router-dom";
import type { Connection } from "../lib/api";

export function ConnectionSelect({
  label,
  kind,
  connections,
  value,
  onChange,
  disabled,
}: {
  label: string;
  kind: Connection["kind"] | "";
  connections: Connection[];
  value: string | null;
  onChange: (id: string | null) => void;
  disabled?: boolean;
}) {
  const options = connections.filter((c) => c.kind === kind && c.state === "active");
  return (
    <label>
      {label}
      <select
        value={value ?? ""}
        disabled={disabled || kind === ""}
        onChange={(e) => onChange(e.target.value || null)}
      >
        <option value="">Selecione…</option>
        {options.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
        {value && !options.some((c) => c.id === value) && (
          <option value={value}>Conexão indisponível (revogada ou de outro tipo)</option>
        )}
      </select>
      {kind !== "" && options.length === 0 && (
        <small className="muted">
          Nenhuma conexão ativa deste tipo. <Link to="/connections">Cadastre em Conexões</Link>.
        </small>
      )}
    </label>
  );
}
