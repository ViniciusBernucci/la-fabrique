import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { applyLayout, readLayout } from "./layout-preference";
import "./theme-hud.css";
import "./styles.css";
import "./theme-sobrio.css";

applyLayout(readLayout());

const root = document.getElementById("root");
if (!root) {
  throw new Error("Elemento raiz não encontrado");
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
