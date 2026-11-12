> Leitura: [← Anterior](03-como-contribuir.md) · [Índice didático](../02-INDEX.md) · [Próximo →](05-matriz-cobertura.md)

# Checks reais e evidência

package.json versiona `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`; integração opt-in `npm run test:postgres` e `npm run test:redis`. Linux/systemd/cgroups são requisitos dos ensaios pertinentes; não rodar serviços/DB/provider real por migração documental.

Schemas testam payloads inválidos e compatibilidade; API testa transações/gates; worker testa lease/stop/journal/handoff; runtime testa contexto/perfis/snapshot/crypto; web testa navegação/configuração/ações. Fakes de CLI não comprovam isolamento/login/plano real. [Testes SEC propostos](12-testes-seguranca.md) não foram executados por existir catálogo.

Baseline separada de regressão, revisão/config exatas e logs sanitizados. Até duas correções, depois checkpoint/diagnóstico. Relatos anteriores preservam testes de suas revisões; não reusar como PASS de código novo. [Validação desta migração](../00-governanca/11-VALIDACAO.md) registra comandos realmente executados, inclusive NOT_RUN.
