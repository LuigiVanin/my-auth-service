# Feature: Grants — permissões por recurso no documento de profile

> **Estado deste documento:** plano, nada implementado. Decisões fechadas com o
> autor do projeto na sessão de 2026-09-03.
>
> **Leia antes:** [docs/steering/modules/profiles.md](../steering/modules/profiles.md) — o
> modelo de permissão, a hierarquia e a distinção clamp vs recusar. Este plano
> adiciona uma segunda **forma de escrever** o mesmo documento; não muda a álgebra.
>
> **Relacionado:** [2026-08-23-scoped-profiles.md](2026-08-23-scoped-profiles.md) —
> ainda não implementado. A seção "Interação com scoped-profiles" abaixo emenda a
> Fase 3 daquele plano e precisa ser lida junto se as duas frentes forem juntas.

---

## Context

`profiles.permissions` hoje só entende uma chave:

```json
{ "api": { "/core/users": { "methods": ["GET"], "query": { "skip": "^[0-9]+$" } } } }
```

Isso é preciso e é o que o `PermissionsGuard` sabe enforçar, mas é intragável para
escrever à mão: quem cria um profile precisa saber o path registrado de cada rota,
o verbo de cada uma, e todos os regexes de query — e o guard nega qualquer
parâmetro não declarado, então esquecer `limit` quebra a listagem. O
`MANAGER_PROFILE` seedado tem 1.400 caracteres numa linha só. Nenhuma UI de edição
de perfil vai ser construída em cima disso.

Esta frente adiciona **grants**: uma segunda chave, ao lado de `api`, que descreve
a mesma coisa em termos de recurso e ação.

```json
{ "grants": ["as::users::READ", "as::apps::CREATE"] }
```

A intenção final é que **grants sejam o formato de autoria** — o que uma UI de
criação/edição de profile manipula — e que `api` fique como o escape hatch manual,
para quando se quer liberar algo com granularidade que grant nenhum expressa
(um regex de query, um path específico). Esse caso é raro e continua sendo escrito
direto no banco.

O ponto que faz a feature ser barata: **grant vira api antes de qualquer coisa**.
Toda a álgebra — `Resolve`, `IsSubsetOf`, o guard, a hierarquia
pool → organization → participant — continua operando exclusivamente sobre rotas
de api e não fica sabendo que grants existem.

---

## Decisões

Todas fechadas. Onde houve alternativa considerada e descartada, está registrado.

| Tema | Decisão |
| --- | --- |
| Formato | `as::{feature}::{subfeature?}::(CREATE\|READ\|UPDATE\|DELETE)` |
| Namespace | Só `as::`. Qualquer outro prefixo é `400` na escrita |
| Tradução grant → api | **Mapa estático explícito** em `shared/permissions`, grant completo → fragmento no formato exato de `permissions.api` |
| Query nos grants | Sempre **aberta**. Restringir query é o que justifica escrever `api` à mão |
| Onde a expansão acontece | Dentro de `Document.resolved()`, antes de qualquer interseção |
| Merge `api` + `grants` no mesmo documento | **`api` vence no path inteiro** — se o path existe em `api`, a expansão do grant é ignorada para aquele path |
| Merge entre grants | União de métodos por path |
| Clamp na escrita | O que é gravado são os grants que **sobrevivem inteiros** ao teto do caller |
| Clamp em runtime | Continua sendo o `Resolve` de sempre — rede de segurança para linhas gravadas sob um teto que depois estreitou |
| `Resolved` | Ganha `Grants []string`: os grants que sobreviveram à interseção |
| Grant inválido | `400` na escrita; **ignorado com warn** na leitura |
| Descoberta | `GET /core/grants`, derivado do mapa |
| Seed | `MANAGER_PROFILE`, `LOGIN_PROFILE` e `MEMBER_PROFILE` reescritos em grants. `ADMIN` continua `api` |
| Migração | **Nenhuma.** `permissions` é `jsonb`, `grants` é só uma chave nova |

### Por que um mapa explícito, e não derivação por convenção

A alternativa considerada foi derivar as rotas do grant por convenção:
`as::users::READ` → `GET /core/users` + `GET /core/users/:id`, sem mapa nenhum
para manter.

Foi descartada por causa dos **paths fantasma**. A convenção necessariamente gera
paths que não existem: `as::apps::UPDATE` derivaria `PUT /core/apps` além de
`PUT /core/apps/:id`. Em runtime isso é inócuo — o guard só consulta
`ctx.Route().Path`, que é sempre uma rota registrada, então uma chave que não
corresponde a rota nenhuma nunca é lida. Mas ela **quebra toda comparação de
contenção**:

- `IsSubsetOf(requisitado, teto)` é o que recusa um `profile_id` largo demais em
  `POST /core/organizations` e `POST /core/users_pool`. Contra o
  `MANAGER_PROFILE` de hoje, que lista `/core/apps` com `["POST","GET"]`, o
  fantasma `PUT /core/apps` reprova o grant inteiro — mesmo o grant só concedendo
  de fato `PUT /core/apps/:id`, que cabe no teto.
- O mesmo vale para o filtro de escrita.

O resultado seria uma feature em que quase todo grant é recusado contra qualquer
teto escrito à mão, por causa de rotas que não existem. Um mapa escrito à mão só
nomeia rota real, e o problema não chega a existir.

O custo do mapa é divergência: uma rota nova sob o `PermissionsGuard` sem entrada
correspondente fica inalcançável por grant. Isso é coberto por um teste
(Fase 5) que usa a tabela de rotas do fiber para falhar quando acontece — a
tabela de rotas entra como **verificação**, nunca como dependência de runtime.

### Por que `api` vence no path inteiro

Dentro de um documento, `api` e `grants` são duas fontes para a mesma coisa, e a
união seria a leitura literal de "merge". O problema é o `query`: um objeto vazio
significa "aceita qualquer parâmetro", e `ResolvedRule.Query` só sabe expressar
E-lógico entre padrões — não existe forma de escrever "casa com um OU com outro".
Como grants sempre abrem a query, unir os dois lados abriria a query de qualquer
path que um grant também alcança, apagando silenciosamente o regex que alguém
escreveu à mão. Isso é escalonamento de privilégio disfarçado de conveniência.

Com `api` vencendo o path inteiro não há regra de desempate: a entrada manual é o
override, que é exatamente o papel que `api` passa a ter.

**Consequência aceita:** uma entrada em `api` pode *reduzir* o que o grant
concedia naquele path. Quem escreve `api` está refinando o path de propósito.

**Consequência que morde:** um path coberto por `api` para de acompanhar o mapa.
Se `as::users::READ` ganhar uma rota nova amanhã, um profile que declara
`/core/users` em `api` não a recebe naquele path. É o preço do override, e é o
motivo da regra de escrita da Fase 4: **nunca gravar em `api` algo que veio da
expansão de um grant**.

---

## Formato do grant

```
as :: {feature} :: {subfeature}? :: {ACTION}
```

| Parte | Regra |
| --- | --- |
| `as` | Literal. Namespace deste serviço; qualquer outro prefixo é `400` |
| `feature` | `[a-z0-9_]+` — o recurso: `users`, `apps`, `users_pool`, `organizations` |
| `subfeature` | `[a-z0-9_]+`, opcional. A maioria dos recursos não usa |
| `ACTION` | `CREATE`, `READ`, `UPDATE` ou `DELETE`, maiúsculas, exatas |

```
^as::[a-z0-9_]+(::[a-z0-9_]+)?::(CREATE|READ|UPDATE|DELETE)$
```

Casar o regex **não** basta: o grant também tem que ser uma chave do mapa. Um
grant bem formado e desconhecido é `400` na escrita, porque a alternativa é o
caller achar que concedeu algo e não ter concedido nada.

O `::` como separador é o motivo de `feature` e `subfeature` não aceitarem `:`.
A grafia é a mesma do nome do recurso na rota, incluindo o underscore de
`users_pool` — nada de tradução entre `users_pool` e `usersPool`.

---

## Fase 1 — O mapa

`shared/permissions/grants.go` (novo).

O valor é **exatamente** o que se escreveria dentro de `permissions.api`. Essa
identidade é a decisão de design inteira: expandir um grant é produzir um pedaço
de documento api, e nada além disso sabe da existência de grants.

```go
// Cada valor é um fragmento no formato de Document.Api. Query ausente significa
// aberta: um grant nunca restringe parametro de query - ver o Context.
//
// Uma chave so pode nomear rota registrada sob o PermissionsGuard. O teste de
// cobertura em tests/shared/grants_test.go falha quando uma rota nova fica sem
// grant que a alcance.
var catalog = map[string]map[string]Rule{
    "as::apps::CREATE": {
        "/core/apps": {Methods: []string{"POST"}},
    },
    "as::apps::READ": {
        "/core/apps":     {Methods: []string{"GET"}},
        "/core/apps/:id": {Methods: []string{"GET"}},
    },
    "as::apps::UPDATE": {
        "/core/apps/:id": {Methods: []string{"PUT"}},
    },

    "as::users::READ": {
        "/core/users":     {Methods: []string{"GET"}},
        "/core/users/:id": {Methods: []string{"GET"}},
    },
    "as::users::me::READ": {
        "/core/users/me": {Methods: []string{"GET"}},
    },

    "as::users_pool::CREATE": {
        "/core/users_pool": {Methods: []string{"POST"}},
    },
    "as::users_pool::READ": {
        "/core/users_pool":     {Methods: []string{"GET"}},
        "/core/users_pool/:id": {Methods: []string{"GET"}},
    },

    "as::organizations::CREATE": {
        "/core/organizations": {Methods: []string{"POST"}},
    },
    "as::organizations::READ": {
        "/core/organizations": {Methods: []string{"GET"}},
    },
    "as::organizations::switch::UPDATE": {
        "/core/organizations/switch": {Methods: []string{"PUT"}},
    },
    "as::organizations::participants::READ": {
        "/core/organizations/:id/participants": {Methods: []string{"GET"}},
    },

    "as::grants::READ": {
        "/core/grants": {Methods: []string{"GET"}},
    },
}
```

### As rotas que o mapa cobre

O `PermissionsGuard` só é montado em `/core` — `/auth` e `/otp` nunca passam por
ele. O universo é fechado, e hoje são estas dez rotas mais a nova:

| Path | Métodos | Grants |
| --- | --- | --- |
| `/core/apps` | POST, GET | `as::apps::CREATE`, `as::apps::READ` |
| `/core/apps/:id` | GET, PUT | `as::apps::READ`, `as::apps::UPDATE` |
| `/core/users` | GET | `as::users::READ` |
| `/core/users/me` | GET | `as::users::me::READ` |
| `/core/users/:id` | GET | `as::users::READ` |
| `/core/users_pool` | POST, GET | `as::users_pool::CREATE`, `as::users_pool::READ` |
| `/core/users_pool/:id` | GET | `as::users_pool::READ` |
| `/core/organizations` | GET, POST | `as::organizations::READ`, `as::organizations::CREATE` |
| `/core/organizations/switch` | PUT | `as::organizations::switch::UPDATE` |
| `/core/organizations/:id/participants` | GET | `as::organizations::participants::READ` |
| `/core/grants` | GET | `as::grants::READ` (Fase 6) |

Duas rotas não seguem o padrão REST do recurso — `/core/users/me` e
`/core/organizations/switch`. Elas entram como **subfeature**, que é exatamente
para isso que a subfeature serve: um segmento estático que não é um id.

Não existe nenhum grant `DELETE` hoje porque nenhuma rota `DELETE` está
registrada sob `/core`. O `MANAGER_PROFILE` seedado concede
`DELETE /core/apps/:id` e `PUT /core/users_pool`, ambos inexistentes — a
reescrita da Fase 5 os perde, e é uma correção, não uma regressão.

### As funções

```go
// Expande uma lista de grants no fragmento de api que ela concede. Grant
// desconhecido ou malformado e ignorado: o guard nunca pode virar 500 porque
// uma chave saiu do mapa. Ignorar concede nada, que e o lado seguro.
func expandGrants(grants []string) map[string]ResolvedRule

// Valida na escrita, onde o silencio e o lado errado.
func ValidateGrant(grant string) error
func ValidateGrants(grants []string) error

// Alimenta GET /core/grants. Ordenado, para a resposta ser estavel.
func Catalog() []string
```

---

## Fase 2 — Document, expansão e merge

### `shared/permissions/document.go`

```go
type Document struct {
    Api    map[string]Rule `json:"api"`
    Grants []string        `json:"grants"`
}
```

Documento sem `grants` desserializa em nil e se comporta exatamente como hoje.
**Nenhuma linha existente muda de significado.**

`resolved()` passa a ser expansão seguida de overlay:

```go
func (this Document) resolved() Resolved {
    resolved := Resolved{Api: expandGrants(this.Grants), Grants: knownGrants(this.Grants)}

    // api vence o path inteiro: substitui, nunca mescla. Ver "Por que api vence".
    for path, rule := range this.Api {
        resolved.Api[path] = ResolvedRule{
            Methods: rule.Methods,
            Query:   singlePatterns(rule.Query),
        }
    }

    return resolved
}
```

`expandGrants` faz **união de métodos** quando dois grants tocam o mesmo path
(`as::apps::READ` e `as::apps::UPDATE` ambos alcançam `/core/apps/:id`), e deixa
`Query` nil, que já significa aberta.

### `Resolved` ganha `Grants`

```go
type Resolved struct {
    Api    map[string]ResolvedRule `json:"api"`
    Grants []string                `json:"grants,omitempty"`
}
```

A regra de sobrevivência **não** é interseção das listas. Se ela fosse, uma
organização com `grants: [A, B]` e um participante com `api: {"*": {"methods":
["*"]}}` — que é o que o dono de qualquer organização tem — resolveria para
`grants: []`, o que é falso: o dono tem A e B.

A regra é contenção contra o resultado:

> **candidatos** = união dos grants declarados em todas as camadas.
> **sobrevive** o grant cuja expansão inteira está contida no `resolved.Api`
> final.

Isso reaproveita `resolvePath`, `methodsWithin` e `queryWithin`, que já existem
para o `IsSubsetOf`. E responde certo nos casos que importam:

| Camadas | `resolved.Grants` |
| --- | --- |
| org `[A, B]` → part `api: {"*"}` | `[A, B]` — o `*` colapsa no teto |
| org `[A, B]` → part `[A]` | `[A]` |
| org `[A]` → part `[A, B]` | `[A]` — o participante não amplia |
| org `api` manual que cobre A → part `[A]` | `[A]` |

**Um grant coberto só pela metade não aparece.** Se o teto concede
`GET /core/users` mas não `GET /core/users/:id`, `as::users::READ` não sobrevive —
embora `resolved.Api` continue concedendo a rota de listagem. `Api` é a verdade do
que o guard faz; `Grants` é um resumo honesto por recurso. Onde as duas divergem,
`Api` manda. Isso está registrado nos Pontos em aberto.

### `IsSubsetOf` — o buraco silencioso

```go
for path, childRule := range childDoc.Api {   // hoje
for path, childRule := range childDoc.resolved().Api {   // tem que virar
```

`IsSubsetOf` hoje itera `childDoc.Api` cru. Se ficar assim, **os grants de um
profile candidato não são conferidos contra o teto**, e
`POST /core/organizations` / `POST /core/users_pool` passam a aceitar um
`profile_id` que concede, via grant, mais do que o teto de quem pediu. É
escalonamento de privilégio, e é silencioso: nada quebra, nenhum teste existente
falha, a feature parece funcionar.

É a mudança mais importante da fase. Depois dela, o loop lê `ResolvedRule`, então
`singlePatterns(childRule.Query)` sai — `childRule.Query` já é `[]string`.

---

## Fase 3 — O guard

**Nenhuma mudança em `permissions.guard.go`.**

Está aqui só para constar: ele consome `*Resolved` e olha `resolved.Api`. Como a
expansão acontece dentro de `Document.resolved()`, antes de `intersect`, o guard
recebe exatamente a mesma forma de sempre.

O único ponto que vale reler é o payload de negação, que já reporta
`organization_permissions` e `participant_permissions` crus. Com grants ele passa
a mostrar a lista de grants em vez de um api gigante, o que melhora o diagnóstico
sem código nenhum.

---

## Fase 4 — Escrita: filtro pelo teto

Duas defesas, e as duas são necessárias:

1. **Na escrita** — só é gravado o que cabe no teto do caller. A linha não mente
   sobre o que concede.
2. **Em runtime** — `Resolve` clampa de novo a cada request. Cobre a linha gravada
   sob um teto que depois estreitou, e o caso de um profile de participante
   apontar para algo mais largo que o teto da organização.

O filtro de escrita **não precisa de função nova**. O clamp que a spec de
scoped-profiles já descreve produz exatamente a lista:

```go
resolved, err := permissions.Resolve(
    caller.Organization.Profile.Permissions,  // teto da organização
    caller.Participant.Profile.Permissions,   // o que o participante tem nela
    payload,                                  // o que está sendo pedido
)

// resolved.Grants JA E a lista filtrada pelo teto.
```

### A regra de gravação

```
grants gravados = resolved.Grants
api gravado     = clamp de payload.api, e SO de payload.api
```

O `api` gravado **nunca** pode conter path que veio de expansão de grant. Se
contivesse, a regra "api vence no path inteiro" congelaria aquele path: o profile
pararia de acompanhar o mapa, e um grant que ganhasse uma rota nova amanhã não a
concederia ali.

Como `resolved.Api` mistura as duas origens, separá-las depois é adivinhação.
São duas chamadas explícitas — a segunda com um documento que só carrega `api`:

```go
apiOnly, err := permissions.Resolve(
    caller.Organization.Profile.Permissions,
    caller.Participant.Profile.Permissions,
    documentWithOnlyApi(payload),
)
```

Um payload que só manda `grants` — o caso normal, e o que a UI vai fazer — nem
precisa da segunda chamada.

### O que continua em aberto de scoped-profiles

Converter `Resolved.Api` de volta para a forma de `Document` antes de gravar
(`map[string][]string` vs `map[string]string`) é o ponto em aberto nº 2 daquele
plano e **continua em aberto**. Grants o contornam: `resolved.Grants` é uma lista
de strings, grava direto. Ele só volta a morder no caso de payload com `api`.

---

## Fase 5 — Seed e o teste de cobertura

### `cmd/database/init.go`

```go
adminPermissions = json.RawMessage(`{"api": {"*": {"methods": ["*"]}}}`)

managerPermissions = json.RawMessage(`{"grants": [
    "as::apps::CREATE", "as::apps::READ", "as::apps::UPDATE",
    "as::users::READ", "as::users::me::READ",
    "as::users_pool::CREATE", "as::users_pool::READ",
    "as::organizations::CREATE", "as::organizations::READ",
    "as::organizations::switch::UPDATE",
    "as::organizations::participants::READ",
    "as::grants::READ"
]}`)

loginPermissions = json.RawMessage(`{"grants": [
    "as::organizations::READ", "as::organizations::switch::UPDATE"
]}`)

memberPermissions = json.RawMessage(`{"grants": [
    "as::organizations::READ", "as::organizations::participants::READ"
]}`)
```

`ADMIN` continua `api`: grants não produzem wildcard, e um `ADMIN` escrito como
lista de grants precisaria ser editado a cada rota nova. Não existe
`as::*::*` — se um dia fizer falta, é decisão nova.

`LOGIN_PROFILE` perde `/auth/login` e `/auth/register`. O comentário que está lá
hoje já diz que eram documentação: nenhuma das duas roda o `PermissionsGuard`.

**Quatro coisas mudam de comportamento e são intencionais:**

1. `MANAGER_PROFILE` perde `DELETE /core/apps/:id` e `PUT /core/users_pool`, que
   não são rotas registradas.
2. Todos os regexes de query desses três profiles somem. `skip`, `limit`, `name`,
   `email`, `app_id`, `pool_id` passam a aceitar qualquer valor.
3. Consequência de (2) que precisa de atenção: **o guard era a única validação de
   query que existe**. `UserListQuery` não tem tag `validate` nenhuma, então
   `?limit=abc` deixa de ser negado com 403 e chega no binding do fiber. Ver
   Pontos em aberto nº 1.
4. Um profile passa a caber numa linha legível, que é o objetivo.

### `tests/shared/grants_test.go` (novo)

O teste que impede o mapa de divergir das rotas. Sobe o `fx` como
`cmd/main_test.go` já faz, pega `server.GetRoutes()`, filtra o que está sob
`/core`, e afirma:

- toda rota `/core` registrada é alcançada por pelo menos um grant do mapa;
- todo path nomeado no mapa corresponde a uma rota registrada, com aquele método.

As duas direções importam. A primeira pega a rota nova que ninguém tornou
concedível; a segunda pega o mapa que ficou apontando para rota removida ou para
um verbo errado — que é a falha que produz um profile silenciosamente inútil.

Esta é a única aparição da tabela de rotas do fiber. Em runtime, o pacote
`permissions` continua sem dependência nenhuma.

### Os outros testes

`tests/shared/` — expansão, união de métodos entre grants, `api` vencendo o path,
`Resolve` com grants nas duas pontas, a tabela de sobrevivência de
`Resolved.Grants`, `IsSubsetOf` com child grant-shaped contra parent api-shaped.

`tests/middlewares/` — um profile grant-shaped passando e negando no guard, e o
caso em que `api` estreita um path que o grant abriria.

---

## Fase 6 — `GET /core/grants`

Para a UI de edição de profile saber quais grants existem. Sem isso o front
hardcoda uma lista que diverge do mapa no primeiro recurso novo.

| Método | Path | Comportamento |
| --- | --- | --- |
| GET | `/core/grants` | Lista os grants do mapa, agrupados por feature |

Cadeia normal: `authGuard → organizationGuard → permissionsGuard`. Derivado de
`Catalog()`, sem tocar no banco e sem service — é leitura de constante.

```json
{
  "data": [
    { "feature": "apps",   "actions": ["CREATE", "READ", "UPDATE"], "grants": ["as::apps::CREATE", "..."] },
    { "feature": "users",  "subfeature": "me", "actions": ["READ"], "grants": ["as::users::me::READ"] }
  ]
}
```

**Onde mora o controller.** O módulo `profile` não tem controller hoje; a spec de
scoped-profiles cria um. Se as duas frentes forem juntas, esta rota entra lá. Se
esta for primeiro, ela cria o controller do módulo `profile`.

O path é `/core/grants` e não `/core/profiles/grants` de propósito: o fiber casa
rota na ordem de registro, então `/core/profiles/grants` teria que ser registrada
antes de `/core/profiles/:id` para não ser engolida. Um path separado não tem essa
armadilha.

---

## Interação com scoped-profiles

[2026-08-23-scoped-profiles.md](2026-08-23-scoped-profiles.md) não foi
implementada. As duas frentes são independentes — esta não precisa daquela — mas
se forem juntas, a Fase 3 daquele plano é emendada assim:

- `CreateProfilePayload` / `UpdateProfilePayload` aceitam `grants []string` além
  de `permissions`. `ValidateGrants` roda no validador, antes do service.
- O clamp é a mesma chamada única de `Resolve` que aquele plano já descreve. O que
  muda é o que se faz com o retorno: `resolved.Grants` grava direto, e só o `api`
  do payload passa pela conversão `Resolved` → `Document`.
- A nota sobre listagem devolver `permissions` cru vale igual: quem escolhe um
  profile precisa ver os grants como estão.

Se esta frente for primeiro, ela entra sem endpoint de escrita nenhum: grants são
escritos pelo seed e pelo banco, e a Fase 4 vira documentação do que o endpoint
tem que fazer quando existir.

---

## Verificação

```bash
go build ./...
go test ./tests/shared/       # expansão, merge, Resolve, IsSubsetOf, cobertura
go test ./tests/middlewares/  # guard com profile grant-shaped
go test ./cmd/                # fx.ValidateApp, se a Fase 6 entrar
make fresh                    # reseeda com os profiles em grants
```

Depois, com as chaves de `credentials.txt`:

1. Login como admin (`ADMIN`, api wildcard) → `GET /core/apps` `200`. Prova que
   documento sem `grants` não mudou de comportamento.
2. Um usuário com `MANAGER_PROFILE` (agora grant-shaped) → `GET /core/apps`,
   `GET /core/users`, `GET /core/users/me`, `PUT /core/organizations/switch`
   todos `200`.
3. Esse mesmo usuário → `DELETE /core/apps/:id` `403`. O grant `as::apps::DELETE`
   não existe.
4. `GET /core/users?limit=abc` como manager → chega no handler, não é mais `403`.
   Confirmar qual é a resposta e decidir o Ponto em aberto nº 1.
5. Um profile de participante com `{"grants": ["as::apps::READ"]}` dentro de uma
   organização cujo teto não alcança apps → `GET /core/apps` `403`. **Prova de que
   o grant é clampado em runtime pelo teto.**
6. `POST /core/organizations` passando `profile_id` de um profile grant-shaped que
   excede o teto do caller → `403`. **Prova de que `IsSubsetOf` enxerga grants.**
7. `user.profile.permissions.grants` na resposta de login → lista dos grants
   sobreviventes.
8. `GET /core/grants` → o mapa inteiro, e todo grant listado ali é aceito na
   escrita.

---

## Pontos em aberto

1. **A query ficou sem validação.** O guard negava `?limit=abc` com 403 por causa
   do regex no profile; com os seeds em grants, ninguém nega. `UserListQuery`,
   `GetAppsQuery` e as outras não têm tag `validate`. A correção certa é validar
   query no lugar onde validação mora (`middlewares/validator.go` +
   `models-layer.md`), não devolver o regex para o profile. **Fica fora do escopo
   desta frente, mas ela é a causa** — decidir se entra junto ou vira issue.
2. **Grant meio coberto não aparece em `Resolved.Grants`.** Aceito e documentado
   acima. Se na prática incomodar (uma UI marcando checkbox por grant vai mostrar
   desmarcado algo que concede metade), a saída é `Resolved` reportar
   `partial_grants` em vez de mudar a regra de contenção.
3. **Sem grant `DELETE` nenhum hoje.** O primeiro `DELETE` registrado sob `/core`
   estreia a ação. O teste de cobertura vai cobrar.
4. **`as::*::*` não existe.** `ADMIN` continua api wildcard. Se um dia um profile
   grant-shaped precisar de "tudo", é decisão nova — e ela reabre a pergunta de
   se o mapa deve ter uma entrada sintética.
5. **Renomear uma rota quebra os profiles gravados em `api`, não os em grants.**
   É a vantagem prática da feature e vale registrar: mudar `/core/users_pool` de
   path exige reescrever todo profile api-shaped no banco, mas só uma linha no
   mapa para os grant-shaped.
6. **Nada versiona o mapa.** Remover uma chave torna inertes os grants gravados
   que a nomeiam (são ignorados com warn na leitura), sem aviso para quem tem a
   linha. Um `DELETE` de grant do mapa devia vir com uma migração de dados; hoje
   não vem.

---

## O que NÃO muda

Para o próximo agente não sair procurando:

- **`PermissionsGuard`** — nenhuma linha. Ele consome `Resolved.Api` como sempre.
- **`intersect`, `intersectMethods`, `intersectQuery`, `resolvePath`** — nenhuma
  linha. A expansão acontece antes, em `Document.resolved()`.
- **A hierarquia** pool → organization → participant, e a regra de que nenhuma
  camada amplia a anterior.
- **`profiles.permissions`** — mesma coluna, mesmo tipo, sem migração.
- **Documentos api existentes** — mesmo significado, byte por byte.

O que muda é: um mapa novo, um campo em `Document`, um campo em `Resolved`, uma
linha em `IsSubsetOf`, três literais de seed, dois arquivos de teste e uma rota.
