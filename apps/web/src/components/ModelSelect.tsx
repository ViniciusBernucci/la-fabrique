import { useState } from "react";
import type { AgentProvider, Model } from "../lib/api";

const CUSTOM = "__outro__";

/** Modelos ativos que servem ao provedor (e, no gateway de API, ao gateway da conta escolhida). */
export function modelsFor(
  provider: AgentProvider,
  gateway: string | undefined,
  all: Model[],
): Model[] {
  return all
    .filter(
      (m) =>
        m.state === "active" &&
        m.provider === provider &&
        (provider !== "api" || !gateway || m.gateway === gateway),
    )
    .sort((a, b) =>
      (a.label || a.name).localeCompare(b.label || b.name, "pt-BR", { numeric: true }),
    );
}

const optionText = (m: Model) =>
  m.label && m.label !== m.name ? `${m.label} (${m.name})` : m.name;

/**
 * Escolha do modelo a partir do catálogo do provedor. Vazio = padrão da conta (CLI). "Outro" permite
 * digitar um nome que ainda não está no catálogo. Use `key={provider}` para recomeçar ao trocar de provedor.
 */
export function ModelSelect({
  provider,
  gateway,
  models,
  value,
  onChange,
}: {
  provider: AgentProvider;
  gateway?: string;
  models: Model[];
  value: string;
  onChange: (model: string) => void;
}) {
  const options = modelsFor(provider, gateway, models);
  const [custom, setCustom] = useState(value !== "" && !options.some((m) => m.name === value));
  const selected = custom ? CUSTOM : value;

  return (
    <div className="model-select">
      <label>
        Modelo
        <select
          value={selected}
          onChange={(e) => {
            if (e.target.value === CUSTOM) {
              setCustom(true);
              onChange("");
              return;
            }
            setCustom(false);
            onChange(e.target.value);
          }}
        >
          <option value="">{provider === "api" ? "Escolha um modelo…" : "Padrão da conta"}</option>
          {options.map((m) => (
            <option key={m.id} value={m.name}>
              {optionText(m)}
            </option>
          ))}
          <option value={CUSTOM}>Outro (digitar o nome)…</option>
        </select>
      </label>
      {custom && (
        <label>
          Nome do modelo
          <input
            value={value}
            maxLength={80}
            placeholder="ex.: nome exato do modelo"
            onChange={(e) => onChange(e.target.value)}
          />
        </label>
      )}
      <small className="muted">
        {options.length === 0
          ? "Nenhum modelo cadastrado para este provedor: use “Outro” e digite o nome."
          : provider === "api"
            ? "Modelos cadastrados para o gateway da conta."
            : "Vazio usa o modelo padrão da conta."}
      </small>
    </div>
  );
}
