# Feature: Organizações — 2026-08-23

> **Estado deste documento:** as-built. Ele descreve o que **está no código**, não o
> que foi planejado. A seção "Divergências do plano original" registra onde os dois
> se separaram e por quê. A seção "Handoff" é o que um próximo agente precisa saber
> antes de tocar em qualquer coisa.
>
> **Nível de verificação:** `go build ./...` limpo, `go test ./cmd/` (fx.ValidateApp)
> passando, forma de resposta do login conferida por serialização real. **Nenhum
> fluxo foi exercitado contra o banco pelo agente.** Ver "O que NÃO foi verificado".

---

## Context

A autorização do serviço era achatada: um usuário tinha **um** `profile_id`, e esse
documento decidia tudo que ele podia fazer, em qualquer contexto. Isso tornava
impossível o modelo de `organization-feature.md` — um mesmo usuário participando de
vários grupos dentro de um `users_pool`, com poderes diferentes em cada um, e com um
teto por grupo limitando o que qualquer participante alcança ali dentro.

Esta mudança introduz **Organization** (grupo de usuários dentro de um pool, com um
profile-teto) e **Participant** (a participação de um usuário numa organização, com
o profile dele naquele contexto). A permissão efetiva de um request é
`organization.profile ∩ participant.profile`, resolvida a cada request.
`users.profile_id` deixou de existir; `users.current_organization_id` carrega o
escopo ativo, e toda leitura e escrita de recurso é filtrada por ele.

---

## Invariantes do modelo

Quatro regras que o código garante e que nada garantia antes:

1. **Todo usuário tem exatamente uma organização própria**, da qual é
   `owner_user_id` e na qual participa **com o profile-teto da própria
   organização**. Criada junto com o usuário, sempre, na mesma transação.
2. **`current_organization_id` sempre aponta para uma organização em que o usuário
   participa.** Validado em `OrganizationService.Switch`.
3. **Uma organização e seus participantes vivem no mesmo pool.** Validado em
   `Switch`; garantido por construção na criação. **Não é enforçado por constraint**
   — ver "Armadilhas".
4. **A org dona de um recurso não vive dentro do pool desse recurso** — exceto a
   organização do ADMIN da plataforma, o caso excepcional do artefato. É uma
   propriedade do seed, não uma constraint.

---

## Modelo de dados (as-built)

### Tabelas novas

**`organizations`** — `infra/entities/organization.entity.go`

| Coluna | Notas |
| --- | --- |
| `id` uuid PK | |
| `users_pool_id` uuid NOT NULL, index | |
| `owner_user_id` bigint NULL, index | **Nullable de propósito**: quebra o ciclo de inserção com `users` |
| `profile_id` uuid NOT NULL | O teto da organização |
| `name`, `description` | `description` tem default `''` |
| `metadata` jsonb | |

`Profile` é `json:"-"` — o teto nunca é serializado. Ver `docs/steering/modules/profiles.md`.

**`participants`** — `infra/entities/participant.entity.go`

| Coluna | Notas |
| --- | --- |
| `id` uuid PK | |
| `organization_id` uuid NOT NULL | unique `(organization_id, user_id)` |
| `user_id` bigint NOT NULL | `uint`, não uuid — todo FK para user no repo é `uint` |
| `profile_id` uuid NOT NULL | O papel do usuário nessa organização |
| `metadata` jsonb | |

O unique composto é o que torna o `COUNT` sobre o join de organizações seguro sem
`DISTINCT`. Se ele cair, `FindByParticipantUserIdCount` passa a contar errado.

### Alterações em tabelas existentes

| Tabela | Mudança |
| --- | --- |
| `users` | **`profile_id` REMOVIDA**. `current_organization_id` uuid NOT NULL adicionada |
| `users_pool` | `default_profile_id` uuid NOT NULL, `organization_id` uuid NULL |
| `apps` | `organization_id` uuid NULL |
| `app_role_profiles` | **TABELA REMOVIDA** |

`users_pool.organization_id` e `apps.organization_id` são nullable porque a org dona
do pool seedado vive dentro dele (invariante 4).

### Órfãos deixados de propósito

| O quê | Situação |
| --- | --- |
| `apps.role` + enum `app_role` | Sem consumidor. Único leitor era `GetProfileByAppRole`. Mantidos como metadado descritivo, com comentário na entidade |
| `apps.parent_app_id` | Ainda gravado em `CreateWithUserPool`, nunca lido. O escopo virou `organization_id` |
| `apps.owner_user_id`, `users_pool.owner_user_id` | Ainda gravados, não usados em nenhum filtro |
| `profiles.parent_profile_id` | Nunca foi lido pelo código; só o seed preenche. **Remoção agendada** em [2026-08-23-scoped-profiles.md](2026-08-23-scoped-profiles.md) — um campo que parece significativo e não é vira armadilha justamente na feature de profiles escopados |

---

## Migração

**`AutoMigrate` do GORM nunca dropa coluna nem tabela.** `cmd/database/migration/main.go`
tem um passo destrutivo explícito, `dropRemoved`, antes do AutoMigrate:

```go
DROP TABLE IF EXISTS app_role_profiles
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS profile_id
```

Sem ele, `users.profile_id` continua `NOT NULL` no Postgres, o GORM para de enviá-la
no INSERT e **o seed quebra na primeira inserção de usuário**. Falha é fatal de
propósito: um warning no log não seria notado.

### `make fresh` é obrigatório e a ordem importa

`make fresh` = `seed reset && migrate && seed init`.

1. `reset` trunca → tabelas vazias
2. `migrate` adiciona `users.current_organization_id` e `users_pool.default_profile_id`
   como `NOT NULL` — **só funciona porque as tabelas estão vazias**
3. `init` popula

`make migrate` sozinho num banco com linhas falha. Não há script de backfill.

---

## Permissões

Documentado por inteiro em **[docs/steering/modules/profiles.md](../../docs/steering/modules/profiles.md)**.
Resumo do que existe:

- `shared/permissions` — pacote sem dependências com `Document`, `Resolved`,
  `Resolve(documents ...json.RawMessage)` (variádico, pai primeiro) e `IsSubsetOf`.
- `Resolved.Query` é `map[string][]string`, não `map[string]string`. Duas camadas
  podem restringir o mesmo parâmetro com regexes diferentes e RE2 não tem lookahead;
  escolher um lado seria escalação. **Isso muda a forma do JSON de resposta** —
  `"skip": ["^[0-9]+$"]` em vez de `"skip": "^[0-9]+$"`. Ponto em aberto.
- `IProfileService` ficou só com lookup — `FindById` e `FindByKey`. A álgebra não é
  método de service.
- **Um caller sempre nomeia profile por id, nunca por chave.** Os payloads usam
  `default_profile_id` e `profile_id` (validados como `uuid4`). A chave é handle de
  seed: serve para o `init.go` ser idempotente e para um humano reconhecer a linha.
  A única exceção é `resolveDefaultProfile`, que resolve `LOGIN_PROFILE` por chave
  quando um pool é criado sem teto — o único profile que o código precisa nomear sem
  ter id em mão.

Quatro profiles seedados: `ADMIN`, `MANAGER_PROFILE`, `LOGIN_PROFILE`,
`MEMBER_PROFILE`. Chaves em `shared/constants/profile.go`.

**Não existe profile de dono.** A posse é `organizations.owner_user_id`, e o dono
participa no mesmo profile que é o teto da organização — tem o máximo que aquela
organização pode ter, e nem um passo além. Com isso `users_pool.default_profile_id`
vira a única decisão sobre o que um signup recebe.

`MANAGER` e `CONSUMER` **não existem mais** — viraram `MANAGER_PROFILE` e
`LOGIN_PROFILE`. Renomear só é seguro porque o `reset` trunca antes.

---

## Módulos e wiring

```
app/modules/core/organization/     module, controller, models, repository, services
app/modules/core/participant/      module, models, repository, services  (sem controller)
app/modules/core/profile/          reescrito sobre entity.Profile
app/middlewares/guards/organization.guard.go
shared/permissions/
shared/constants/profile.go
```

`participant` não tem controller: as rotas de participante são escopadas por
organização e vivem no `OrganizationController` — mesmo formato de `session` e
`profile`.

### Escritas transacionais vão ao repositório

`ProvisionUser` escreve organização + usuário + participação numa transação, e para
isso `RegisterService` segura `IOrganizationRepository`, `IUserRepository` e
`IParticipantRepository` diretamente. `OrganizationService.CreateForUser` faz o
mesmo com o repositório de participante.

O motivo é o `repo.Option`: passar a escrita pelo service do outro módulo obrigaria
a atravessar a option por uma fronteira de service — ou, pior, a perder a transação
silenciosamente. Regra registrada em
[docs/steering/service-layer.md](../../docs/steering/service-layer.md).

`IOrganizationService.Create`, `SetOwner` e `IParticipantService.Create` continuam
existindo mas **não são chamados por ninguém hoje**, e já não recebem
`options ...repo.Option`. Ficam para as rotas que farão essas mutações
isoladamente — mutação única não precisa de transação e deve passar pelo service.

Leituras continuam sempre pelo service: `FindForCurrentOrganization` carrega a regra
de resolução e não pode ser duplicada.

### Resultados nomeados

Duas operações devolvem mais de uma coisa e por isso têm struct de resultado em
`services/interface.go`, conforme a regra de retorno em `CLAUDE.md`:

- `IRegisterService.ProvisionUser(app, user) (*ProvisionedUser, error)` — **público**.
  Escreve organização + usuário + participação numa transação, na única ordem que o
  ciclo `users ↔ organizations` permite. Exposto porque criar usuário não é só coisa
  de registro: um fluxo de operador, e o convite, vão precisar da mesma unidade de
  trabalho.
- `IParticipantService.FindForCurrentOrganization(user) (*ResolvedParticipation, error)`
  — participação + permissões resolvidas. Todo caminho de resposta de um único
  usuário passa por aqui (login senha, login OTP, register, refresh), para que
  nenhum fique sem e nenhum repita a regra de resolução.

`cmd/main.go` provê `guards.NewOrganizationGuard` e os módulos `organization` e
`participant` na seção Core.

### Cadeia de guards

```
AppGuard (prefixo /auth /otp /core)  →  authGuard  →  organizationGuard  →  permissionsGuard
```

`OrganizationGuard` é **por rota, nunca por prefixo**: guards de prefixo rodam antes
dos de rota, então montado em `/core` ele executaria antes do `AuthGuard` e não
acharia `user` em `Locals`.

Ele lê `user.CurrentOrganization` (pré-carregado pelo `AuthorizeService`), busca a
participação e injeta `organization` e `participant` em `Locals`.

`PermissionsGuard` não injeta service nenhum: lê os dois documentos de `Locals` e
chama `permissions.Resolve`.

---

## Rotas

| Método | Path | Guards | Estado |
| --- | --- | --- | --- |
| GET | `/core/organizations` | auth+org+perm | implementada |
| POST | `/core/organizations` | auth+org+perm + validator | implementada |
| PUT | `/core/organizations/switch` | auth+org+perm + validator | implementada |
| GET | `/core/organizations/:id/participants` | auth+org+perm | implementada |
| POST | `/core/apps` | auth+org+perm + validator | escopada por org |
| GET | `/core/apps` | auth+org+perm | escopada por org |
| GET | `/core/apps/:id` | auth+org+perm | **stub 501** (pré-existente) |
| GET | `/core/apps/:id/users` | auth+org+perm | escopada por org |
| PUT | `/core/apps/:id` | auth+org+perm | **stub 501** (pré-existente) |
| POST | `/core/users_pool` | auth+org+perm + validator | escopada por org |
| GET | `/core/users/:id` | auth | **stub 501**, sem permissionsGuard nem organizationGuard |

### Escopo aplicado

| Caminho | Antes | Agora |
| --- | --- | --- |
| `AppService.FindAll` | `owner_user_id` + `parent_app_id` para ADMIN | `apps.organization_id = currentOrg.ID`, **sem branch por profile** |
| `AppService.CreateWithUserPool` | — | grava `OrganizationId` |
| `AppService.CreateWithUserPool` (pool por id) | `FindById` **sem escopo** | `FindByIdInOrganization` → 404 fora da org |
| `AppService.Update` | `owner_user_id == user.ID \|\| Profile.Key == ADMIN` | `stored.OrganizationId == currentOrg.ID` |
| `UserService.FindAllUsersFromApp` | branch ADMIN/MANAGER | app tem que pertencer à org atual |
| `UserPoolService.Create` | só `OwnerUserId` | + `OrganizationId`, `DefaultProfileId` validado |

**O furo fechado:** `AppService.CreateWithUserPool:41` chamava `userPoolService.FindById`
com um id vindo do payload, sem escopo nenhum. Qualquer MANAGER anexava seu app a
qualquer pool do banco.

### Caminhos que NÃO ganharam escopo, de propósito

Não "conserte" nenhum destes sem pensar:

| Caminho | Por quê |
| --- | --- |
| `UserService.IsAlreadyCreated`, `FindUserInPool` | Pré-autenticação. Não existe org atual; o escopo correto é o pool do app |
| `LoginService` | Idem |
| `OtpService` | Escopado por `app_id` + contato |
| `SessionService`, `AuthorizeService.Authorize/Refresh` | Escopados por user + app |
| `AuthorizeService.ForgotPassword` | Pré-autenticação |
| `AppGuard` | Resolve o app pelas chaves, antes de existir usuário |

---

## Forma das respostas

`entity.User` **não** é mais devolvido cru em login, register e refresh. Eles usam
`models.UserResponse` (`app/modules/core/user/models/user.dto.go`):

```go
type UserResponse struct {
    entity.User
    Profile *ProfileResponse `json:"profile,omitempty"`
}

type ProfileResponse struct {
    entity.Profile
    Permissions *permissions.Resolved `json:"permissions"`
}
```

Ambos usam **embedding com sombreamento**: o campo externo tem profundidade 0 e vence
o promovido de profundidade 1 no `encoding/json`. É o que permite reaproveitar a
entidade inteira sem duplicar campos e sem função construtora — nenhum outro DTO do
projeto tem uma.

Forma resultante (conferida por serialização real):

```json
"user": {
  "id": 2, "name": "...", "email": "...", "current_organization_id": "...",
  "current_organization": { "id", "users_pool_id", "owner_user_id", "profile_id",
                            "name", "description", "metadata", timestamps },
  "profile": {
    "id": "...", "name": "Manager Profile", "key": "MANAGER_PROFILE",
    "permissions": { "api": { "/core/apps": { "methods": ["POST","GET"],
                                              "query": { "skip": ["^[0-9]+$"] } } } }
  }
}
```

- `current_organization` **sem** `profile` — o teto não é serializado.
- Um único `profile`, o do participante.
- `profile.permissions` é o documento **resolvido**, não o cru do profile.

`AuthorizeResponse` (`POST /auth/authorize`) continua com `entity.User` cru, **sem
profile**. Ele é o contrato interno do `AuthGuard`, que faz
`ctx.Locals("user", &res.User)` e precisa de um `*entity.User`. Inconsistência
conhecida e deliberada.

---

## Seed

`cmd/database/init.go`, nesta ordem — ditada pelos `NOT NULL`, sem folga:

1. Profiles: `ADMIN`, `MANAGER_PROFILE`, `LOGIN_PROFILE`, `MEMBER_PROFILE`
2. Users pool `main_app_pool`, `DefaultProfileId = MANAGER_PROFILE`
3. App `main_app`
4. Organization `admin_organization`, `ProfileId = ADMIN`, **owner nil**
5. Admin user, `CurrentOrganizationId = admin_organization.ID`
6. `UPDATE organizations SET owner_user_id`
7. Participant admin ↔ admin_organization com o profile `ADMIN` — o mesmo teto da
   organização
8. `UPDATE users_pool/apps SET organization_id`

Os passos 4–8 violam a invariante 4 de propósito (caso do ADMIN da plataforma).

`credentials.txt` e `printCredentials` passaram a incluir o Organization ID.

`reset.go` — `seededTables`: `Otp`, `Session`, `Participant`, `App`, `Organization`,
`User`, `Profile`, `UsersPool`.

---

## Divergências do plano original

Registradas porque o plano aprovado em `/home/luisf/.claude/plans/` diverge do código:

| Plano | Como ficou | Motivo |
| --- | --- | --- |
| Álgebra em `IProfileService.Intersect` | Pacote `shared/permissions` | Guard e entidades precisam sem carregar DI; decisão do usuário |
| `Resolved.Query` como `map[string]string` | `map[string][]string` | RE2 não tem lookahead para fundir dois regexes |
| Respostas com `entity.User` | `models.UserResponse` | Service não deve devolver entidade crua |
| `Organization.Profile` visível | `json:"-"` | Teto cru é enganoso e ruidoso |
| Sem `MEMBER_PROFILE` | Seedado | Sem ele a interseção nunca roda com valor não trivial |
| Payloads com `*_profile_key` | `*_profile_id`, validados como `uuid4` | Chave é handle de seed, não identificador de contrato. Sobrou um `FindByKey` interno, para o default `LOGIN_PROFILE` |
| Participante-dono com `OWNER_PROFILE` (`*`) | Participa no teto da própria organização; `OWNER_PROFILE` **removido** | `teto ∩ teto == teto ∩ (*)`, verificado. Um profile a menos no caminho crítico do registro, e uma decisão só (`default_profile_id`) em vez de duas linhas que têm de concordar |
| `AppSearch.OrChildrenOfAppId` mantido | Removido | Só existia para o branch ADMIN que saiu |
| `GetAppsQuery.OwnerUserId` mantido | Removido | Escopo já é a org atual |
| `RegisterService` mantém `userPoolRepository` | Removido | Injetado e nunca usado |

Correções de passagem, fora do escopo declarado:

- Register por OTP passou a fazer `strings.ToLower` no email. O fluxo de senha já
  fazia, e `FindUserInPool` busca em lowercase — sem isso um usuário registrado por
  OTP com maiúscula nunca conseguia logar.
- `CreateUserPool` responde com `userPool.Name` em vez de ecoar `payload.Name`.

---

## Handoff

### O que NÃO foi verificado

Nada abaixo foi exercitado contra o banco pelo agente que escreveu isto:

- `make fresh` nunca foi executado. A DDL destrutiva e a ordem do seed estão
  **raciocinadas, não testadas**.
- Nenhuma das 4 rotas de organização foi chamada.
- O escopo por `organization_id` em apps/pools não foi exercitado.
- `Switch`, `CreateForUser`, `IsSubsetOf` na criação de pool: sem execução.
- Register (senha e OTP) com a transação nova: sem execução.

O que **foi** verificado: `go build ./...`, `go test ./cmd/` (fx.ValidateApp sobre o
grafo inteiro), `gofmt`, e a serialização de `UserResponse` impressa num programa
descartável.

### Testes quebrados

Adiados a pedido do usuário. Quatro pacotes não compilam:

| Pacote | Erro |
| --- | --- |
| `tests/shared/` | `AppSearch.OwnerUserId` não existe mais |
| `tests/modules/login/` (service + controller) | fixtures usam `entity.User` onde agora é `UserResponse`; `NewLoginService` mudou de assinatura |
| `tests/modules/register/` (service + controller) | idem, mais `ITransactionManager` novo no construtor |

Passando: `./cmd/`, `./tests/bootstrap/`, `./tests/middlewares/`.

Os mocks em `tests/modules/mock/` **estão atualizados** — `organization.mock.go` e
`participant.mock.go` são novos, `profile/user/user_pool/app` foram ajustados. As
asserções `var _ I... = &Mock...{}` são o contrato em tempo de compilação.

### Armadilhas para o próximo agente

1. **`go build` e `fx.ValidateApp` não detectam uma chamada ausente nem uma forma de
   resposta errada.** Duas vezes nesta sessão um fluxo ficou sem a chamada que o
   outro tinha, com build limpo. Confira a resposta HTTP.
2. **`LoginService` tem dois fluxos** (senha e OTP) e `RegisterService` também.
   Qualquer coisa adicionada num tem que ir no outro. Foi exatamente onde escorregou.
3. **Invariante 3 não é constraint.** Nada no banco impede um participante de um pool
   diferente do da organização. Só `Switch` valida. Um endpoint de convite tem que
   validar também.
4. **`PermissionsGuard` nega query param não declarado.** Adicionar um parâmetro a
   uma rota exige atualizar os documentos de todo profile que a alcança, no seed.
5. **Chaves de permissão casam contra `ctx.Route().Path`**, o padrão registrado.
   `/core/apps/:id`, nunca `/core/apps/9f3c...`.
6. **`sessionService.CreateNew` fica fora da transação de registro.** Não aceita
   `repo.Option` e dispara invalidação de sessões numa goroutine solta.
7. **`OtpGuard` chama `authGuard.Act(ctx)` e depois `ctx.Next()` de novo**, e
   `AuthGuard.Act` já chama `ctx.Next()`. A cadeia roda duas vezes nas ações
   autenticadas. **Bug pré-existente, não tocado.**
8. `PUT /auth/forgot_password` e `PUT /core/apps/:id` documentam body mas não têm
   `BodyValidator` na cadeia. Pré-existente.

### Pontos em aberto

1. **ADMIN não tem visão cross-org.** Depois desta mudança o ADMIN só vê os apps da
   organização atual. Não existe lente cross-org.
2. **Forma do `query` resolvido.** `"skip": ["^[0-9]+$"]` em vez de string. Colapsar
   para string quando há um padrão só é possível, ao custo de perder o caso de duas
   restrições concorrentes.
3. **Sem endpoint de criação/edição de profile.** As cinco chaves seedadas são o
   universo. Quando o endpoint existir, ele deve **clampar** com `Resolve`, não
   recusar — ver `docs/steering/modules/profiles.md`.
4. **`AuthorizeResponse` sem profile**, por ser o contrato interno do `AuthGuard`.
5. **`GetUsersAppResponse` não tem campo `Limit`**, ao contrário dos outros listings.
   Pré-existente.
6. **Subir o teto de uma organização não sobe o dono dela.** A participação aponta
   para uma *linha* de profile, não para "o que o teto for". Quem escrever o fluxo
   de alterar o teto tem que mover a participação junto.

---

## Próximas etapas

Duas frentes, uma já planejada:

**Profiles escopados por organização** —
[2026-08-23-scoped-profiles.md](2026-08-23-scoped-profiles.md), plano completo com
as decisões fechadas. Adiciona `profiles.organization_id`, o endpoint de criação com
clamp nos dois verbos, e fecha o escopo de visibilidade. Também remove
`profiles.parent_profile_id`, hoje código morto.

**Convite de participante**

Convite de participante — não é "adicionar usuário", precisa de aceite por email:

- `POST /core/organizations/:id/participants/invite`
- `PUT /core/organizations/:id/invite/resolve?answer=accept|decline`

Depende de profiles escopados para valer a pena: sem profile próprio, uma
organização só tem `MEMBER_PROFILE` para oferecer a quem convida.

Puxa junto: entidade de convite com expiração, integração com o `EmailManager`, e
como um usuário criado por convite define senha.

Duas coisas já estão prontas para essa etapa: `MEMBER_PROFILE` seedado, e
`RegisterService.ProvisionUser` público — um convidado que ainda não existe no pool
precisa exatamente do mesmo provisionamento (usuário + organização própria +
participação) que o registro faz.
