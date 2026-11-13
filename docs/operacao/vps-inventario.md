# Inventário e medição da VPS (T02-01)

**Estado: NOT_RUN.** Não há acesso à VPS nesta sessão e acessar infraestrutura exige autorização do responsável. Estão prontos os scripts **somente leitura/não destrutivos**; a medição precisa ser feita por quem tem acesso.

## Como executar (na VPS)
```sh
./scripts/vps-inventory.sh  > inventario.md   # capacidade, vizinhos, serviços, Docker, acesso
./scripts/vps-benchmark.sh  > medidas.md      # CPU, disco, início de container (sem cargas de projeto)
```
Cole os resultados abaixo (sem segredos). Os scripts não leem variáveis de ambiente de processos nem argumentos de linha de comando.

## Registro (preencher)
| Item | Valor | Fonte |
|---|---|---|
| vCPUs / RAM / swap | _NÃO MEDIDO_ | inventário |
| Disco (tipo, livre, I/O) | _NÃO MEDIDO_ | inventário + benchmark |
| Processos e serviços vizinhos | _NÃO MEDIDO_ | inventário |
| Docker / cgroup / rootless | _NÃO MEDIDO_ | inventário |
| Backup/snapshot do provedor, plano, janela | _NÃO REGISTRADO_ | painel do provedor (manual) |
| Acesso (SSH, firewall) | _NÃO REGISTRADO_ | inventário + manual |

## Envelopes a medir com carga real (ainda sem medida)
Build e checks de um projeto real, navegador headless, estabilidade após reinício, concorrência de **dois** projetos e dos serviços de controle (API, banco, proxy). Até lá, **nenhum limite global ou envelope de perfil tem valor medido**: os limites atuais do Compose (`LIMIT_*`) são provisórios.

## Reserva e limite inicial propostos (hipótese, não medida)
Reservar para API, banco, proxy e sistema pelo menos metade da RAM e 2 vCPUs, e começar com **1 execução simultânea** até o ensaio de concorrência existir. Ajustar com os números reais.
