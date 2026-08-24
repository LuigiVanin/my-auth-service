# Feature: Profiles escopados por organização — plano

> **Estado deste documento:** plano, nada implementado. Escrito na sessão que
> entregou [2026-08-23-organizations.md](2026-08-23-organizations.md), com as
> decisões já fechadas com o autor do projeto.
>
> **Leia antes:** [docs/steering/modules/profiles.md](../../docs/steering/modules/profiles.md) — o modelo de
> permissão, a hierarquia e a distinção clamp vs recusar já estão lá. Este plano
> adiciona escopo e um endpoint; não muda `permissions.Resolve` nem `IsSubsetOf`.

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
| Clamp | Em `POST` **e** `PUT`. `permissions.Resolve(tetoDaOrg, profileDoParticipante, requisitado)` |
| Visibilidade | Um único choke point: `FindByIdVisibleTo(id, organizationId)` |
| Global vs aplicável | Global significa **enxerga**. Quem decide se pode aplicar é o clamp, sempre |
| `ADMIN` | Passa a ser escopado à `admin_organization`, num passo novo do seed |
| `ParentProfileId` | **Removido** da entidade e da migração |
| Criação de profile global | Não existe pela API. Global é só seed |
| `DELETE /core/profiles/:id` | **Fora de escopo** — ver "Pontos em aberto" |

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
// dele nela. As duas metades são necessárias - ver o clamp abaixo.
type ProfileWriteContext struct {
    Organization *entity.Organization
    Participant  *entity.Participant
}
```

### O clamp, nos dois verbos

Uma chamada só, porque `Resolve` é variádico com fold da esquerda:

```go
resolved, err := permissions.Resolve(
    caller.Organization.Profile.Permissions,  // teto da organização
    caller.Participant.Profile.Permissions,   // o que o participante tem nela
    payload.Permissions,                      // o que ele está pedindo
)
```

Não é `Resolve(Resolve(a,b), c)` — `Resolve` devolve `*Resolved` e consome
`json.RawMessage`, então o encadeamento não tipa. Os três documentos numa chamada
dão o mesmo resultado.

O que é gravado é o **resolvido**, não o requisitado. Clampar em vez de recusar é a
regra já documentada em `docs/steering/modules/profiles.md`: escolher um profile nomeado que não
cabe é erro que merece 403; **autorar** um documento é pedido para ser limitado.

`Resolved` precisa ser serializado de volta para `json.RawMessage` antes de gravar
em `profiles.permissions`. Atenção: `Resolved.Query` é `map[string][]string` e
`Document.Query` é `map[string]string`. Gravar o `Resolved` cru mudaria a forma do
documento armazenado e `Parse` passaria a falhar naquela linha. **Ou** se converte
para a forma de `Document` na gravação (colapsando listas de um padrão só, e
recusando o caso de dois padrões concorrentes), **ou** `Document.Query` passa a
aceitar as duas formas com um `UnmarshalJSON` customizado. Decidir antes de
escrever a Fase 3 — é o detalhe que mais provavelmente vai morder.

### O update

- `Permissions` nil no payload → não toca, não clampa.
- `Permissions` preenchido → clampa contra o teto **atual**.

Consequência aceita: se o teto da organização estreitou desde a criação, um update
que só quer mudar permissões **reduz** a linha para o teto de hoje. É correto — não
se pode manter mais do que o teto concede — mas surpreende, e vale a resposta dizer
o que foi gravado (ela já devolve o profile).

Um update só é possível sobre profile **escopado à organização do caller**. Global é
seed: `FindByIdVisibleTo` acha `MANAGER_PROFILE`, e o service tem que recusar
`organization_id == nil` com 403 antes de escrever.

---

## Fase 4 — Rotas

Grupo `/core`, cadeia `authGuard → organizationGuard → permissionsGuard`.

| Método | Path | Comportamento |
| --- | --- | --- |
| GET | `/core/profiles` | Página dos globais + os da organização atual. `permissions` cru, ver nota |
| POST | `/core/profiles` | Cria escopado à organização atual. Body `{name, permissions, metadata?}`. `201` |
| GET | `/core/profiles/:id` | Um profile visível. `404` fora do escopo |
| PUT | `/core/profiles/:id` | Só escopado à organização atual. `403` em global |

**Nota sobre `permissions` cru na listagem.** A regra de
`docs/steering/modules/profiles.md` é que respostas carregam documento resolvido, porque um
profile de participante lido sozinho engana. Aqui é diferente e de propósito: o
profile **é** o assunto da resposta, não uma camada debaixo de outra. Quem lista
profiles está escolhendo qual aplicar, e precisa ver o documento como ele é. A
seção "What is exposed" do steering precisa ganhar essa exceção explicitamente.

Documentos de permissão do seed: `/core/profiles` e `/core/profiles/:id` entram no
`MANAGER_PROFILE`. Não entram no `LOGIN_PROFILE`. O guard nega query param não
declarado, então `skip`/`limit` têm que constar.

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

---

## Fase 6 — Fechar o choke point

A parte que decide se a feature é segura ou decorativa. Todo lugar que aceita um
`profile_id` de fora tem que passar por `FindByIdVisibleTo`. Hoje são dois, e ambos
usam o `FindById` irrestrito:

| Arquivo | Chamada |
| --- | --- |
| `user_pool.service.go`, `resolveDefaultProfile` | `FindById(profileId)` → escopado |
| `organization.service.go`, `resolveCeiling` | `FindById(profileId)` → escopado |

Os dois já recebem a organização do caller no contexto, então é troca de assinatura,
não de arquitetura. **Remover `FindById` da interface** em vez de deixá-lo ao lado do
escopado: enquanto ele existir, o próximo caller vai usar o mais curto.

Os três pontos onde um `profile_id` é gravado — `participants.profile_id`,
`organizations.profile_id`, `users_pool.default_profile_id` — não recebem id de
payload em nenhum fluxo atual (vêm do teto ou do pool). Quando o convite chegar,
`participants.profile_id` **vai** receber, e é o próximo lugar a exigir o escopo.

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
3. `POST /core/profiles` pedindo mais que o teto do caller → `201` com as permissões
   **clampadas**, não `403`. Conferir o corpo, não o status
4. `PUT` nesse profile alargando → `200` com o resultado clampado de novo
5. `PUT` em `MANAGER_PROFILE` (global) → `403`
6. Registrar um usuário num app de outro pool, logar, `GET /core/profiles` → só os
   globais, o profile da admin_org **não** aparece
7. `POST /core/organizations` com `profile_id` do profile escopado da admin_org,
   autenticado como o usuário do passo 6 → `400`, profile não encontrado. **Esta é a
   prova de que o choke point fechou**
8. `POST /core/profiles` com `{"name": "!!!"}` → `400`, não uma chave `<uuid>:`

---

## Pontos em aberto

1. **`DELETE /core/profiles/:id` não entra.** Não há FK declarada de
   `participants.profile_id` para `profiles`, então um delete deixaria linhas com
   `profile_id` pendurado e o `OrganizationGuard` responderia 500 em vez de negar.
   Antes de haver delete: ou FK com `ON DELETE RESTRICT`, ou o service recusa quando
   existe participante/organização/pool apontando para a linha.
2. **Forma do `permissions` gravado** — o ponto de decisão da Fase 3, entre
   converter `Resolved` para a forma de `Document` na gravação ou fazer
   `Document.Query` aceitar as duas formas.
3. **Subir o teto de uma organização não sobe quem participa nela** — herdado da
   feature de organizações, e piora aqui: com profiles escopados customizados, mais
   linhas ficam pinadas a um teto antigo.
4. **Um profile escopado usado como `users_pool.default_profile_id`.** Discutido e
   aceito: a organização filha fica com um teto que não enxerga. A identidade da
   linha não vaza (`Organization.Profile` é `json:"-"`, o que aparece é o resolvido)
   e o efeito prático é forçar a org filha a criar os próprios profiles.
5. **Sem `organization_id` no listing de outra org, nem para ADMIN.** Não existe
   lente cross-org — mesma limitação já registrada na feature de organizações.

---

## O que NÃO muda

Para o próximo agente não sair procurando: `permissions.Resolve`, `IsSubsetOf`,
`Document`, `Resolved` e o `PermissionsGuard` ficam **intocados**. A hierarquia
`pool → organization → participant` continua a mesma. Esta frente é uma coluna, um
choke point de visibilidade, uma função de texto, quatro rotas e os dois verbos
clampando.
