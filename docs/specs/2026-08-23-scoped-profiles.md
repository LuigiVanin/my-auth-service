# Feature: Profiles escopados por organização

> **Estado:** implementado em 2026-09-06 — as oito fases estão em código, migração
> ainda não rodada contra banco nenhum. Ficam abertos os pontos 1, 3, 4, 5, 7, 10 e
> 11 do fim deste documento, e a verificação manual da lista abaixo. Escrito na
> sessão que entregou [2026-08-23-organizations.md](2026-08-23-organizations.md),
> com as decisões já fechadas com o autor do projeto.
>
> Três coisas ficaram diferentes do que está escrito abaixo, e são deliberadas:
>
> - **`MANAGER_PROFILE` ganhou as quatro grants novas** (`as::profiles::*` e
>   `as::organizations::participants::UPDATE`). A Fase 5 não pedia, mas sem isso as
>   rotas novas só seriam alcançáveis pelo `ADMIN` da plataforma, e nenhuma
>   organização criada pela API chegaria nelas.
> - **`PUT /core/organizations/:id/participants/:participant_id` exige que `:id` seja
>   a organização atual.** O `PermissionsGuard` decide a requisição contra a
>   organização atual do caller; deixar a escrita cair em outra a autorizaria com
>   poder tido em outro lugar. O `GET` irmão continua aceitando qualquer organização
>   em que o caller participe — ele só lê.
> - **O `PUT /core/profiles/:id` preserva a metade `api` da linha.** Está escrito na
>   Fase 4 e não estava implementado: gravar o payload inteiro apagaria em silêncio
>   uma `api` escrita à mão.
>
> **Emendado em 2026-09-06**, em três pontos:
>
> - a [Fase 7](#fase-7--o-profile-de-admin-da-organização) faz toda organização nascer
>   com um profile de administrador escopado a ela;
> - a [Fase 8](#fase-8--quem-pode-escrever-permissão-de-quem) define quem pode escrever
>   permissão de quem, e fecha os caminhos de auto-elevação;
> - a [Fase 3](#a-recusa-nos-dois-verbos) passou de **clamp** para **recusa com 403**,
>   o que fechou o ponto em aberto nº 2.
>
> As emendas dependem dos **grants**, entregues em
> [2026-09-03-permission-grants.md](2026-09-03-permission-grants.md), e por isso não
> podiam ser escritas antes.
>
> **Leia antes:** [docs/steering/modules/profiles.md](../../docs/steering/modules/profiles.md) — o modelo de
> permissão e a hierarquia já estão lá. Este plano adiciona escopo, endpoints e as
> regras de quem escreve o quê. Ele não muda `permissions.Resolve` nem `IsSubsetOf`,
> mas acrescenta um `IsWithin` ao lado deles.

---

## Context

Hoje os quatro profiles são seedados e globais: `ADMIN`, `MANAGER_PROFILE`,
`LOGIN_PROFILE`, `MEMBER_PROFILE`. Não existe endpoint que crie profile, então
customizar o que uma organização concede exige mexer no banco. Isso trava o fluxo
mais óbvio depois de organizações existirem: uma organização querer papéis próprios
para os seus participantes.

Esta frente introduz **profiles escopados** — linhas que pertencem a uma
organização, que só ela enxerga e só ela aplica — convivendo com os **globais**, que
todos enxergam. E um endpoint para criá-los e editá-los, sempre limitado ao que o
criador já pode fazer.

O objetivo não é só conveniência: sem profiles próprios, uma organização que convida
alguém só tem `MEMBER_PROFILE` para oferecer, e a granularidade do modelo de
participação fica sem uso.

---

## Decisões

Todas fechadas. Onde houve alternativa considerada e descartada, está registrado.

| Tema | Decisão |
| --- | --- |
| Escopo | `profiles.organization_id` nullable. NULL = global, preenchido = exclusivo daquela organização |
| Chave | **Gerada**: `<org_uuid>:<NOME_EM_MACRO_CASE>`. O caller manda `name`, nunca `key` |
| Unique de `key` | **Fica como está**, global. O prefixo de uuid já garante unicidade entre organizações |
| Escrita que excede o teto | **Recusa com 403**, em `POST` e `PUT`. Não clamp — decisão revista em 2026-09-06, ver Fase 3 |
| Quem pode escrever permissão de quem | Ninguém se eleva: não edita o profile que usa, não escolhe qual usa, e a participação do criador é congelada — Fase 8 |
| Escolher um profile pelo nome | Conferido contra o que **quem pede** tem, não contra o teto da organização dele. Vale para atribuição a participante e para `default_profile_id` de pool; a criação de organização fica adiada — Fase 8 |
| Visibilidade | Um único choke point: `FindByIdVisibleTo(id, organizationId)` |
| Global vs aplicável | Global significa **enxerga**. Quem decide se pode aplicar é a conferência contra o teto, sempre |
| `ADMIN` | Passa a ser escopado à `admin_organization`, num passo novo do seed |
| `ParentProfileId` | **Removido** da entidade e da migração |
| Criação de profile global | Não existe pela API. Global é só seed |
| `DELETE /core/profiles/:id` | **Fora de escopo** — ver "Pontos em aberto" |
| Profile de admin da organização | Toda organização nasce com um, escopado a ela, documento `{"grants": ["as::*::*"]}`. O dono participa nele — Fase 7 |

### Por que a chave gerada, e não índice parcial

A alternativa considerada foi manter a chave informada pelo caller e trocar o
`unique(key)` por dois índices parciais:

```sql
UNIQUE (key) WHERE organization_id IS NULL
UNIQUE (organization_id, key) WHERE organization_id IS NOT NULL
```

Foi descartada por dois motivos. Ela exige DDL crua (GORM não expressa índice
parcial em tag), e principalmente: ela deixa o usuário tropeçar em conflito de chave
como parte do fluxo normal, porque `EDITOR` numa org e `EDITOR` em outra são pedidos
legítimos. Com o uuid no prefixo, o conflito só acontece quando é conflito de
verdade — mesmo nome, mesma organização — e aí o erro é informativo.

Um composto simples `(organization_id, key)` **não** resolveria: Postgres trata
NULLs como distintos num unique, então dois globais com a mesma chave passariam.

---

## Fase 1 — Entidade e migração

### `infra/entities/profile.entity.go`

```go
type Profile struct {
    ID   string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
    Name string `gorm:"not null" json:"name"`
    Key  string `gorm:"unique;not null" json:"key"`

    // NULL: global, visível para qualquer organização.
    // Preenchido: exclusivo daquela organização - ninguém mais enxerga ou aplica.
    OrganizationId *string `gorm:"type:uuid;default:null;index" json:"organization_id,omitempty"`

    Permissions json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"permissions"`
    Metadata    json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`

    Organization *Organization `gorm:"foreignKey:OrganizationId" json:"-"`

    CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"createdAt"`
    UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updatedAt"`
}
```

`ParentProfileId` sai. Ele nunca foi lido pelo código — só o seed preenchia — e um
campo que parece significativo e não é vira armadilha exatamente nesta feature, onde
"profile escopado derivado de um global" é a primeira ideia que alguém tem.

`Organization` é `json:"-"`: a organização dona não precisa aparecer, e serializá-la
abriria ciclo com `Organization.Profile`.

### `cmd/database/migration/main.go`

`AutoMigrate` não dropa coluna. `parent_profile_id` entra no passo `dropRemoved`,
junto das duas linhas que já estão lá:

```go
ALTER TABLE IF EXISTS profiles DROP COLUMN IF EXISTS parent_profile_id
```

`profiles.organization_id` é nullable, então o `AutoMigrate` a adiciona sem exigir
tabela vazia — **esta migração não precisa de `make fresh`**, ao contrário da de
organizações.

---

## Fase 2 — Geração da chave

`shared/utils/text.go` (novo): `func MacroCase(text string) string`.

Requisito: `<org_uuid>:<MacroCase(name)>` tem que ser um identificador estável e
sem ambiguidade. Não basta trocar espaço por `_`:

| Entrada | Saída esperada |
| --- | --- |
| `Editor` | `EDITOR` |
| `Gestão de Vendas` | `GESTAO_DE_VENDAS` |
| `read-only viewer` | `READ_ONLY_VIEWER` |
| `  duplo   espaço ` | `DUPLO_ESPACO` |
| `a:b` | `A_B` |

Regras: transliterar acentos para ASCII, upper, trocar qualquer runa não
alfanumérica por `_`, colapsar runs de `_`, aparar das pontas. **O `:` tem que cair
na troca**, senão a chave gerada fica ambígua com o separador de escopo.

Um nome que reduza a string vazia é `400`, não uma chave `<uuid>:`.

A chave é fixada na criação e **nunca muda**. `Name` é editável, `Key` não —
`ProfileUpdateDao` já não expõe `Key`. Consequência aceita: depois de renomear, a
chave reflete o nome de origem. Ela é handle, não rótulo; o rótulo é `Name`.

---

## Fase 3 — Repositório e service

### `IProfileRepository`

```go
// Global (organization_id IS NULL) mais os da organização passada. É o único
// caminho de leitura que os callers usam; FindOne cru fica para o seed.
FindVisibleTo(organizationId string, options ...repo.Option) ([]entity.Profile, error)
FindVisibleToCount(organizationId string, options ...repo.Option) (int64, error)
FindByIdVisibleTo(id string, organizationId string, options ...repo.Option) (*entity.Profile, error)
Create(profile entity.Profile, options ...repo.Option) (*entity.Profile, error)
Update(where entity.Profile, dao dto.ProfileUpdateDao, options ...repo.Option) (int64, error)
```

Predicado compartilhado entre os três, no padrão de `AppRepository.searchScope`:

```sql
(profiles.organization_id IS NULL OR profiles.organization_id = ?)
```

`organization_id IS NULL` não pode ser expresso pelo `where` tipado — o gorm
descarta o ponteiro nil — então é query dedicada, como manda
`docs/steering/repository-pattern.md`.

### `IProfileService`

`FindById` atual **passa a ser escopado**. Isto é a mudança mais importante da fase,
e é uma quebra silenciosa se feita pela metade: hoje `FindById` é irrestrito e é o
que `UserPoolService.resolveDefaultProfile` e `OrganizationService.resolveCeiling`
usam para resolver um id vindo de payload. Se ele continuar irrestrito, um caller
aplica um profile escopado de outra organização.

```go
type IProfileService interface {
    // Substitui o FindById irrestrito. nil, nil quando não existe OU não é visível.
    FindByIdVisibleTo(id string, organizationId string) (*entity.Profile, error)

    // Continua com um único job: o default LOGIN_PROFILE de um pool novo.
    FindByKey(key string) (*entity.Profile, error)

    FindAllVisibleTo(organizationId string, query *dto.GetProfilesQuery) (*dto.GetProfilesResponse, error)

    CreateForOrganization(caller *ProfileWriteContext, payload *dto.CreateProfilePayload) (*entity.Profile, error)
    UpdateForOrganization(id string, caller *ProfileWriteContext, payload *dto.UpdateProfilePayload) (*entity.Profile, error)
}

// ProfileWriteContext é o teto do caller: a organização atual e a participação
// dele nela. As duas metades são necessárias - ver a recusa abaixo.
type ProfileWriteContext struct {
    Organization *entity.Organization
    Participant  *entity.Participant
}
```

### A recusa, nos dois verbos

> Revisto em 2026-09-06. Este bloco dizia clamp — gravar o documento resolvido, mais
> estreito que o pedido. Passou a ser recusa: o que excede o teto de quem escreve é
> `403`, e nada é gravado.

O teto de quem escreve é a cadeia inteira acima dele, resolvida numa chamada só:

```go
ceiling, err := permissions.Resolve(
    caller.Organization.Profile.Permissions,  // teto da organização
    caller.Participant.Profile.Permissions,   // o que o participante tem nela
)

// O documento que vai ser gravado, montado uma vez e usado para as duas coisas:
// conferir e gravar. Ver "O corpo da requisição" na Fase 4 - o payload é struct
// tipado, para o validador alcançar Grants.
document, err := json.Marshal(permissions.Document{Grants: payload.Permissions.Grants})

within, err := permissions.IsWithin(document, ceiling)

if !within {
    return nil, e.ThrowPermissionDeniedError("...")
}
```

A camada do participante não é opcional. Conferir só contra o teto da organização
deixaria um membro autorar um profile mais largo do que ele mesmo tem.

`IsWithin(child json.RawMessage, parent *Resolved) (bool, error)` **ainda não
existe**. É o helper `withinApi` sobre o qual o `IsSubsetOf` já é construído,
exportado com um `*Resolved` no lugar do documento cru. São três linhas em
`shared/permissions/resolve.go`, não um mecanismo novo.

**O que é gravado é o payload, tal como veio.** Esse é o ganho prático de recusar em
vez de clampar, e ele **fecha o ponto em aberto nº 2**: não existe documento
resolvido para converter de volta antes de gravar, então `Resolved.Query` ser
`map[string][]string` enquanto `Document.Query` é `map[string]string` nunca vira
problema. Era o item que a versão anterior desta spec chamava de "o detalhe que mais
provavelmente vai morder".

Pedir `as::*::*` sob um teto que não alcança o catálogo inteiro é recusado inteiro.
Nunca é gravado pela metade.

### O update

- `Permissions` nil no payload → não toca, não confere.
- `Permissions` preenchido → conferido contra o teto **atual**.

Consequência aceita: se o teto da organização estreitou desde a criação, um update
que reenvia o documento como ele está hoje é **recusado**, mesmo sem alargar nada.
É correto — não se pode manter mais do que o teto concede — mas surpreende, e a
mensagem do 403 precisa dizer o que excedeu, não só que excedeu.

Um update só é possível sobre profile **escopado à organização do caller**. Global é
seed: `FindByIdVisibleTo` acha `MANAGER_PROFILE`, e o service tem que recusar
`organization_id == nil` com 403 antes de escrever.

---

## Fase 4 — Rotas

Grupo `/core`, cadeia `authGuard → organizationGuard → permissionsGuard`.

| Método | Path | Grant | Comportamento |
| --- | --- | --- | --- |
| GET | `/core/profiles` | `as::profiles::READ` | Página dos globais + os da organização atual. `permissions` cru, ver nota |
| POST | `/core/profiles` | `as::profiles::CREATE` | Cria escopado à organização atual. Body `{name, permissions, metadata?}`. `201`, ou `403` se exceder o teto do caller |
| GET | `/core/profiles/:id` | `as::profiles::READ` | Um profile visível. `404` fora do escopo |
| PUT | `/core/profiles/:id` | `as::profiles::UPDATE` | Só escopado à organização atual. `403` em global, no profile do próprio caller e no `Admin` — Fase 8 |

As quatro grants entram em `shared/permissions/grants.go` **junto com as rotas**, nunca
antes: uma chave do catálogo só pode nomear rota registrada, senão ela vira um path
fantasma e reprova comparações de contenção que deveriam passar.

**O controller já existe.** Quando esta spec foi escrita o módulo `profile` não tinha
nenhum; a frente de grants criou `app/modules/core/profile/controller/profile.controller.go`
para servir `GET /core/grants`. As quatro rotas acima entram nesse mesmo arquivo, e o
módulo já está registrado no grafo do fx — nada muda em `cmd/main.go`.

### O corpo da requisição

Decidido em 2026-09-06: **o endpoint só aceita grants.** Permissão de `api` continua
sendo escrita à mão no banco, e a API não a lê, não a escreve e não a valida.

```json
{
  "name": "Gestão de Vendas",
  "permissions": { "grants": ["as::users::READ", "as::apps::READ"] }
}
```

`permissions` é um **objeto**, e não uma lista solta de grants, de propósito: no dia em
que a escrita de `api` entrar, ela entra como segunda chave do mesmo objeto e ninguém
que já manda `grants` precisa mudar. É também exatamente a forma que fica gravada na
coluna, então não há montagem no meio do caminho.

```go
type CreateProfile struct {
    Name        string             `json:"name" validate:"required"`
    Permissions ProfilePermissions `json:"permissions" validate:"required"`
    Metadata    json.RawMessage    `json:"metadata"`
}

type ProfilePermissions struct {
    Grants []string `json:"grants" validate:"required,min=1"`
}
```

Tipar `Permissions` como struct, e não como `json.RawMessage`, é o que deixa o
validador alcançar `Grants` e rodar `permissions.ValidateGrants` antes do service.

`ProfilePermissions` não declara `api`, então um `api` que venha no corpo é
**descartado em silêncio** pelo `json.Unmarshal`, que é o comportamento padrão do Go
para campo desconhecido. A resposta devolve o profile como foi gravado, então quem
mandou consegue ver que não foi aplicado. No dia em que a escrita de `api` entrar, é um
campo novo no mesmo struct.

**Nome do tipo:** `CreateProfile` e `UpdateProfile`, sem sufixo, seguindo a regra de
`docs/steering/models-layer.md`, que foi alterada em 2026-09-06 justamente para tirar o
`...Payload`.

### A query da listagem

```go
type GetProfilesQuery struct {
    Skip           int    `query:"skip"`
    Limit          int    `query:"limit"`
    Name           string `query:"name"`
    OrganizationId string `query:"organization_id"`
}

type GetProfilesResponse struct {
    Total  int64            `json:"total"`
    Amount int              `json:"amount"`
    Skip   int              `json:"skip"`
    Limit  int              `json:"limit"`
    Data   []entity.Profile `json:"data"`
}
```

`Skip`, `Limit` e a forma da resposta seguem `GetOrganizationsResponse`. `Name` filtra
por nome sem diferenciar maiúsculas, como `GetUserPoolsQuery` já faz.

`organization_id` estreita a listagem aos profiles daquela organização. Ele **não é uma
lente para outra organização**: a visibilidade continua sendo decidida por
`FindVisibleTo`, e o filtro só pode estreitar o que ela já devolveu, nunca alargar —
qualquer id que não seja o da organização atual devolve página vazia.

Ele não expressa "só os globais", porque não há como mandar nulo numa query string. Se
esse caso aparecer, a saída é trocar o campo por um enum `scope` com `all` (padrão),
`global` e `organization`.

### O que a escolha "só grants" muda no resto da spec

Nada do que está escrito deixa de valer, e três coisas ficam mais simples:

- `IsWithin` continua sendo a verificação e continua recebendo um documento. Um
  documento que só tem `grants` é um documento.
- A conversão de `Resolved` para `Document` continua não existindo.
- A regra "o `api` gravado nunca pode conter path vindo de expansão de grant" passa a
  ser automática: o endpoint não escreve `api`.

Mas aparecem duas coisas que precisam estar escritas.

**A verificação NÃO pode virar "os grants pedidos estão em `Resolved.Grants` de quem
pede".** É a simplificação óbvia agora que só grants entram, e ela quebra justamente o
usuário mais poderoso. Medido: o admin do seed tem `Resolved.Grants == []`, porque o
`ADMIN` é escrito em `api` e nenhuma camada dele declara grant algum. Comparar as duas
listas deixaria quem pode tudo sem conseguir criar profile nenhum. A comparação é sempre
`IsWithin(documentoPedido, tetoResolvido)`, que confere contra o `Api`.

**O `PUT` só encosta na chave `grants`.** Uma linha pode ter uma chave `api` escrita à
mão, e gravar o payload inteiro a apagaria em silêncio. A jurisdição do endpoint é a
metade que ele sabe falar.

Consequência aceita: a metade `api` de uma linha pode passar a exceder o teto sem que a
API perceba, porque ela nunca a confere. É seguro em runtime — o `Resolve` de cada
request clampa de qualquer jeito, e é para isso que ele é a rede de segurança. O que se
perde é a garantia de que a linha não mente sobre o que concede, e só naquela metade.

---

## Fase 5 — Seed

Passo novo, depois do 7 (participante do admin), antes ou depois do 8, indiferente:

```
9. UPDATE profiles SET organization_id = admin_organization.ID
   WHERE id = adminProfile.ID
```

`ADMIN` nasce global porque a organização dele ainda não existe quando os profiles
são criados — a ordem do seed é ditada pelos `NOT NULL` e não tem folga — e é
escopado no fim.

`MANAGER_PROFILE`, `LOGIN_PROFILE` e `MEMBER_PROFILE` **continuam globais**.
`main_app_pool` usa `MANAGER_PROFILE` como default, então nenhum caminho existente
passa a depender de um profile invisível.

`upsertProfile` para de receber `ParentProfileId`.

A `admin_organization` **não** ganha o profile de admin da Fase 7: o dono dela continua
participando na própria linha `ADMIN`. O motivo está na Fase 7, em "A exceção".

---

## Fase 6 — Fechar o choke point

A parte que decide se a feature é segura ou decorativa. Todo lugar que aceita um
`profile_id` de fora tem que passar por `FindByIdVisibleTo`. Hoje são dois, e ambos
usam o `FindById` irrestrito:

| Arquivo | Chamada |
| --- | --- |
| `user_pool.service.go`, `resolveDefaultProfile` | `FindById(profileId)` → escopado |
| `organization.service.go`, `resolveCeiling` | `FindById(profileId)` → escopado |

Os dois já recebem a organização de quem pediu no contexto, então é troca de
assinatura, não de arquitetura. **Remover `FindById` da interface** em vez de
deixá-lo ao lado do escopado: enquanto ele existir, o próximo chamador vai usar o mais
curto.

`resolveDefaultProfile` muda duas coisas de uma vez, e vale fazer na mesma passada:
além de escopar a busca, ele passa a conferir o profile escolhido contra o que **quem
pediu** tem, e não contra o teto da organização. Isso está na Fase 8, em "As três
escolhas de profile". `resolveCeiling` só muda a busca; a comparação dele fica como
está, por decisão adiada.

Os três pontos onde um `profile_id` é gravado — `participants.profile_id`,
`organizations.profile_id`, `users_pool.default_profile_id` — não recebem id de
payload em nenhum fluxo atual (vêm do teto ou do pool). Quando o convite chegar,
`participants.profile_id` **vai** receber, e é o próximo lugar a exigir o escopo.

---

## Fase 7 — O profile de Admin da organização

> Emenda de 2026-09-06. Depende dos grants, e de tudo que vem antes nesta spec: a
> coluna de escopo (Fase 1), a chave gerada (Fase 2) e o `Create` do repositório
> (Fase 3).

### O problema

Nos dois caminhos que criam organização, o participante dono aponta hoje para a
**mesma linha** que é o teto da organização:

| Caminho | Linha |
| --- | --- |
| `RegisterService.ProvisionUser` — signup | `register.service.go:148`, `app.UsersPool.DefaultProfileId` |
| `OrganizationService.CreateForUser` — `POST /core/organizations` | `organization.service.go:245`, `ceiling.ID` |

Em permissões dá exatamente no mesmo: `Resolve(X, X) = X`, o dono faz tudo que a
organização faz. O que incomoda é outra coisa:

1. **A participação aponta para fora da organização.** Depois desta frente,
   `pool.default_profile_id` é uma linha **global**, compartilhada por todas as
   organizações nascidas naquele pool. Editar essa linha muda a participação do dono
   de todas elas de uma vez. É o único lugar onde uma participação legitimamente
   aponta para fora, e não precisa ser.
2. **Não existe identidade de administrador.** `GET /core/organizations/:id/participants`
   mostra o dono com `profile.key = "LOGIN_PROFILE"`. Não há papel de admin para
   exibir, nem para oferecer num convite.
3. **O dono fica pinado a um teto antigo.** É o ponto em aberto nº 3: a participação
   aponta para uma *linha*, não para "o que o teto for". Subir o teto da organização
   não sobe o dono.

### A decisão

Toda organização nasce com um profile **`Admin` escopado a ela mesma**, e o dono
participa nele.

| Tema | Decisão |
| --- | --- |
| Documento | `{"grants": ["as::*::*"]}` |
| Chave | `<org_uuid>:ADMIN`, pelo esquema da Fase 2 |
| Nome | `Admin` |
| Quando | Dentro da mesma transação que cria a organização, nos dois caminhos |
| Proteção | Nenhuma. É um profile escopado como qualquer outro — ver "Pontos em aberto" nº 6 |

### Por que o curinga, e não uma cópia do teto

"As mesmas liberações da organização" tem duas leituras, e elas divergem com o tempo:

- **Cópia do documento do teto** numa linha nova. É um *snapshot*: se o teto mudar
  depois, o admin não acompanha, e o ponto em aberto nº 3 continua valendo para ele.
- **Curinga.** `Resolve(teto, curinga)` colapsa no teto, sempre. É literalmente para
  isso que `candidatePaths` existe — o comentário em `resolve.go:164-167` diz que é o
  que faz um participante de `"*"` colapsar no teto em vez de alargá-lo.

O curinga ganha: acompanha o teto para sempre, sem drift e sem ninguém ter que
lembrar de sincronizar duas linhas. **Resolve o ponto em aberto nº 3 para o dono** —
e só para ele; os outros participantes continuam pinados.

### Por que o curinga é legível agora, e não era antes

`docs/steering/modules/profiles.md:355-359` registra que um participante curinga já
foi tentado e descartado: *"tracked the ceiling automatically but read as unlimited
everywhere it was shown"*. Duas coisas mudaram:

1. **A linha ganha identidade.** O que aparece é um profile chamado `Admin`, escopado
   à organização, não um documento solto.
2. **Grants tornaram o documento legível.** `as::*::*` contra
   `{"api": {"*": {"methods": ["*"]}}}`.

E o mais importante, medido: um dono com `as::*::*` sob um teto `LOGIN_PROFILE`
reporta em `user.profile.permissions.grants`

```json
["as::login::CREATE", "as::organizations::READ",
 "as::organizations::switch::UPDATE", "as::register::CREATE"]
```

e **não** um `as::*::*` pelado. `Resolved.Grants` guarda o grant cuja expansão inteira
cabe no api resolvido, e o curinga só sobrevive inteiro sob um teto total. Ou seja, o
frontend recebe a lista concreta do teto. O curinga cru só aflora na listagem de
participantes, e lá vem acompanhado de um nome.

### A exceção: a organização do seed

`as::*::*` é limitado pelo **catálogo**; `api: {"*"}` casa **qualquer rota
registrada**. Para um teto grant-shaped os dois são idênticos. Eles divergem quando o
teto alcança fora do catálogo — hoje só o `ADMIN`, e de propósito, para que uma rota
sem entrada no mapa continue alcançável pelo operador da plataforma.

Medido: `Resolve(ADMIN, as::*::*)` perde a chave `"*"` e cai para os paths do
catálogo. Dar um `Admin` grant-shaped ao operador da plataforma o clamparia ao
catálogo e anularia justamente o motivo de o `ADMIN` continuar escrito em `api`.

**A `admin_organization` do seed é a exceção:** o dono continua participando na
própria linha `ADMIN`, que a Fase 5 já escopa àquela organização. Nela, teto e profile
de admin são a mesma linha, de propósito. É o único lugar onde isso vale.

### Onde escrever

Os dois caminhos já abrem transação. O profile é escrito pelo **repositório**, não
pelo service: é a regra 6 do `CLAUDE.md` — uma unidade de trabalho escreve por
repositórios, para que um rollback desfaça tudo. `RegisterService` e
`OrganizationService` ganham `IProfileRepository`. É mudança de construtor, então
`go test ./cmd/` é obrigatório.

A ordem em `ProvisionUser` — a organização vem antes por imposição do schema:
`users.current_organization_id` é `not null` e `organizations.owner_user_id` é
nullable exatamente para quebrar o ciclo (`organization.entity.go:8-10`):

```
1. organizationRepository.Create     teto = app.UsersPool.DefaultProfileId
2. profileRepository.Create          Admin, organization_id = org.ID, {"grants":["as::*::*"]}
3. userRepository.Create             current_organization_id = org.ID
4. organizationRepository.Update     owner_user_id
5. participantRepository.Create      profile_id = admin.ID
```

`CreateForUser` é a mesma coisa sem os passos 3 e 4, que lá já estão resolvidos: o
usuário existe e o `OwnerUserId` entra no próprio `Create` da organização.

### O invariante que isto fecha

> `participants.profile_id` sempre aponta para um profile **visível** àquela
> organização: escopado a ela, ou global.

A Fase 6 fecha os `profile_id` que chegam por payload; esta fecha os que o próprio
código escreve. `MEMBER_PROFILE` continua global, e é o caso que mostra que o
invariante é "visível", não "próprio". O endpoint de convite, quando existir, é o
próximo a ter que respeitá-lo.

### Custo

Uma linha em `profiles` por organização criada. Em troca, o dono deixa de compartilhar
linha com organizações que não conhece, e a organização passa a ter um papel de admin
para exibir e para clonar quando for oferecer papéis aos convidados.

---

## Fase 8 — Quem pode escrever permissão de quem

> Emenda de 2026-09-06. A Fase 3 responde "o que pode ser gravado"; esta responde
> "por quem, e sobre quem".

Recusar o que excede o teto do caller resolve metade do problema. A outra metade é
que ninguém pode se elevar, e para isso não basta olhar o documento — é preciso olhar
**de quem** é o profile que está sendo mexido.

### As regras

| # | Regra | Verificação |
| --- | --- | --- |
| 1 | Ninguém edita o documento do profile que ele mesmo usa | `PUT /core/profiles/:id` recusa quando `id == caller.Participant.ProfileId` |
| 2 | Ninguém troca qual profile ele mesmo usa | a rota de trocar participante recusa quando o alvo é o próprio caller |
| 3 | A participação de quem criou a organização é congelada | recusa quando `participante.UserId == organization.OwnerUserId` |
| 4 | Ninguém cria ou edita profile mais largo do que ele tem | a recusa da Fase 3 |
| 5 | Ninguém entrega a outro um profile mais largo do que ele mesmo tem | `IsWithin(profileEscolhido, Resolve(tetoDaOrg, profileDeQuemPede))` → 403 |
| 6 | O profile `Admin` da organização nunca é editável | recusa quando a linha tem `key == "<org_uuid>:ADMIN"` |

A regra 3 vale inclusive para o próprio dono e para qualquer outro admin. Ela tem um
efeito colateral que é uma **garantia**, não um acidente: toda organização tem sempre
ao menos um administrador pleno que não pode ser rebaixado, então ninguém consegue se
trancar para fora da própria organização.

A regra 6 é o que mantém o profile de admin como `as::*::*` para sempre, e é o que
faz ele continuar acompanhando o teto conforme a Fase 7. Não existe "admin desta
organização menos apps": quem quer isso cria um profile novo e o atribui a alguém.

### Os caminhos de escalonamento, e qual regra fecha cada um

A tabela é o teste da fase. Se uma regra sair, a linha correspondente reabre.

| Caminho | Fechado por |
| --- | --- |
| Editar o documento do profile que eu uso | 1 |
| Criar um profile mais largo e passar a usá-lo | 4 (não consigo criá-lo) e 2 (não consigo me atribuir) |
| Me atribuir um profile mais largo que já existe, como o `Admin` da organização | 2 |
| **Duas pessoas com a grant de atribuição se promoverem mutuamente** | **5** |

O último merece o exemplo, porque não é óbvio e não era coberto pelas cinco primeiras
regras. Ana e Bruno participam da mesma organização, os dois com profile estreito e os
dois com a grant de alterar participantes. Sem a regra 5, Ana atribui o `Admin` a
Bruno, Bruno atribui o `Admin` a Ana, e os dois passam a ter tudo que a organização
tem sem ninguém acima aprovar.

A regra 5 não inventa mecanismo: atribuir um profile é **escolher um profile pelo
nome**, que é exatamente o que `resolveCeiling` e `resolveDefaultProfile` já fazem com
`IsSubsetOf` e 403. É o mesmo `IsSubsetOf` num terceiro lugar. Efeito colateral
desejado: só quem já tem tudo da organização consegue atribuir o `Admin`.

### As três escolhas de profile, e contra o que cada uma é conferida

A regra 5 vale sempre que alguém **escolhe um profile pelo nome**. Isso acontece em
três lugares, e hoje eles não usam o mesmo critério:

| Onde | Confere hoje contra | Decisão |
| --- | --- | --- |
| Atribuir profile a um participante | não existe rota | contra o que quem atribui tem |
| `POST /core/users_pool`, `default_profile_id` | o teto da organização | **muda**: contra o que quem pede tem |
| `POST /core/organizations`, `profile_id` | o teto da organização | **fica como está** — adiado, ver abaixo |

A diferença entre "o teto da organização" e "o que quem pede tem" é invisível hoje,
porque todo participante usa exatamente o mesmo profile que é o teto da sua
organização: no cadastro ele recebe o profile padrão do pool, que também vira o teto
da organização criada para ele, e no `POST /core/organizations` ele recebe o mesmo
profile que virou o teto. **É esta spec que abre a diferença**, ao permitir
participantes com profile mais estreito que o teto.

O que muda em `resolveDefaultProfile` é o segundo argumento da comparação, e a função
passa a receber a participação de quem pediu além da organização:

```go
ceiling, err := permissions.Resolve(
    caller.Organization.Profile.Permissions,
    caller.Participant.Profile.Permissions,
)

within, err := permissions.IsWithin(profile.Permissions, ceiling)
```

É o terceiro chamador de `IsWithin`, junto com o `POST` e o `PUT` de profile.

### Por que a criação de organização ficou de fora, por ora

O caso é o mesmo na forma e diferente no efeito. Exemplo: a organização ACME tem teto
largo, e a Ana participa dela só podendo ver e criar organizações. Hoje a Ana cria uma
organização escolhendo o teto largo da ACME, vira dona dela, e — com o profile `Admin`
da Fase 7 — passa a fazer lá tudo que a ACME faz, sem nunca ter podido fazer nada disso
na ACME.

Fechar isso do mesmo jeito que o pool tem um custo que o pool não tem: se uma
organização nova nunca puder ser mais larga do que quem a criou, delegar "criar
organização" a alguém estreito produz só organizações estreitas, e a pessoa encarregada
de montar organizações para os outros não consegue montar nenhuma útil. É uma decisão
de produto, não de segurança, e por isso está adiada.

**Quem for decidir precisa olhar junto o caminho sem `profile_id`.** Quando o payload
omite o campo, `resolveCeiling` devolve o profile padrão do pool com um `return`
antecipado e **não confere nada** — nem contra o teto da organização, nem contra
ninguém. É a única das três escolhas com um caminho sem verificação, e é mais
surpreendente do que a comparação em si: no pool principal do seed o padrão é o
`MANAGER_PROFILE`, então omitir o campo é hoje a forma mais curta de nascer com um teto
largo. `resolveDefaultProfile`, para pools, **não** tem esse buraco: ele confere nos
dois ramos.

### As rotas e as grants que faltam

Nenhuma das grants que estas regras mencionam existe no catálogo, porque nenhuma das
rotas existe. As três entram junto com as rotas que as tornam alcançáveis:

| Rota | Grant | Fase |
| --- | --- | --- |
| `GET /core/profiles` | `as::profiles::READ` | 4 |
| `POST /core/profiles` | `as::profiles::CREATE` | 4 |
| `GET /core/profiles/:id` | `as::profiles::READ` | 4 |
| `PUT /core/profiles/:id` | `as::profiles::UPDATE` | 4 |
| `PUT /core/organizations/:id/participants/:participant_id` | `as::organizations::participants::UPDATE` | **8, nova** |

Duas correções de nomenclatura em relação à conversa que originou esta fase:

- É **`as::profiles::`**, no plural. A convenção do catálogo é usar o nome do recurso
  como ele aparece na rota, e a rota é `/core/profiles` — a mesma razão de existir
  `as::users::READ` e não `as::user::READ`.
- Trocar o profile de um participante altera a tabela `participants`, não `users`,
  então é `as::organizations::participants::UPDATE` e não `as::users::UPDATE`. Ela
  acompanha a rota de listagem que já existe.

**A rota de trocar o profile de um participante não existe hoje** — nem ela, nem
qualquer outra que escreva em `participants` depois da criação. A regra 2 depende dela
para ter sentido: enquanto ninguém puder trocar o profile de ninguém, "você não troca o
seu" não descreve nada.

### Consequência: não há como transferir uma organização

Direto da regra 3. Como a participação do criador é congelada, não existe caminho para
passar uma organização a outra pessoa. Não é problema hoje, porque a funcionalidade não
existe. Quando alguém for construí-la, vai precisar de um fluxo próprio que mova
`organizations.owner_user_id` e a participação na mesma transação — e que decida o que
acontece com o dono antigo.

---

## Verificação

```bash
go build ./...
go test ./cmd/                 # fx.ValidateApp: IProfileService mudou de forma
make migrate                   # organization_id é nullable, não precisa de fresh
```

Depois, com as chaves de `credentials.txt`:

1. `GET /core/profiles` como admin → `ADMIN` (escopado a ele) + os três globais
2. `POST /core/profiles` `{"name": "Gestão de Vendas", "permissions": {...}}` →
   `201`, `key == "<admin_org_uuid>:GESTAO_DE_VENDAS"`
3. `POST /core/profiles` pedindo mais que o teto do caller → **`403`**, e nada
   gravado. A mensagem diz o que excedeu
4. `PUT` nesse profile alargando → `403` pelo mesmo motivo
5. `PUT` em `MANAGER_PROFILE` (global) → `403`
6. Registrar um usuário num app de outro pool, logar, `GET /core/profiles` → só os
   globais, o profile da admin_org **não** aparece
7. `POST /core/organizations` com `profile_id` do profile escopado da admin_org,
   autenticado como o usuário do passo 6 → `400`, profile não encontrado. **Esta é a
   prova de que o choke point fechou**
8. `POST /core/profiles` com `{"name": "!!!"}` → `400`, não uma chave `<uuid>:`

Da Fase 7:

9. Signup num app de um pool cujo default é `LOGIN_PROFILE` → nasce um profile
   `<org_uuid>:ADMIN` escopado à organização nova, e `participants.profile_id` aponta
   para ele, **não** para `LOGIN_PROFILE`
10. A resposta do signup traz `user.profile.permissions.grants` com os grants
    concretos do `LOGIN_PROFILE`, não `["as::*::*"]`
11. `POST /core/organizations` → mesma coisa para a organização criada à mão
12. Subir o teto daquela organização (trocar `organizations.profile_id` por um profile
    mais largo, direto no banco) → o dono passa a alcançar as rotas novas **sem**
    tocar na participação. É a prova de que o curinga acompanha o teto
13. Login como o usuário admin do seed → `GET` numa rota `/core` que não tenha entrada
    no catálogo de grants continua `200`. Prova que a `admin_organization` ficou de
    fora da regra

Da Fase 8:

14. `PUT /core/profiles/:id` no próprio profile do caller → `403`, mesmo com
    `as::profiles::UPDATE`
15. `PUT` no profile `<org_uuid>:ADMIN` → `403`, por qualquer um, inclusive o dono
16. Trocar o profile do próprio participante → `403`
17. Trocar o profile do participante que criou a organização → `403`, inclusive
    quando quem pede é ele mesmo
18. Atribuir a outro participante um profile mais largo do que o caller tem → `403`.
    **É a prova de que a promoção mútua fechou**
19. O admin pleno atribuindo a outro participante um profile que cabe no que ele tem
    → `200`
20. `POST /core/users_pool` com `default_profile_id` de um profile que cabe no teto da
    organização mas **não** no que quem pediu tem → `403`. Precisa de um participante
    com profile mais estreito que o teto, que é o que esta spec passa a permitir
21. O mesmo `POST /core/users_pool` sem `default_profile_id`, feito por esse mesmo
    participante estreito → `403` se o `LOGIN_PROFILE` não couber no que ele tem. O
    ramo sem id também confere

---

## Pontos em aberto

1. **`DELETE /core/profiles/:id` não entra.** Não há FK declarada de
   `participants.profile_id` para `profiles`, então um delete deixaria linhas com
   `profile_id` pendurado e o `OrganizationGuard` responderia 500 em vez de negar.
   Antes de haver delete: ou FK com `ON DELETE RESTRICT`, ou o service recusa quando
   existe participante/organização/pool apontando para a linha.
2. ~~**Forma do `permissions` gravado**~~ — **fechado em 2026-09-06.** Com a Fase 3
   recusando em vez de clampar, o que é gravado é o payload tal como veio, e não
   existe `Resolved` para converter. A divergência entre `Resolved.Query`
   (`map[string][]string`) e `Document.Query` (`map[string]string`) deixa de aparecer
   no caminho de escrita.
3. **Subir o teto de uma organização não sobe quem participa nela** — herdado da
   feature de organizações, e piora aqui: com profiles escopados customizados, mais
   linhas ficam pinadas a um teto antigo. **Resolvido para o dono** pela Fase 7: o
   profile de admin é curinga e colapsa no teto qualquer que ele seja. Continua aberto
   para todo o resto.
4. **Um profile escopado usado como `users_pool.default_profile_id`.** Discutido e
   aceito: a organização filha fica com um teto que não enxerga. A identidade da
   linha não vaza (`Organization.Profile` é `json:"-"`, o que aparece é o resolvido)
   e o efeito prático é forçar a org filha a criar os próprios profiles.
5. **Sem `organization_id` no listing de outra org, nem para ADMIN.** Não existe
   lente cross-org — mesma limitação já registrada na feature de organizações.
6. ~~**O profile de admin não é protegido.**~~ — **fechado em 2026-09-06.** Ele é
   travado: `PUT /core/profiles/:id` recusa na linha `<org_uuid>:ADMIN`, e o documento
   fica em `as::*::*` para sempre, que é o que o mantém acompanhando o teto. Não existe
   "admin desta organização menos apps" — quem quer isso cria um profile novo e o
   atribui. É a regra 6 da Fase 8.
7. **Organização deletada deixa o profile de admin órfão.** Coberto pelo ponto nº 1,
   que já exige resolver FK e delete antes de qualquer `DELETE`.
8. ~~**Formato do corpo de `POST` e `PUT /core/profiles`**~~ — **fechado em
   2026-09-06.** Um campo `permissions`, objeto, aceitando só `grants`. `api` no corpo
   é `400`. Ver a Fase 4.
9. ~~**`GetProfilesQuery` e `GetProfilesResponse` não têm campos definidos.**~~ —
   **fechado em 2026-09-06**, com `organization_id` além da paginação. Ver a Fase 4.
10. **Os corpos de requisição antigos ainda usam `...Payload`.** A regra de nomes mudou
    em 2026-09-06 e `CreateProfile` já a segue, mas renomear os dez tipos que existem é
    cerca de noventa ocorrências em vinte e cinco arquivos. Fica para uma passada
    própria, ou não é feito.
11. **Contra o que conferir o teto de uma organização nova — adiado.** Decidido em
   2026-09-06 que o `default_profile_id` de um pool passa a ser conferido contra o que
   quem pede tem, e que o `profile_id` de `POST /core/organizations` fica como está
   por enquanto. O exemplo da Ana e o custo de produto de fechar estão na Fase 8. Quem
   retomar precisa decidir junto o caminho em que `profile_id` é omitido, que hoje não
   passa por verificação nenhuma.

---

## O que NÃO muda

Para o próximo agente não sair procurando: `permissions.Resolve`, `IsSubsetOf`,
`Document`, `Resolved` e o `PermissionsGuard` ficam **intocados**. A hierarquia
`pool → organization → participant` continua a mesma. Esta frente é uma coluna, um
choke point de visibilidade, uma função de texto, cinco rotas, os dois verbos
recusando o que excede o caller, uma linha de profile no nascimento de cada
organização e as seis regras de quem escreve permissão de quem.

O que ela **acrescenta** a `shared/permissions` é uma função só: `IsWithin`, que é o
`withinApi` já existente exportado com um `*Resolved` como teto.
