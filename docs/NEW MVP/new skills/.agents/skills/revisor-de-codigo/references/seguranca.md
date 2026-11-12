# Critérios adaptados de security-reviewer

Origem: agente `security-reviewer`, preservado sem alterações. Esta referência é a adaptação portátil; aplique o protocolo do SKILL.md. Exemplos Laravel/Angular só se aplicam à stack encontrada. Regras específicas do CLAUDE.md precisam existir no projeto revisado.

## Processo

1. Delimite o escopo e liste os arquivos alterados.
2. Classifique cada arquivo: Laravel backend · Angular frontend · config · migration · outro.
3. Rode o checklist correspondente sobre as linhas alteradas **e** sobre o contexto imediato
   que elas afetam.
4. Rastreie origem e destino de dado sensível: user input → query/output; secret → log/response.
5. Classifique por severidade e escreva o relatório.

## Checklist Laravel (PHP)

- **SQL Injection:** `DB::raw`, `whereRaw`, `selectRaw`, `orderByRaw`, `havingRaw` com
  interpolação/concatenação; query montada por string; `DB::statement` com input externo.
- **Mass Assignment:** `$request->all()` em `create()`/`update()`/`fill()`; `$guarded = []`;
  campo sensível (`role`, `is_admin`, `price`, `status`) em `$fillable`.
- **Autorização / IDOR:** rota ou controller sem middleware de auth; ação sobre recurso de
  terceiro sem Policy/Gate; `Model::find($request->id)` sem escopo do usuário.
- **XSS:** `{!! !!}` no Blade com dado de usuário; API que devolve HTML para o Angular consumir.
- **CSRF:** rota web fora do middleware; exceção no `VerifyCsrfToken`.
- **Upload:** validação de extensão/MIME ausente; armazenamento em path público; nome de
  arquivo controlado pelo usuário (path traversal).
- **Command Injection:** `exec`, `shell_exec`, `system`, `Process` com input externo.
- **Deserialização insegura:** `unserialize()` com dado externo.
- **Exposição de dados:** Model completo em resposta de API (sem Resource) vazando coluna
  sensível; secret hardcoded; valor de `.env`/config commitado; senha/token/PII em log.
- **Sessão/Auth:** comparação de token sem `hash_equals`; token gerado com `rand()`/`uniqid()`
  em vez de `Str::random`/`random_bytes`.
- **SSRF:** `Http::get($url)` ou cURL com URL vinda do usuário.
- **Open redirect:** `redirect($request->input(...))`.
- **Banco de produção:** migration ou script tocando `sqlsrv_CRM` / `sqlsrv_CorporeRM`, ou
  sem `connection('sqlsrv_NEW_CRM')` explícito. **[CRÍTICO] no perfil Intranet**, quando essas conexões e a restrição do CLAUDE.md §2 forem confirmadas. Não presuma nomes de banco em outro projeto.

## Checklist Angular (TypeScript)

- **XSS:** `bypassSecurityTrustHtml/Url/ResourceUrl/Script`; `[innerHTML]` com dado externo;
  manipulação direta de DOM (`ElementRef.nativeElement.innerHTML`); `eval`/`new Function`.
- **Secret no frontend:** API key, token ou credencial em `environment.ts`, service ou
  hardcoded — **tudo que vai no bundle é público**, inclusive arquivo fora do git.
- **Armazenamento de token:** JWT ou dado sensível em `localStorage`/`sessionStorage` sem
  justificativa.
- **Autorização só no front:** botão/rota escondido por `@if` ou guard sem indicação de
  checagem equivalente no backend → marcar para verificação cruzada.
- **HTTP:** chamada `http://` em produção; interceptor anexando token em domínio de terceiro;
  dado sensível em query string.
- **Open redirect:** `window.location` / `router.navigate` com valor de query param sem validação.
- **Template dinâmico:** compilação dinâmica com input do usuário.
- **Dependências:** se o diff altera `package.json`/`composer.json`, sinalize pacote com
  vulnerabilidade conhecida e sugira `npm audit` / `composer audit`.


No achado confirmado, explique a visão do atacante com pré-condições, entrada controlada e impacto, usando somente exemplos inertes. Registre mitigações comprovadas em “Verificado e OK” e dependências de código inacessível em “Precisa de validação”.

Use o formato único de achados, relatório e plano definido no SKILL.md. Não aplique alterações.
