import type { OfficeFeed } from "./office-feed";
import { VirtualOffice } from "./VirtualOffice";
import "./office.css";

export function OfficePanel({ feed }: { feed?: OfficeFeed }) {
  return (
    <main className="shell office-panel">
      <header>
        <div>
          <p className="eyebrow">La fabrique</p>
          <h1>Escritório virtual</h1>
          <p className="office-intro">
            Cada agente é um funcionário: trabalha sentado na própria mesa, levanta para entregar um
            trabalho a outro agente e, quando livre, circula, toma café ou lê.
          </p>
        </div>
        <span className="office-badge">Dados simulados</span>
      </header>
      <VirtualOffice feed={feed} />
      <p className="office-note">
        Demonstração visual, sem ações. Não representa execuções reais; mesas vazias ficam
        disponíveis para novos agentes.
      </p>
    </main>
  );
}
