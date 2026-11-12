import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, api, request } from "./api";

function mockFetch(response: Response | Error) {
  const fn = vi.fn(async () => {
    if (response instanceof Error) throw response;
    return response;
  });
  vi.stubGlobal("fetch", fn);
  return fn;
}

afterEach(() => vi.unstubAllGlobals());

describe("request", () => {
  it("devolve o JSON em caso de sucesso e envia o cookie de sessão", async () => {
    const fn = mockFetch(new Response(JSON.stringify({ status: "ready" }), { status: 200 }));
    await expect(api.ready()).resolves.toEqual({ status: "ready" });
    expect(fn).toHaveBeenCalledWith(
      "/api/readyz",
      expect.objectContaining({ credentials: "same-origin" }),
    );
  });

  it("converte o erro padronizado da API em ApiError", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          error: {
            code: "unauthorized",
            message: "Sessão ausente.",
            action: "Entre novamente.",
            correlation_id: "abc12345",
          },
        }),
        { status: 401 },
      ),
    );
    const error = await request("/session").catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      status: 401,
      code: "unauthorized",
      message: "Sessão ausente.",
      action: "Entre novamente.",
      correlationId: "abc12345",
      isUnauthorized: true,
    });
  });

  it("trata corpo não-JSON (ex.: erro de proxy) com mensagem genérica", async () => {
    mockFetch(new Response("<html>Bad Gateway</html>", { status: 502 }));
    await expect(request("/projects")).rejects.toMatchObject({ status: 502, code: "http_error" });
  });

  it("trata falha de rede como network_error", async () => {
    mockFetch(new TypeError("Failed to fetch"));
    await expect(request("/projects")).rejects.toMatchObject({ status: 0, code: "network_error" });
  });

  it("escapa o id do projeto na URL", async () => {
    const fn = mockFetch(new Response("{}", { status: 200 }));
    await api.getProject("a/b?c");
    const [url] = fn.mock.calls[0] as unknown as [string];
    expect(url).toBe("/api/projects/a%2Fb%3Fc");
  });
});
