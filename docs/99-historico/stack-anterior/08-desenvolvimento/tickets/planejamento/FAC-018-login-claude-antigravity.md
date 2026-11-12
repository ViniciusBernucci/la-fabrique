# FAC-018 — Autenticação de Claude e Antigravity

Status AWAITING_HUMAN; baseline d6802fe; developer autorizado. Objetivo: verificar fluxos reais e incorporar login oficial na interface quando suportado, com particularidades de autorização/código/armazenamento. Escopo contratos/API/worker/web/scripts/docs, sem migrations, inferência, adapter Antigravity ou gastos. URLs/códigos temporários somente Redis e memória, owner/TTL/uso único obrigatórios; nada em outbox/SQL/log. Cliente oficial confirma disponibilidade, não popup. Provider elegível para inferência nenhum; até duas rodadas de correção. Não contornar consentimento/limites nem compartilhar sessão pessoal. Expor bloqueio operacional concreto quando armazenamento/CLI não tiver suporte comprovado. Checks completos e aceite humano separado.

Implementação 0182a4fa02dfaeb008cad2079084b251a317b12b; checks finais e limites nos relatórios de configuração. Aceite da revisão exata pendente.
