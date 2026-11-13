# FAC-017 — Login device Codex e aba de autorização

Status AWAITING_HUMAN (iniciado READY). Baseline c2e3095; developer autorizado pelo responsável. Objetivo: corrigir extração de desafio real do cliente e apresentar autorização em nova aba/fallback com status compreensível. Diagnóstico real: sessão RUNNING sem challenge, conta AUTH_REQUIRED; cliente produz ANSI e formato de código não capturado. Escopo worker/web/docs; sem tokens em logs, sem inferência, sem cobranças ou migration. Confirmar login por status oficial AVAILABLE, não por popup. Até duas rodadas de correção, checks pertinentes e relatório exato; aceite humano pendente.

Código b4411fe, 547 testes/checks passaram. Publicação do desafio real isolado confirmada; assinatura real ainda AUTH_REQUIRED, teste visual/aceite humano pendentes. [Relatório](../../../09-entregas/2026/configuracao/2026-10-05-FAC-017-login-device-codex.md).
