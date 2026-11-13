> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo →](01-referencia-v2.3.md)

# Dimensionamento da VPS única - MVP v2.3
Status: PLANEJADO. Estimativa de engenharia, sujeita a ensaio do repositório. Não são requisitos oficiais dos CLIs. Uma VPS abriga controle, banco/fila, clientes, execução e checks; inferência nos fornecedores, sem GPU.
## Perfis
| Recurso | Mínimo | Bom - recomendação inicial | Ideal para o MVP |
|---|---|---|---|
| CPU | 4 vCPU | 8 vCPU | 8 vCPU; ampliar se CPU for gargalo |
| RAM | 8 GB | 16 GB | 32 GB |
| Disco útil provisionado | 120 GB SSD | 200 GB SSD/NVMe | 300 GB SSD/NVMe |
| Jobs de código simultâneos | 1 leve | 1 completo | 1 inicialmente; até 2 validados |
| Browser/E2E | Opcional, leve e sequencial | 1 sessão sequencial | 1-2 conforme medição |
| Projetos ativos no experimento | 1 | 1 | 1; expandir depois |
Todos: Linux 64 bits suportado/atualizado, preferência x86_64; compatibilidade dos binários confirmada; Docker/Compose quando usados; Git/Node.js LTS suportado e toolchains adicionais exigidas pelo piloto; HTTPS, saída aos fornecedores/Git/dependências; backup externo e Git remoto. Tráfego depende de imagens, builds e artefatos; medir antes de contratar franquia. GUI completa e GPU não são requisitos; browser headless só quando necessário.
## Mínimo: 4 vCPU / 8 GB / 120 GB
Viável como ponto de partida para repositório pequeno, poucos serviços e build leve. Envelope inicial: SO/controle/Postgres/Redis/supervisor até 3 GB; execução agregada até 3 GB; margem 2 GB. Limite do job 2 vCPU/3 GB, 256 processos e 30 min, ajustáveis após ensaio. Limitar serviços de integração e não rodar builds e E2E simultâneos.
Não representa garantia de que qualquer React/NestJS/build caberá. Se baseline exceder limite, usar Bom antes do experimento completo. Swap pode amortecer picos, mas não substitui RAM; evitar execução sustentada em swap. Não selecionar VPS de 2 GB/4 GB para este escopo completo com builds.
## Bom: 8 vCPU / 16 GB / 200 GB
Perfil recomendado para começar o MVP com folga: controle e um job, incluindo build/testes; browser e revisão sequenciais. Envelope: SO 1.5 GB, controle/API/scheduler 1.5 GB, PostgreSQL 1.5 GB, Redis 0.5 GB, supervisor/observabilidade 1 GB = 6 GB; execução agregada até 6 GB; margem 4 GB.
São envelopes de planejamento, não configurações finais de PostgreSQL/Node.js. Guardar orçamento global com overhead/cache do kernel; medir RSS e memória do cgroup. Limite inicial job 4 vCPU/6 GB/256 processos/30 min. Capacidade de review e dependências auxiliares precisa caber no mesmo envelope se ativa na etapa.
## Ideal: 8 vCPU / 32 GB / 300 GB
Prioriza margem de memória para builds, serviços de integração e QA. Envelope: host/controle até 8 GB; até dois jobs com 8 GB cada; margem 8 GB. Inicialmente apenas um job, limite 4 vCPU/8 GB. Ao testar dois, limitar cada um a aproximadamente 3 vCPU e validar latência/I/O, sem dois writers no mesmo worktree.
Mais RAM não cria cota de IA. Dois jobs podem disputar CPU, disco e cotas; se CPU sustentada for o gargalo, avaliar 12-16 vCPU ou CPU dedicada apenas após medir. Ideal significa conforto para este MVP, não alta disponibilidade nem tamanho final para múltiplos produtos.
## Disco e retenção
Bom, exemplo de orçamento dos 200 GB: SO/ferramentas 25 GB; imagens/caches 55 GB; worktrees/dependências temporárias 45 GB; dados persistentes 15 GB; logs/artefatos temporários 20 GB; margem livre 40 GB. Valores dependem do projeto; limitar logs, cache e número de workspaces.
Disco anunciado não é todo livre após instalação; confirmar espaço útil. Backup no mesmo disco não protege falha da VPS. Limpar apenas jobs concluídos e evidências já preservadas. Não usar prune indiscriminado nem apagar volumes persistentes.
## Condições de contratação
Conferir vCPU compartilhada/dedicada e política de CPU sustentada, espaço realmente útil, IOPS/latência, arquitetura, acesso administrativo, virtualização necessária, possibilidade de upgrade, tráfego e preços de renovação/backup. Não selecionar fornecedor só por número nominal de vCPU. Não há preço ou fornecedor aprovado neste kit.
## Ensaio e critérios de escala
Registrar build/checks reais, pico de RAM, CPU, I/O, OOM, swap, disco e latência administrativa com job ativo. Definir meta provisória p95 API de 1s nas operações administrativas comuns, sem incluir upload/download/build; validar baseline e instrumentação.
Pausar admission se RAM acima de 85% por 5 min, disco livre abaixo de 20%, OOM ou perda de heartbeat. Se build não cabe no envelope, subir perfil/otimizar antes de diminuir isolamento. Ampliar CPU se saturação sustentada e fila/latência ruins; ampliar RAM se picos/OOM; ampliar disco se retenção/caches excederem envelope. Validar novamente após mudança do piloto.
## Segurança e disponibilidade
Usuários/redes/volumes separados, bancos não públicos, credenciais apenas no runtime oficial, root restrito ao gestor confiável, sem socket Docker no código do piloto. Containers compartilham kernel; a VPS única é ponto único de falha. RPO/RTO e restauração precisam de ensaio real. Backup externo pode usar serviço de armazenamento, sem exigir segunda VPS de execução.
