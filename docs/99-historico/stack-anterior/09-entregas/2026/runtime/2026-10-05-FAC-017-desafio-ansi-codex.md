# FAC-017 — Desafio oficial colorido

O cliente Codex produz sequências ANSI na URL/código do device flow. O worker normaliza os controles antes de extrair, preserva caso e aguarda linha completa. O contrato de desafio Codex aceita letras em ambos os casos sem ampliar tamanhos nem relaxar host HTTPS. Não há código/token em evidências; o payload transitório fica no Redis privado pelo TTL existente.

Diagnóstico real isolado publicou desafio válido do host auth.openai.com sem concluir autorização. [Relatório, testes e rollback](../configuracao/2026-10-05-FAC-017-login-device-codex.md). AWAITING_HUMAN.
