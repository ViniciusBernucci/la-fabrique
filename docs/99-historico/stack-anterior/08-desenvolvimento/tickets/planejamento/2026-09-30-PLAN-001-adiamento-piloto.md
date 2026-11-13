# PLAN-001 — adiamento do piloto externo

Data: 2026-09-30

Status: DONE

Domínio: planejamento

## Decisão

O responsável decidiu definir e conduzir manualmente o piloto externo depois que o núcleo da plataforma estiver pronto. FAC-001 deixa de bloquear a construção e passa a depender de FAC-011. FAC-003 depende diretamente do FAC-000 aceito e torna-se o próximo ticket READY.

FAC-002 preserva o preflight obrigatório dos clientes oficiais, autenticação por assinatura, bloqueio de API/extras e medição de recursos, inicialmente com fixture ou repositório sintético controlado. Ele ocorre após FAC-004 e antes do primeiro adapter em FAC-005. FAC-012 depende de FAC-001 e continua exigindo projeto real para o ensaio operacional.

## Motivo

O projeto e o formato do piloto ainda não foram definidos. Exigir essa escolha agora bloquearia componentes internos que podem ser construídos e verificados sem assumir stack, dados ou requisitos de um projeto externo.

## Critérios e limites

- Apenas tickets com dependências satisfeitas no backlog podem avançar.
- Fixtures sintéticas não contam como sucesso do piloto.
- Código de projeto real não será executado antes do contrato do piloto.
- Preflight de cliente oficial, isolamento, assinatura e cobrança continua obrigatório antes do adapter correspondente.
- APIs de IA, créditos extras, autorecharge e fallback pago permanecem desligados.
- Um writer e um executor inicial permanecem obrigatórios.

## Verificação

Foram atualizados plano, backlog, contrato do piloto, política, arquitetura, operação, handoff, README raiz, prompt inicial, índice e changelog. As dependências resultantes deixam FAC-003 como único ticket READY e recolocam FAC-001 antes do FAC-012.

## Rollback

Reverter esta revisão documental restaura FAC-001 e FAC-002 como gates anteriores ao FAC-003. Não há alteração de banco, containers ou código da aplicação.

## Lessons

Nenhuma lesson foi criada: a mudança reorganiza o plano e não introduz conceito técnico novo no repositório.
