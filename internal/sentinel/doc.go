// Package sentinel é a Sentinela: a única autoridade interna que valida e efetiva
// transições de desenvolvimento (LF-AJ-03). Eventos internos (conclusão, falha,
// mudança de estado, vaga liberada) provocam reavaliação imediata e duráveis; rotinas
// agendadas (sync, correções pendentes, PRs, varredura) têm relógio, cursor e retentativa
// próprios por escopo e nunca bloqueiam umas às outras. Tudo desemboca no mesmo coordenador.
package sentinel
