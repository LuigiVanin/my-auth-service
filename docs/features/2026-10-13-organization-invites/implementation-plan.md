---
status: draft
context: organization-invites
created: 2026-10-13
updated: 2026-10-13
implemented:
drivers: >
  Nenhuma organização consegue trazer um usuário para dentro: só se entra numa
  criando-a ou nascendo nela. O convite é o que faz uma organização ter mais de um
  participante de verdade, e é a primeira rota que cria participação fora do
  provisionamento — por isso herda, de uma vez, o teto de quem convida e o escopo
  por pool.
---

# Implementation Plan: organization-invites

> **Esqueleto.** Cresce decisão por decisão durante a implementação. Toda decisão
> abaixo cita a fonte; as que dependem de uma resposta às `D-*` de
> [requirements.md](requirements.md) estão marcadas.

## Objetivo

Entregar o par criar/aceitar: `POST /core/organizations/{id}/invites` cria o
convite, guarda só o hash do segredo e envia o link; `GET /auth/invite` descreve o
convite ao portador do token; `POST /auth/invite/accept` cria a participação para
quem já tem conta na pool; `POST /auth/register?method=invite` cadastra na pool e
cria a participação para quem não tem. Tabela nova `invites`, grant novo, módulo
OTP intocado.

## Decisões tomadas

### Arquitetura

- **`invite` é um módulo core**, em `app/modules/core/invite/`, com entidade,
  repositório, serviço e controller. Possui tabela, logo é core; tem controller
  porque expõe rotas, como `otp` e `user` (fonte: `steering/module-structure.md`,
  "The feature owns a database table → core module").

- **O serviço é exportado** (sem `fx.Private`), porque `register` o consome para o
  cadastro por convite. O repositório é privado (fonte: regra "a service other
  modules consume is provided without fx.Private"). *Depende de D-2.*

- **O aceite é uma unidade de trabalho dentro do serviço de convite**: criar a
  participação, consumir o convite e, quando for o caso, marcar o e-mail
  verificado — três escritas, uma transação. Por isso `InviteService` segura
  `IParticipantRepository` e `IUserRepository` para essas escritas, e continua
  lendo pelos serviços (`IParticipantService.FindForUserInOrganization`,
  `IProfileService.FindByIdVisibleTo`, `IUserService.FindUserInPool`). É a tabela
  de três casos de `steering/service-layer.md`; `writeEmailChange` é o precedente
  mais próximo (fonte: leitura de `authorize.service.go`).

- **`IParticipantService.Create` continua sem chamador.** A escrita acontece
  dentro da transação e vai ao repositório; a pendência 16 não fecha aqui (fonte:
  mesma tabela de três casos).

- **O cadastro por convite vive em `RegisterService.RegisterWithInvite`**, que
  chama `IInviteService.Resolve(token, app)` para validar e ler, `ProvisionUser`
  para criar, e `IInviteService.Accept(invite, user)` para a participação. A
  direção é api → core; o contrário exporia `ProvisionUser` a um módulo core
  (fonte: `steering/module-structure.md`, "Depending on another module"). *Depende
  de D-2.*

- **O token é 32 bytes de `crypto/rand` em base64url; o banco guarda
  `sha256(token)` em hex.** A busca é pelo hash, sem id na URL. `IHashService`
  (argon2) fica para senhas e códigos curtos, que são o que precisa de hash lento
  (fonte: RC-9; a alternativa `{id}.{segredo}` com argon2 espelharia o OTP e
  custaria um id a mais na URL sem ganho).

- **O e-mail é enviado dentro da requisição, depois do `INSERT`**, pelo
  `EmailManager`, como `GenerateConsumable` faz. Falha no envio responde `500` e a
  linha fica pendente; o próximo `POST` a substitui (fonte: leitura de
  `otp.service.go`; RF-8).

### Contrato

- **Entidade `infra/entities/invite.entity.go`**, tabela `invites`:

  | Coluna | Tipo | Nota |
  | --- | --- | --- |
  | `id` | uuid | |
  | `organization_id` | uuid, índice | FK `organizations` |
  | `profile_id` | uuid | FK `profiles`; visível à organização (RC-1) |
  | `invited_by_user_id` | bigint | FK `users`; só auditoria e o nome no e-mail |
  | `email` | text, índice | minúsculo, como `users.email` |
  | `token_hash` | text, único | `json:"-"` (RC-13) |
  | `expires_at` | timestamp | |
  | `accepted_at` | timestamp, nulo | escrita só por `Accept` |
  | `revoked_at` | timestamp, nulo | escrita só por `RevokePending…`; nasce para D-5 |
  | `created_at`, `updated_at` | | |

  Relações: `Organization` (`json:"-"`), `Profile` (serializado, como em
  `Participant`), `InvitedBy` (`json:"-"`). Tags `json` desde o primeiro dia, para
  não repetir a pendência 13.

- **`InviteUpdateDao` não tem campo.** Nenhuma coluna é editável por payload; as
  duas transições são métodos dedicados com predicado em SQL (RC-10, RC-13).

- **Repositório** — além de `Create` e `FindOne`:

  | Método | Faz |
  | --- | --- |
  | `FindPendingByTokenHash(hash, options)` | `token_hash = ? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now()`, com `Organization` e `Profile` |
  | `Accept(id, at, options) (int64, error)` | `UPDATE … SET accepted_at WHERE id = ? AND accepted_at IS NULL AND revoked_at IS NULL`; `0` linhas é a corrida perdida |
  | `RevokePendingByOrganizationAndEmail(orgId, email, at, options)` | o "substituir" do RF-8 |

  Cada um ganha um DryRun em `tests/shared/dedicated_queries_test.go` e uma linha
  em `steering/repository-pattern.md`.

- **Rotas**:

  | Rota | Guards | Grant | Resposta |
  | --- | --- | --- | --- |
  | `POST /core/organizations/{id}/invites` | cadeia completa | `as::organizations::invites::CREATE` | `201` convite, sem token |
  | `GET /auth/invite?token=` | `AppGuard` | `as::invite::READ` (expressivo) | `200` descrição |
  | `POST /auth/invite/accept` | `AppGuard`, `AuthGuard` | `as::invite::accept::CREATE` (expressivo) | `200 { organization, participant }` em views |
  | `POST /auth/register?method=invite` | `AppGuard` | `as::register::CREATE` (já existe) | `201 RegisterResponse` |

  As duas de `/auth` não carregam `EmailVerificationGuard`, como nenhuma rota de
  `/auth` carrega — e o aceite é justamente o que verifica (RC-8). `OrganizationGuard`
  e `PermissionsGuard` ficam fora do aceite de propósito: eles avaliariam a
  participação do chamador na organização **atual**, que não tem nada a ver com
  aceitar um convite para outra (fonte: leitura de `organization.guard.go` e
  `permissions.guard.go`).

- **Corpo de `POST …/invites`**: `{ "email": string, "profile_id": uuid }`, nome
  `CreateInvite` sem sufixo (fonte: `steering/models-layer.md`, "Naming").

- **Resposta de `GET /auth/invite`** é um DTO mínimo, não a entidade: organização
  (`id`, `name`, `avatar_url`), `invited_by` (só o nome), `email`, profile (`id`,
  `name`, `key`), `expires_at`, `account_exists`. Um portador de token não
  autenticado não recebe `metadata`, `owner_user_id` nem o perfil completo de
  quem convidou (fonte: RF-6, RNF-7).

- **Corpo de `register?method=invite`**: `{ "token", "name", "password"?, "phone"?,
  "metadata"? }`. Sem `email`: vem do convite (RC-11). `RegisterValidator` ganha o
  `case "invite"`.

- **Erros, nenhum código novo.** `404` para token desconhecido ou de outra pool;
  `400` para expirado, consumido ou substituído, com o `detail` dizendo qual, como
  o OTP; `403` para outra organização na criação e para e-mail diferente no aceite;
  `409` para quem já participa, na criação; `501` sem `INVITE_ACCEPT_URL` (fonte:
  `steering/error-handling.md`, "would a client write a different branch?" — a
  página de aceite mostra a mesma tela para os três `400`).

- **Tag OpenAPI nova, `Invites`**, para as três rotas do módulo; a de `register`
  continua em `Auth`.

### Permissões

- Três chaves no catálogo: `as::organizations::invites::CREATE`, `as::invite::READ`,
  `as::invite::accept::CREATE`. A primeira é a que importa: é alcançada por todo
  dono de organização no instante em que entra, clampada pelo teto (fonte:
  `steering/modules/profiles.md`, "adding a map entry is a permission change").
- Profiles semeados: `MANAGER_PROFILE` recebe as três. `LOGIN_PROFILE` recebe as
  duas de `/auth`; a de `/core` *depende de D-3*.
- `steering/modules/profiles.md` ganha o parágrafo das três chaves, e registra que
  o convite é o segundo lugar, depois de `UpdateParticipant`, que entrega um
  profile a alguém e por isso compara contra `Resolve(teto, participação)`.

### Configuração

- Bloco `invite` em `config.Config`: `accept_url` (`INVITE_ACCEPT_URL`, template
  com `{token}` e `{organization_id}`) e `ttl_seconds` (`INVITE_TTL_SECONDS`,
  padrão 604800). Nos dois arquivos de exemplo de `env/` (fonte: formato do bloco
  `avatar`; D-6, D-7).
- Remetente: *depende de G-3* — literal repetido ou `EMAIL_FROM` no bloco `email`.

### Testes

- `tests/modules/invite/`: criação — profile fora do teto `403`, outro id `403`,
  já participa `409`, substituição antes do `INSERT`, envio depois do `INSERT`,
  falha de envio `500`; resolução — desconhecido `404`, outra pool `404`, expirado,
  consumido e substituído `400`; aceite — e-mail diferente `403`, participação
  criada e convite consumido na mesma transação, já participa `200` sem mudar
  profile, `verify_email` vira `true` quando era `false`, corrida perdida `400`.
- `tests/modules/register/`: `RegisterWithInvite` — regra de senha por
  `login_types`, e-mail já cadastrado `409`, provisiona e depois aceita, sessão
  devolvida.
- `tests/shared/dedicated_queries_test.go`: o SQL dos três métodos dedicados.
- `tests/modules/routes/`: ordem das rotas e o teste de views sobre o OpenAPI.
- `cmd/main_test.go`, obrigatório: três construtores novos no grafo.
- Mocks: `MockInviteRepository`, `MockInviteService`; `MockRegisterService` ganha
  `RegisterWithInvite`.

## Dependências

- Nenhuma biblioteca nova: `crypto/rand`, `crypto/sha256`, `encoding/base64`.
- Migração: `&entity.Invite{}` nas duas listas de `AutoMigrate`; nenhum enum.
- Seed: só se D-3 alterar `LOGIN_PROFILE` — `make fresh`.
- Handoff em `docs/handoff/2026-10-13-organization-invites.md`: a página de
  aceite, `replaceState` (G-5), `core-api.md` §4 e §9 e `ParticipantsPage.tsx:112`
  deixam de ser verdade.

## Pendências a deixar (vão para `pendencias.md` ao implementar)

- Limite de taxa de envio — G-1.
- Endereço de aceite por aplicação — G-2.
- Dois pendentes por corrida de `POST` — G-6.
- `ProvisionUser` fora de transação externa — G-7.
- `math/rand` nos geradores de `shared/utils/random.go` — G-8.
- Listar e revogar convites — D-5.

## O que divergiu do plano

Vazio até a implementação.
