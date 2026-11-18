import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ErrorBoundary } from "./ErrorBoundary";

function Boom({ explode }: { explode: boolean }) {
  if (explode) throw new Error("falha de teste");
  return <p>conteúdo ok</p>;
}

function Harness() {
  const [explode, setExplode] = useState(true);
  const [key, setKey] = useState("a");
  return (
    <>
      <button type="button" onClick={() => setExplode(false)}>
        corrigir
      </button>
      <button type="button" onClick={() => setKey("b")}>
        navegar
      </button>
      <ErrorBoundary resetKey={key}>
        <Boom explode={explode} />
      </ErrorBoundary>
    </>
  );
}

afterEach(() => vi.restoreAllMocks());

describe("barreira de erro da tela", () => {
  it("mostra o erro em vez de uma página em branco e se recupera ao navegar", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    render(<Harness />);
    expect(screen.getByRole("alert")).toHaveTextContent("Esta tela encontrou um erro");
    expect(screen.getByText("falha de teste")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "corrigir" }));
    // mudar a rota (resetKey) limpa o erro e a tela volta
    await userEvent.click(screen.getByRole("button", { name: "navegar" }));
    expect(await screen.findByText("conteúdo ok")).toBeInTheDocument();
  });
});
