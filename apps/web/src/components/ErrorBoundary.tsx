import { Component, type ErrorInfo, type ReactNode } from "react";

/**
 * Barreira de erro da área de conteúdo: se uma tela falhar ao renderizar, a pessoa vê o que houve e
 * pode tentar de novo, em vez de uma página em branco. Troque `resetKey` (ex.: a rota) para limpar.
 */
export class ErrorBoundary extends Component<
  { children: ReactNode; resetKey?: string },
  { error: Error | null }
> {
  override state: { error: Error | null } = { error: null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  override componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("falha ao renderizar a tela", error, info.componentStack);
  }

  override componentDidUpdate(prev: { resetKey?: string }) {
    if (this.state.error && prev.resetKey !== this.props.resetKey) this.setState({ error: null });
  }

  override render() {
    if (!this.state.error) return this.props.children;
    return (
      <div className="page">
        <section className="panel panel--pad" role="alert">
          <h2>Esta tela encontrou um erro</h2>
          <p className="muted">
            Não foi possível exibir esta página. Seus dados não foram alterados. Tente de novo; se o
            erro continuar, informe a mensagem abaixo.
          </p>
          <pre className="code-block">{this.state.error.message}</pre>
          <div className="actions">
            <button type="button" onClick={() => this.setState({ error: null })}>
              Tentar de novo
            </button>
          </div>
        </section>
      </div>
    );
  }
}
