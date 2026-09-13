# Tracking por entidade: coluna dedicada e `Track`

> **Status: implementado.** Coluna, pacote de cálculo, os dois `Track`, os disparos
> e o helper de goroutine estão no código. Pontos abertos no fim.

## O problema

A feature das rotas de update deixou plantada uma chave reservada `record` dentro
de `users.metadata`, descrita como "onde o histórico de login vai ser escrito".
Nada escrevia nela. Esta é a feature que ia ocupar aquela chave, e ela cresceu:
não é só histórico de login, é métrica de tracking por entidade, com um método
`Track` no serviço de cada uma.

## Por que coluna dedicada, e não metadata nem tabela

Três candidatos foram pesados:

**`metadata.tracking`**, que era o desenho original. Zero mudança de schema e
reaproveita a maquinaria de chave reservada. Mas o `PUT` do cliente mescla
`metadata` e o tracker escreveria no mesmo campo — dois escritores concorrentes num
único blob — e a proteção continuaria sendo uma regra própria em vez da fronteira
que já existe.

**Tabela dedicada**, com unique em (entidade, tag, período, app) e
`INSERT ... ON CONFLICT DO UPDATE SET count = count + 1`. É o mais correto sob
concorrência e o único agregável entre pools. Mas não existe nenhum consumidor de
leitura agregada hoje — nenhum endpoint agrupa por app ou por mês — então a
capacidade de `GROUP BY` não tem cliente, e o custo é entidade, repositório,
migração de tabela e reset.

**Coluna `tracking jsonb` em `users` e `users_pool`**, que é o que foi feito:

1. **A proteção deixa de ser regra nova e passa a ser a fronteira que já existe.**
   `docs/steering/models-layer.md` já afirma que uma coluna ausente do update dao
   nunca é escrita, qualquer que seja o payload. Com `tracking` fora do
   `UserUpdateDao` e do `UserPoolUpdateDao`, nenhum corpo de requisição alcança a
   coluna.
2. **O cliente e o serviço param de escrever no mesmo campo.**
3. **O dado volta junto com a entidade**, ao contrário da tabela.

**Consequência: a chave reservada foi removida.** `shared/constants/metadata.go`,
`utils.FindReservedKey` e os dois call sites saíram, junto com os testes e os
parágrafos de documentação. Ela era aspiracional e ficou sem nada para guardar.
`utils.MergeJsonPatch` ficou — é o que mescla o `metadata` do cliente nas cinco
rotas de update, e não tem relação com tracking.

## Os dois shapes

`users_pool.tracking`, contadores agrupados:

```json
{"signup": {"periods": {"2026-09": {"total": 30, "apps": {"<app-uuid>": 12}}}}}
```

Chaveado por objeto e não por lista, com o período em `YYYY-MM`, para que ordem
lexicográfica seja ordem cronológica — é assim que a remoção escolhe o mês mais
antigo. O período é formatado em UTC, então o bucket não depende do timezone do
processo. Máximo `tracking.Limit` (10) períodos.

Nenhum signup individual é guardado: só o mês em que aconteceu e o app por onde
entrou.

`users.tracking`, eventos individuais:

```json
{"login": {"events": [{"app_id": "…", "at": "…", "ip": "…", "user_agent": "…", "login_type": "…"}]}}
```

Mais recente primeiro, truncado em 10.

**Isto é uma exceção consciente à regra de "nunca armazenar itens individuais,
apenas contador e parâmetros agrupados".** "Os últimos logins do usuário" é uma
lista de eventos por definição, e a exceção foi escolhida sabendo do conflito. O
signup segue a regra; o login não.

`shared/tracking/document.go` tem os tipos, o limite e as duas funções puras —
`RecordSignup` e `RecordLogin` — que fazem parse da coluna, mexem na tag certa e
remontam. Fica em `shared/` e não num módulo pela mesma razão de
`shared/permissions`: é cálculo puro sobre JSON que duas entidades precisam.

## Como a escrita é segura sob concorrência

Cada `Track` abre transação, lê a linha com `FOR UPDATE`, calcula o documento novo
em Go e escreve com um método nomeado:

```go
FindOneForUpdate(id, options...) (*entity.X, error)   // Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate})
WriteTracking(id, document, options...) (int64, error) // UpdateColumn("tracking", document)
```

`WriteTracking` é a única forma de a coluna ser escrita — ela está fora do update
dao, então nenhum payload a alcança. `UpdateColumn`, então um signup não move o
`updated_at` da pool e um login não move o do usuário: aquelas colunas querem dizer
"a linha foi configurada".

**Por que lock e read-modify-write, e não `+ 1` em SQL como o `users_count`.** O
precedente do `users_count` existe para não perder escrita concorrente, e o lock
honra esse princípio. O que não se sustenta é a forma: `users_count` é escalar e
`count + 1` é uma expressão de uma linha, enquanto isto é documento aninhado com
remoção FIFO. O equivalente em SQL puro seria uma torre de
`COALESCE(…) || jsonb_build_object(…)` — porque `jsonb_set` não cria níveis
intermediários — mais um segundo comando com `jsonb_each`/`jsonb_object_agg` para
podar em 10. Ilegível, e o projeto não tem nenhum precedente de `jsonb_set`. O lock
serializa dois tracks na mesma linha, e como tudo roda em goroutine, ninguém
espera por ele.

Uma tag que a entidade não mantém é **erro, não no-op**: o serviço da pool só
aceita `TagSignup` e o do usuário só `TagLogin`, para que uma fiação errada seja
ruidosa em vez de descartar todo evento em silêncio.

## Ordem de lock, que é a parte delicada

**O signup dispara depois do `tx.Commit()` do provisionamento, nunca antes.** Antes
dele a transação ainda segura `FOR NO KEY UPDATE` na linha da pool por causa do
incremento do `users_count`, e o `FOR UPDATE` da goroutine conflita. Não é deadlock
— a goroutine não segura nada que a transação queira — mas é espera jogada fora. É
a mesma invariante que [2026-09-09-update-routes.md](../2026-09-09-update-routes/spec.md)
registrou.

**A outra interação é benigna.** O `FOR UPDATE` conflita também com o `UPDATE` de um
`PUT` concorrente na mesma linha, porque o Postgres tranca linha inteira e um
`UPDATE` comum pega `FOR NO KEY UPDATE`. Mas o `PUT` é uma instrução única em
autocommit, então a espera é do tamanho dela, e não há perda de dado nas duas
direções: `WriteTracking` escreve só `tracking` e o `PUT` escreve só as colunas do
dao — nenhum dos dois reescreve a linha inteira.

## O helper de goroutine

`utils.Detach(logger, name, fn)` roda `fn` fora do tempo de vida da requisição e
transforma panic em linha de log.

O `recover` é a razão de ele existir em vez de um `go` pelado: **o recoverer do
Fiber cobre a pilha do handler, não a de uma goroutine que o handler gerou**, então
um panic ali derruba o processo. Nenhum dos cinco `go` do projeto tinha `recover`
antes desta feature — inclusive `go this.otpService.Invalidate(otp.ID)`, em dois
lugares. Os cinco passaram a usar o helper, o que é correção de exposição anterior
a esta feature.

O `go` que sobrou é o accept loop do servidor em `infra/bootstrap/webserver.go`:
roda pelo tempo de vida do processo, não é trabalho de requisição, e o `Detach`
seria semanticamente errado ali.

**Dois efeitos colaterais em teste que valem saber**, os dois encontrados na prática
ao escrever esta feature:

- O `recover` do `Detach` engole também o panic de "chamada de mock não esperada",
  então um stub faltando num caminho detached não quebra o teste. É por isso que a
  suíte de register stuba os dois `Track` explicitamente e assere por canal, em vez
  de confiar em `AssertCalled`.
- **Uma goroutine detached sobrevive ao teste que a iniciou.** Guardar o canal de
  asserção num campo da suíte fazia a goroutine do teste anterior lê-lo enquanto o
  teste seguinte o reatribuía — corrida que o `-race` acusou. Os canais são locais e
  devolvidos por `arrangeProvisioning`, sem estado compartilhado. Vale rodar
  `go test ./... -race` em qualquer mudança nesses caminhos.

Uma corrida foi corrigida de passagem: o closure do `AuthorizeService.Refresh`
atribuía ao `err` da função envolvente, que o handler lia ao mesmo tempo.

## De onde cada tracking é disparado

| Tracking | Local | Momento |
| --- | --- | --- |
| `signup` na pool | `RegisterService.ProvisionUser` | depois do `tx.Commit()` |
| `login` no usuário | `LoginService.LoginWithPassword` e `LoginWithOtp`, `RegisterService.RegisterWithPassword` e `RegisterWithOtp` | depois de `CreateNew` retornar |

O cadastro dispara os dois: ele também cria sessão, então é o primeiro login do
usuário.

**O login dispara em quatro lugares e não em `SessionService.CreateNew`**, que
seria o funil único, porque `CreateNew` também é alcançado por
`AuthorizeService.Refresh` — e o refresh foi deliberadamente excluído, para não
inflar o contador com renovação automática de cliente. `Refresh` reaproveita
`session.LoginType`, então de dentro de `CreateNew` ele é indistinguível de um
login.

`us.LoginFrom(session)` é o único lugar que transforma sessão em evento: a sessão já
carrega app, ip, user agent, método e momento. **O valor é construído de forma
síncrona, antes da goroutine**, porque `session.User` é atribuído logo abaixo em
todos os quatro sítios e ler a struct de outra goroutine enquanto esta escreve
seria corrida.

## Quem consegue ler

| Coluna | Tag na entidade | Onde aparece |
| --- | --- | --- |
| `users.tracking` | `json:"-"` | só `GET /core/users/me` e `GET /core/users/{id}`, por `dto.GetUserResponse` |
| `users_pool.tracking` | `json:"tracking,omitempty"` | `GET /core/users_pool/{id}`; apagado linha por linha na listagem |

A assimetria é a diferença de conteúdo. O tracking do usuário é histórico de ip, e
`entity.User` está embutido nas respostas de login, cadastro, refresh e na listagem
— com `json:"tracking"` na entidade, quem administra uma pool passaria a ver o
histórico de ip de todo usuário dela. Com `json:"-"`, esquecer de apagar não vaza
nada, que é a diferença entre isto e apagar campo por campo. `dto.GetUserResponse`
embute `entity.User` e sombreia o campo, do mesmo jeito que `ProfileResponse`
sombreia `Permissions`.

O da pool são contadores agregados, sem dado pessoal, e a organização que enxerga a
pool é a dona dela — é onde `users_count` já aparecia. Só sai da listagem porque dez
meses de contadores por linha seriam uma página que ninguém lê.

Detalhe que decidiu as tags: **`omitempty` não esconde a coluna.** Ele omite
`json.RawMessage` só quando o tamanho é zero, e com `default:'{}'` a coluna nasce
com dois bytes — então `omitempty` só serve para omitir um valor que o código zerou
de propósito, que é exatamente o que a listagem de pools faz.

## Pontos abertos

1. **Um quinto caminho de login pode esquecer de chamar o `Track`**, porque o
   disparo não está no funil `CreateNew`. É o custo de excluir o refresh.
2. **`LoginService.LoginWithOtp` grava `loginType` como `"WITH_PASSWORD"`** — bug
   anterior a esta feature, e agora ele aparece também no evento de tracking, que
   vai reportar um login com OTP como login com senha. Corrigir é uma linha, mas
   muda dado de sessão existente.
3. **O tracking pode ser perdido** se o processo morrer entre o commit e a
   goroutine. É métrica, e a alternativa (dentro da transação) devolveria latência
   ao cadastro e ao login, que é justamente o que se quer evitar. A divergência
   possível é sempre "tracking perdido", nunca "tracking sem cadastro".
4. **`sessions` já é a fonte de verdade dos logins** — ip, user agent, app, login
   type e `created_at` por sessão. `users.tracking.login` é desnormalização
   deliberada: sobrevive à invalidação e à poda de sessões, e volta junto com a
   entidade sem uma segunda consulta.
5. **Nada reconcilia nem poda o tracking.** Uma pool sem signup por onze meses
   mantém os dez meses antigos até o próximo signup empurrá-los, e um usuário
   deletado leva seu tracking junto com a linha.
6. **Não há leitura agregada entre pools.** Continua sem consumidor, e é o único
   ponto que justificaria migrar para tabela dedicada mais adiante.
