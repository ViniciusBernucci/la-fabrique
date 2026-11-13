// Package gitops integra o repositório GitHub do projeto: valida a conexão (clone
// de verdade), cria o commit de uma tarefa validada e publica a branch da feature.
//
// Há duas interfaces separadas:
//   - Connector (leitura): Check, Clone e RemoteRef — nunca alteram o remoto;
//   - Effects (escrita): Push, que só roda pelo ledger de efeitos (ADR-0018), com
//     prova por ref.
//
// Regras: o git roda no host do worker (o sandbox não tem rede nem credencial);
// a credencial vem do cofre (security.Secret) e entra só por variável de
// ambiente do processo git (cabeçalho HTTP), nunca em argumento, URL, arquivo de
// configuração ou log; só o protocolo https é aceito (file apenas em teste);
// hooks do repositório clonado nunca executam; o commit é montado com
// `git fast-import` a partir do manifesto do workspace, sem extrair arquivo
// algum no host (sem travessia nem link simbólico); o push é restrito à branch
// da feature (nunca a base, sem force, sem refspec livre).
package gitops
