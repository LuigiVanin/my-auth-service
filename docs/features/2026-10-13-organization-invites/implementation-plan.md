---
status: implemented
context: organization-invites
created: 2026-10-13
updated: 2026-11-15
implemented: 2026-11-15
drivers: >
  Nenhuma organização consegue trazer um usuário para dentro: só se entra numa
  criando-a ou nascendo nela. O convite é o que faz uma organização ter mais de um
  participante de verdade, e é a primeira rota que cria participação fora do
  provisionamento — por isso herda, de uma vez, o teto de quem convida e o escopo
  por pool.
---

# Implementation Plan: organization-invites

> Cresce decisão por decisão durante a implementação. Toda decisão cita a fonte;
> as `D-*` e `RC-*` são as de [requirements.md](requirements.md).

## Objetivo

Entregar criar, listar, revogar e aceitar: `POST`, `GET` e `DELETE` em
`/core/organizations/{id}/invites`; `GET /auth/invite` descreve o convite ao
portador do token; `POST /auth/invite/accept` cria a participação para quem já tem
conta na pool, só com o token; `POST /auth/register?method=invite` cadastra na pool
e cria a participação para quem não tem. O segredo é um OTP de ação `INVITE`.

## Decisões tomadas

### Arquitetura

- **`invite` é um módulo core**, em `app/modules/core/invite/`, com entidade,
  repositório, serviço e controller. Possui tabela, logo é core; tem controller
  porque expõe rotas, como `otp` e `user` (fonte: `steering/module-structure.md`,
  "The feature owns a database table → core module").

- **O serviço é exportado** (sem `fx.Private`), porque `register` o consome para o
  cadastro por convite. O repositório é privado (fonte: "a service other modules
  consume is provided without fx.Private"; D-2).

- **O token é um OTP.** `IOtpService.Issue(action, app, contact, ttl)` cria a linha
  com um segredo de 32 bytes de `crypto/rand` em base64url no lugar do código, sem
  rate limit e sem e-mail, e devolve `IssuedOtp{Otp, Secret}`. Hasheia como
  `GenerateConsumable`: argon2 fora de `development` (fonte: D-1, leitura de
  `otp.service.go`). O nome é genérico de propósito — não sabe o que é convite.

- **O link leva `<otp_id>.<segredo>`.** O argon2 é salgado, então a linha não é
  procurável pelo hash; o id a localiza e o segredo a prova (fonte: RNF-2).

- **A conferência é do OTP, o estado é do convite.** `Resolve(token, app)` lê o
  convite por `otp_id` com o OTP preloaded, recusa outra aplicação com `404`,
  confere aceito, revogado e expirado no convite com mensagens próprias, e só então
  chama `ValidateConsumable(INVITE)` para o segredo — recusa dele que não seja `500`
  vira `404` (fonte: RC-3, RC-9; G-10 registra o custo da ordem).

- **O aceite é uma unidade de trabalho dentro do serviço de convite**: criar a
  participação (se ainda não existe), marcar o convite aceito com predicado de
  pendente e, quando for o caso, marcar o e-mail verificado — uma transação.
  `InviteService` segura `IParticipantRepository` e `IUserRepository` para essas
  escritas e lê pelos serviços. `Invalidate` do OTP vem depois do commit, detached,
  como em `RegisterWithOtp` (fonte: tabela de três casos de
  `steering/service-layer.md`; RNF-3).

- **`AcceptFor(invite, user)` é o núcleo comum.** `Accept(token, app)` o usa para
  quem já tem conta; `RegisterService.RegisterWithInvite` o usa depois de
  `ProvisionUser`. A direção é api → core (fonte: D-2,
  `steering/module-structure.md`, "Depending on another module").

- **A criação não é transação.** O OTP é emitido pelo serviço do OTP, que não aceita
  `repo.Option`; um OTP órfão de uma inserção que falhou não é alcançável por
  nenhum convite e expira sozinho (fonte: leitura de `IOtpService`).

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
  | `invited_by_user_id` | bigint | FK `users`; auditoria e o nome no e-mail |
  | `email` | text, índice | minúsculo, como `users.email` |
  | `otp_id` | uuid, único | FK `otps`; `json:"-"` (RC-13) |
  | `expires_at` | timestamp | cópia imutável da do OTP (RC-15) |
  | `accepted_at` | timestamp, nulo | escrita só por `Accept` |
  | `revoked_at` | timestamp, nulo | escrita só por `Revoke` |
  | `created_at`, `updated_at` | | |

  Relações: `Organization` (`json:"-"`), `Profile` (serializado, como em
  `Participant`), `InvitedBy` (`json:"-"`), `Otp` (`json:"-"`). Tags `json` desde o
  primeiro dia, para não repetir a pendência 13.

- **Repositório `repo.NoUpdate`.** Nenhuma coluna é editável por payload; as duas
  transições são métodos dedicados com predicado em SQL (RC-10, RC-13).

  | Método | Faz |
  | --- | --- |
  | `FindOne(where, options)` | por `otp_id` no aceite, por `id` + `organization_id` na revogação |
  | `FindForOrganization(orgId, status, options)` / `…Count` | a listagem, com o predicado do estado em SQL |
  | `FindPendingFor(orgId, email, options)` | os pendentes que um convite novo substitui |
  | `Accept(id, at, options) (int64, error)` | `SET accepted_at WHERE id = ? AND` pendente; `0` é a corrida perdida |
  | `Revoke(id, at, options) (int64, error)` | `SET revoked_at WHERE id = ? AND` pendente |

- **Rotas**:

  | Rota | Guards | Grant | Resposta |
  | --- | --- | --- | --- |
  | `POST /core/organizations/{id}/invites` | cadeia completa | `as::organizations::invites::CREATE` | `201` convite, sem token |
  | `GET /core/organizations/{id}/invites` | cadeia completa | `as::organizations::invites::READ` | `200` página |
  | `DELETE /core/organizations/{id}/invites/{invite_id}` | cadeia completa | `as::organizations::invites::DELETE` | `200` convite revogado |
  | `GET /auth/invite?token=` | `AppGuard` | `as::invite::READ` (expressivo) | `200` descrição |
  | `POST /auth/invite/accept` | `AppGuard` | `as::invite::accept::CREATE` (expressivo) | `200 { organization, participant }` em views |
  | `POST /auth/register?method=invite` | `AppGuard` | `as::register::CREATE` (já existe) | `201 RegisterResponse` |

  O aceite não tem `AuthGuard` (D-2), nem `OrganizationGuard` e `PermissionsGuard`,
  que avaliariam a organização **atual** do chamador.

- **Corpo de `POST …/invites`**: `{ "email", "profile_id", "ttl_seconds"? }`, nome
  `CreateInvite` sem sufixo; `ttl_seconds` com `omitnil,min=3600,max=2592000`
  (fonte: `steering/models-layer.md`, "Naming"; RC-14).

- **Resposta de convite** é `InviteResponse`: a entidade mais `status`
  (`pending`, `accepted`, `revoked`, `expired`), calculado do mesmo predicado da
  listagem.

- **Resposta de `GET /auth/invite`** é um DTO mínimo: organização (`id`, `name`,
  `avatar_url`), `invited_by` (só o nome), `email`, profile (`id`, `name`, `key`),
  `expires_at`, `account_exists`. Um portador de token não autenticado não recebe
  `metadata`, `owner_user_id` nem o perfil completo de quem convidou (RF-6, RNF-7).

- **Corpo de `register?method=invite`**: `{ "token", "name", "password"?,
  "phone"?, "metadata"? }`. Sem `email`: vem do convite (RC-11).
  `RegisterValidator` ganha o `case "invite"`.

- **Erros, nenhum código novo.** `404` para token malformado, desconhecido, de outra
  aplicação ou de segredo errado; `400` para expirado, aceito ou revogado, com o
  `detail` dizendo qual; `403` para outra organização em `/core` e para profile
  acima de quem convida; `409` para quem já participa, na criação, e para revogar o
  que não está pendente; `409 USER_ALREADY_EXISTS` no cadastro por convite de
  e-mail que já tem conta; `404` no aceite de e-mail sem conta; `501` sem
  `INVITE_ACCEPT_URL` (fonte: `steering/error-handling.md`, "would a client write a
  different branch?").

- **Tag OpenAPI nova, `Invites`**, para as cinco rotas do módulo; a de `register`
  continua em `Auth`.

- **`organizations.invite_ttl_seconds`**, `*int64`, nulo por padrão, no dao e em
  `UpdateOrganization` com os limites do RC-14. Aparece em toda resposta de
  organização pela view, que embute a entidade (D-7; G-11).

### Permissões

- Cinco chaves no catálogo: `as::organizations::invites::CREATE`, `…::READ`,
  `…::DELETE`, `as::invite::READ`, `as::invite::accept::CREATE`. As três de `/core`
  são alcançadas por todo dono de organização com teto que as contenha — Manager e
  acima (fonte: `steering/modules/profiles.md`, "adding a map entry is a permission
  change"; D-3).
- Profiles semeados: `MANAGER_PROFILE` recebe as cinco. `LOGIN_PROFILE` recebe as
  duas de `/auth`.

### Configuração

- Bloco `invite` em `config.Config`: `accept_url` (`INVITE_ACCEPT_URL`, template
  com `{token}` e `{organization_id}`) e `ttl_seconds` (`INVITE_TTL_SECONDS`,
  padrão 604800). Nos dois arquivos de exemplo de `env/` (fonte: formato do bloco
  `avatar`; D-6, D-7).
- Remetente: o literal de `OtpService`, repetido; G-3 vai para `pendencias.md`.

### Migração

- `&entity.Invite{}` nas duas listas de `AutoMigrate`, depois de `Otp`.
- `addEnumValues` roda `ALTER TYPE auth_action ADD VALUE IF NOT EXISTS` para cada
  ação depois de `createEnum`, que passa a não reescrever o slice que recebe (D-1).

### Testes

- `tests/modules/invite/`: criação, resolução, aceite, listagem e revogação.
- `tests/modules/register/`: `RegisterWithInvite`.
- `tests/shared/dedicated_queries_test.go`: o SQL dos métodos dedicados.
- `tests/modules/routes/`: ordem das rotas e o teste de views sobre o OpenAPI.
- `cmd/main_test.go`: construtores novos no grafo.
- Mocks: `MockInviteRepository`, `MockInviteService`; `MockOtpService.Issue`;
  `MockRegisterService.RegisterWithInvite`.

## Dependências

- Nenhuma biblioteca nova: `crypto/rand`, `encoding/base64`.
- Migração: tabela `invites`, coluna `organizations.invite_ttl_seconds`, valor
  `INVITE` em `auth_action`.
- Seed: `MANAGER_PROFILE` e `LOGIN_PROFILE` mudam — `make fresh` ou `make seed`.
- Handoff em `docs/handoff/2026-11-15-organization-invites.md`.

## Pendências a deixar (vão para `pendencias.md` ao implementar)

- Limite de taxa de envio — G-1.
- Endereço de aceite por aplicação — G-2.
- Remetente literal em dois serviços — G-3.
- Dois pendentes por corrida de `POST` — G-6.
- `ProvisionUser` fora de transação externa — G-7.
- `math/rand` nos geradores de `shared/utils/random.go` — G-8.
- `invite_ttl_seconds` sem caminho de volta a `NULL` — G-11.

## O que divergiu do plano

Tudo abaixo foi descoberto durante a implementação; nenhum item muda uma decisão
`D-*`.

- **`createEnum` reescrevia o slice que recebe.** Punha aspas dentro de
  `constants.AuthActions`, então `addEnumValues`, chamado logo depois, receberia
  `''INVITE''`. Passou a citar numa cópia (fonte: leitura de
  `cmd/database/migration/main.go` ao escrever o passo novo).

- **O segredo sai de `utils.GenerateSecureToken`, novo, em `shared/utils/random.go`.**
  Os dois geradores que já estavam lá usam `math/rand` (G-8, item 42 de
  `pendencias.md`); o novo usa `crypto/rand` e devolve base64url sem padding.

- **A emissão exige que a pool da aplicação seja a da organização** (`403`). O plano
  presumia; o código não garantia, e sem isso o convidado seria procurado numa pool e
  o OTP pertenceria a uma aplicação de outra (fonte: RC-3).

- **`RegisterService.openSession` foi extraído.** `RegisterWithPassword` e
  `RegisterWithOtp` terminavam com o mesmo bloco de sessão, credenciais e resposta;
  o cadastro por convite seria a terceira cópia. A extração é mecânica e os testes do
  register passaram sem mudança de asserção.

- **O `login_type` da sessão do cadastro por convite** é `WITH_PASSWORD` quando o
  cadastro grava senha e `WITH_OTP` quando não — o enum `auth_method` só tem os dois,
  e o convite é a prova de e-mail que o OTP seria (fonte: leitura de
  `infra/entities/session.entity.go`).

- **`seededTables` de `cmd/database/reset.go` ganhou `invites`.** O `CASCADE` já
  apagava a tabela, mas a lista é explícita e ordenada por dependência.

- **`MockOtpService` e `MockRegisterService` ganharam a asserção de interface** que o
  CLAUDE.md exige e eles não tinham; sem ela, o método novo de cada interface só
  quebraria dentro do testify. Entrou também `mock.FakeEmailAdapter`, o primeiro
  adaptador de e-mail falso da suíte.

- **`TestActionWildcardReachesTheSubfeaturesOfItsFeature` mudou**: `as::organizations::*`
  passou a alcançar as duas rotas de convite. É a mudança de permissão que
  `steering/modules/profiles.md` avisa que toda chave nova é.

## Verificação

- `go build`, `go vet`, `go test ./...` e `go test -race` nos pacotes de convite e
  register, verdes. O lint acusa só `defer tx.Rollback()` sem checar o erro, o
  idioma que `steering/repository-pattern.md` manda usar.
- `make fresh` contra o banco da worktree; `invites` criada, `INVITE` no enum.
- Servidor de pé, 39 verificações de ponta a ponta por um cliente descartável em
  `cmd/sandbox/` (ignorado pelo git): os limites do `ttl_seconds`, as recusas da
  criação, descrever, segredo errado, aceite sem conta, cadastro por convite, reuso,
  listagem, substituição, revogação, aceite com conta só com o token. Os e-mails
  saíram pelo Resend para `delivered@resend.dev`.
