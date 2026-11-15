---
status: approved
slug: organization-invites
extracted_at: 2026-10-13
decided_at: 2026-11-15
gate: new-capability
modules: [invite (novo, core), otp, register, organization, participant, profile, user, user_pool]
sources:
  - session:2026-10-13 (pedido original; proposta de OTP de longa duração + `otps.token` avaliada em D-1)
  - session:2026-11-15 (respostas às D-1 a D-7, por texto e por AskUserQuestion)
  - code:app/modules/core/otp (GenerateConsumable, ValidateConsumable, Invalidate, entity.Otp)
  - code:app/modules/core/organization/services/organization.service.go (UpdateParticipant, CreateForUser, UpdateForCaller)
  - code:app/modules/core/participant/services/participant.service.go (Create nunca chamado, nota da linha 36)
  - code:app/modules/register/services/register.service.go (ProvisionUser, RegisterWithOtp)
  - code:app/modules/authorize/services/authorize.service.go (ChangeEmail, writeEmailChange)
  - code:app/middlewares/validators/register_custom.go
  - code:app/middlewares/guards/{otp,organization,permissions}.guard.go
  - code:shared/permissions/grants.go
  - code:cmd/database/seeds/profiles.go
  - code:cmd/database/migration/main.go (createEnum)
  - code:infra/config/env.go
  - code:infra/entities/session.entity.go (login_type é o enum auth_method)
  - code:shared/email, shared/models/views.dto.go
  - steering:workflow.md, module-structure.md, service-layer.md, repository-pattern.md, models-layer.md, controller-layer.md, guards-and-middlewares.md, error-handling.md, modules/profiles.md
  - frontend:docs/core-api.md §4 e §9 ("Não há convite nem criação de participante")
  - frontend:src/pages/console/ParticipantsPage.tsx:112
  - pendencias:3, 4, 5, 11, 16, 23, 25
---

# Requirements — Convite de usuário para uma organização

## Contexto

Hoje um usuário só entra numa organização de duas formas: criando uma
(`POST /core/organizations`, e vira dono) ou nascendo nela (`/auth/register`
provisiona a organização própria). Não existe caminho para uma organização
**trazer** alguém para dentro. O frontend registra a ausência em três lugares —
`core-api.md` §4 e §9, e a tela de participantes diz literalmente "There is no
invite or participant creation in /core".

O código já esperava por isto. `IParticipantService.Create` existe e nunca é
chamado (pendência 16), e o comentário acima dele avisa que "a future invite
endpoint" terá de conferir a pool. `steering/modules/profiles.md` fecha dizendo que
"an invite endpoint will be the next place to have to respect" o invariante de que
`participants.profile_id` aponta sempre para um profile visível à organização.

Duas propriedades do modelo definem o desenho:

- **Organização é escopada por pool** (`organizations.users_pool_id`), e **e-mail é
  único por pool** (`users_email_users_pool_unique`). "Convidar `fulano@x` para a
  organização O da pool P" significa o usuário `fulano@x` **da pool P** — que já
  existe, ou que ainda não.
- **Um usuário não tem permissão própria**: tem permissão numa organização, pela
  participação. Convidar é criar uma participação com um profile, e por isso herda
  as três recusas de `UpdateParticipant`, a terceira em especial: ninguém entrega
  um profile mais largo do que o que tem.

O segredo do link é um **OTP de ação `INVITE`** com a duração do convite (D-1). A
tabela `invites` guarda o que o OTP não sabe guardar — organização, profile, quem
convidou, aceito, revogado — e aponta para o OTP que serve de token.

## Atores

| Ator | Papel |
| --- | --- |
| Participante autorizado | Convida um e-mail para a organização em que está, escolhendo o profile e, se quiser, a validade; lista e revoga os convites da organização |
| Convidado (destinatário) | Recebe o e-mail e abre o link. Se já tem conta na pool, aceita só com o token; se não tem, cadastra-se com o token |
| Aplicação chamadora | Identificada pelo `AppGuard`. Um convite é lido e aceito **pela aplicação que o emitiu** (D-1) |
| Frontend | Página de aceite: lê o convite pelo token, decide entre "aceitar" e "criar conta", chama o aceite ou o cadastro |
| Backend | Emite o OTP, guarda só o hash do segredo, envia o e-mail com o link, consome o convite uma única vez |

## Requisitos funcionais

- **RF-1** — Um participante autorizado convida um e-mail para a organização em
  que está, escolhendo o profile que o convidado terá. O convite é um `POST`, e
  dispara um e-mail ao destinatário com um link de aceite.
- **RF-2** — O link contém um token que, sozinho, identifica o convite e prova que
  quem o apresenta recebeu o e-mail. Nada além do token é necessário para aceitar.
- **RF-3** — O destinatário aceita numa rota própria, distinta da de criação.
- **RF-4** — Se o destinatário já é usuário da pool da organização, aceita **só com
  o token, sem login** (D-2), e a participação é criada com o profile do convite.
  Nenhum cadastro acontece: nenhum usuário, organização própria ou sessão nova.
- **RF-5** — Se ainda não é, é criado na hora **naquela pool**, com o e-mail do
  convite, e a participação é criada em seguida. O cadastro por convite devolve o
  mesmo que um cadastro normal: sessão, tokens e o usuário com o profile resolvido.
- **RF-6** — Antes de aceitar, o frontend consegue, só com o token, saber para
  qual organização é o convite, para qual e-mail, com qual profile, até quando
  vale, e se aquele e-mail já tem conta na pool — é o que decide entre "aceitar" e
  "crie sua conta".
- **RF-7** — Um convite é consumido uma única vez e expira. Expirado, consumido,
  revogado ou substituído, deixa de aceitar.
- **RF-8** — Convidar de novo o mesmo e-mail para a mesma organização enquanto há
  convite pendente **substitui** o pendente: token novo, e-mail novo, e todo convite
  pendente anterior para aquele e-mail naquela organização é revogado (D-4).
- **RF-9** — A validade é escolhida na emissão (`ttl_seconds`, opcional). Sem ela,
  vale o padrão da organização (`organizations.invite_ttl_seconds`), e sem este, o
  padrão do serviço, 7 dias (D-7).
- **RF-10** — O padrão da organização é editado por `PUT /core/organizations/{id}`,
  pelo mesmo grant que edita a organização (D-7).
- **RF-11** — A organização lista os seus convites, com o estado de cada um, e
  filtra por estado (D-5).
- **RF-12** — A organização revoga um convite pendente. Revogado, o link deixa de
  valer na hora (D-5).

## Requisitos não funcionais

- **RNF-1** — O segredo do link nunca é guardado em claro fora de `development`: vai
  em `otps.code`, hasheado com argon2 como todo código de OTP. Em `development` fica
  em claro, como o código de seis dígitos já fica hoje (D-1).
- **RNF-2** — O segredo tem 32 bytes de `crypto/rand`, codificados em base64url. O
  token do link é `<otp_id>.<segredo>`: o id localiza a linha, porque o argon2 é
  salgado e não permite procurar pelo hash; só o segredo protege alguma coisa.
- **RNF-3** — Criar a participação e consumir o convite são uma unidade de
  trabalho: ou os dois valem, ou nenhum. Invalidar o OTP vem depois do commit, como
  em todo fluxo que consome um OTP hoje — o convite é a autoridade do consumo.
- **RNF-4** — A duração padrão e o endereço da página de aceite são configuração,
  não constantes.
- **RNF-5** — Toda rota nova tem entrada no catálogo de grants. Nenhuma fica
  alcançável só pelo administrador da plataforma por esquecimento.
- **RNF-6** — O módulo OTP ganha a ação `INVITE` e um método de emissão sem e-mail
  e com validade escolhida pelo chamador. `GenerateConsumable`, `VerifyConsumable` e
  o fluxo de seis dígitos não mudam. O enum `auth_action` do Postgres ganha o valor
  por migração idempotente (D-1).
- **RNF-7** — Nenhuma resposta devolve `entity.User` ou `Organization` cru: as
  views de `shared/models/views.dto.go`, como o teste de OpenAPI exige. O que vai ao
  portador do token não autenticado é um DTO mínimo, não a entidade.

## Regras e invariantes

- **RC-1** — **Quem convida não entrega mais do que tem.** O profile do convite
  tem que ser visível à organização (`FindByIdVisibleTo`) e caber em
  `Resolve(teto da organização, participação de quem convida)`, com `IsWithin` —
  a mesma comparação da terceira recusa de `UpdateParticipant`. Sem isso, convidar
  vira o caminho de escalonamento que `UpdateParticipant` fecha: dois
  participantes estreitos se convidam mutuamente com o profile de admin.
- **RC-2** — **`{id}` é a organização atual.** As três rotas de `/core` recusam com
  `403` qualquer outro id, pela razão de `UpdateForCaller` e `UpdateParticipant`: a
  permissão foi conferida na organização atual.
- **RC-3** — **O convite pertence à aplicação que o emitiu.** O OTP é
  `app + contato + ação`, então ler ou aceitar o convite por outra aplicação
  responde `404` — para ela o convite não existe. A emissão exige que a pool da
  aplicação seja a da organização, e o usuário existente é procurado nessa pool, e o
  novo provisionado nela. É a checagem que o comentário de
  `ParticipantService.Create` cobra do convite.
- **RC-4** — **O token basta para aceitar** (D-2). Não há sessão no aceite de quem
  já tem conta, e por isso nenhuma comparação com um e-mail logado. O token prova a
  posse da caixa postal do convite, que é o que o e-mail de convite autoriza.
- **RC-5** — **Convidar quem já participa é `409`.** Na criação, se o e-mail já
  pertence a um usuário da pool **e** esse usuário já participa da organização,
  recusa. Só "já participa" é revelado — informação que o chamador já tem na
  listagem de participantes. Existir na pool sem participar **não** é revelado.
- **RC-6** — **Aceitar já participando não é erro.** Um usuário que já participa
  da organização e apresenta um convite válido recebe `200`, o convite é
  consumido, e a participação existente **não** muda de profile — mudar profile é
  `UpdateParticipant`, com as recusas dele. Um convite já aceito responde `400`.
- **RC-7** — **Aceitar não troca a organização atual.** Como em `CreateForUser`,
  entrar não é mudar de escopo: a resposta carrega a organização e o cliente chama
  `PUT /core/organizations/switch`. Vale nos dois modos, inclusive para o usuário
  recém-criado, que nasce na própria organização como qualquer cadastro.
- **RC-8** — **Aceitar prova posse do e-mail.** O usuário novo nasce com
  `verify_email = true`, como em `RegisterWithOtp`; o usuário existente que ainda
  não verificou passa a verificado ao aceitar, pela razão do RC-3 da troca de
  e-mail: o token foi entregue naquela caixa postal e consumido.
- **RC-9** — **O segredo é conferido pelo OTP.** `ValidateConsumable` com a ação
  `INVITE` e a aplicação chamadora; qualquer recusa dele que não seja `500` vira
  `404` no convite, para que um segredo errado não se distinga de um convite
  inexistente.
- **RC-10** — **Pendente é um predicado, não uma coluna.**
  `accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now()`, em query
  dedicada do repositório — nunca em condição de struct, pelo motivo de
  `FindActive`: gorm descarta zero values e o predicado some em silêncio.
- **RC-11** — **O cadastro por convite segue as regras do cadastro por OTP.**
  Senha obrigatória se a aplicação aceita `WITH_PASSWORD`; sem senha se a
  aplicação é só OTP. O token do convite ocupa o lugar do OTP como prova do e-mail,
  e o e-mail vem do convite, nunca do corpo. Um e-mail que já tem conta na pool é
  `409`, apontando para o aceite: o cadastro por convite nunca vira aceite (D-2).
- **RC-12** — **Falha depois do provisionamento não perde o usuário.** O cadastro
  por convite são duas transações, porque `ProvisionUser` fecha a dele. Se a
  segunda falha, o usuário existe, o convite continua pendente, e o aceite só com
  token resolve na tentativa seguinte.
- **RC-13** — **`otp_id`, `expires_at`, `accepted_at` e `revoked_at` são colunas
  que o serviço possui.** Nenhuma entra num dao — o repositório é `repo.NoUpdate`;
  as duas transições são métodos dedicados, condicionais ao estado pendente, para
  que dois aceites simultâneos terminem com uma participação só e um aceite não
  ressuscite um convite revogado.
- **RC-14** — **A validade fica entre 1 hora e 30 dias**, no payload e no padrão da
  organização (D-7). O padrão do serviço vem da configuração e não é limitado: é
  decisão de quem opera o serviço.
- **RC-15** — **`expires_at` é escrito duas vezes, uma vez só.** No OTP, onde é
  aplicado, e no convite, onde é listado, com o mesmo valor, no momento da emissão,
  e nunca mais alterado. Uma cópia imutável não diverge; ler a validade do convite
  sem juntar com `otps` é o que mantém a listagem uma query só.

## Escopo

### Dentro

- Módulo core `invite` em `app/modules/core/invite/`: entidade `invites`,
  repositório, serviço e controller.
- `POST /core/organizations/{id}/invites` — cria e envia.
- `GET /core/organizations/{id}/invites` — lista, com `status` opcional (D-5).
- `DELETE /core/organizations/{id}/invites/{invite_id}` — revoga (D-5).
- `GET /auth/invite?token=` — descreve o convite ao portador do token (RF-6).
- `POST /auth/invite/accept` — aceite só com token de quem já tem conta (RF-4).
- `POST /auth/register?method=invite` — cadastro mais aceite (RF-5), no módulo
  `register` (D-2).
- Ação `INVITE` e `IOtpService.Issue` no módulo OTP; valor novo no enum por
  migração (D-1).
- Coluna `organizations.invite_ttl_seconds`, editável por
  `PUT /core/organizations/{id}` (D-7).
- Catálogo de grants, profiles semeados (D-3), configuração, e-mail, mocks, testes,
  OpenAPI, e o handoff para o frontend.

### Fora (exclusão explícita)

- **Camada de template de e-mail.** O corpo continua texto montado no ponto de
  envio, como o OTP e como a troca de e-mail decidiram.
- **Endereço de aceite por aplicação.** Fica por serviço (D-6, G-2).
- **Limite de taxa** de envio, por chamador ou por IP (pendência 4, G-1).
- **Convidar para uma pool**, criando usuário sem organização — é o item do
  roadmap "criar usuário via console", outro fluxo com outra prova.
- **Transferir o dono** ou convidar como dono. O dono continua sendo só
  `owner_user_id`, congelado como antes.
- **Voltar o padrão da organização a "nenhum"** (G-11).
- **A tela do frontend.** O backend entrega o handoff; a página de aceite e a lista
  de convites são do frontend.

## Decisões

- **D-1 — O token é um OTP de ação `INVITE`, referenciado pelo convite.**
  Pedido do autor em 2026-11-15: "orquestrar uma maneira de usar o OTP, mas também
  ter uma entidade de invite que se associa a OTP como token de convite", com a
  tabela própria como saída se ficasse complicado. Avaliado contra
  `app/modules/core/otp` e confirmado por `AskUserQuestion`. O que torna viável sem
  mexer no fluxo de seis dígitos:
  1. `GenerateConsumable` não serve — validade fixa de 5 minutos, corpo de e-mail
     próprio, `switch` por ação, rate limit. O módulo ganha `Issue(action, app,
     contact, ttl)`, que só cria a linha e devolve o segredo; quem chama envia o
     e-mail.
  2. `ValidateConsumable` e `Invalidate` servem como estão: conferem ação,
     aplicação, invalidado, expiração e o código com argon2.
  3. `otps.token` (uuid em claro) **não** é usado: o segredo vai em `otps.code`,
     que é o campo que o módulo protege.
  4. `createEnum` só cria o tipo quando ele não existe. A migração ganha um passo
     `ALTER TYPE … ADD VALUE IF NOT EXISTS` para cada ação, e `createEnum` deixa de
     reescrever o slice que recebe — hoje ele põe aspas dentro de
     `constants.AuthActions`.

  Consequências aceitas: escopo por aplicação em vez de pool (RC-3); segredo em
  claro em `development` (RNF-1); um id não secreto no link (RNF-2).

- **D-2 — Cadastro por convite como `POST /auth/register?method=invite`; aceite de
  quem já tem conta só com o token.** Pedido do autor: "no caso de um convite para
  um email que já existe na POOL não deve engatilhar um novo evento de register,
  apenas a adição de um participante". Por `AskUserQuestion`, o aceite **não exige
  login**. `register` continua sendo o único caminho que cria usuário (pendência
  11); `ProvisionUser` vive em `IRegisterService`, `fx.Private`, e `register` é
  módulo api, que depende de `invite`, que é core.

- **D-3 — Só `MANAGER_PROFILE` recebe os grants de `/core` do convite.** Resposta
  por `AskUserQuestion`. `LOGIN_PROFILE` não convida, não lista e não revoga: uma
  organização nascida numa pool comum não traz ninguém para dentro. Os dois grants
  de `/auth` (`as::invite::READ`, `as::invite::accept::CREATE`) entram nos dois,
  junto do conjunto de autoatendimento, porque aceitar é um ato da pool, como se
  cadastrar.

- **D-4 — Convite repetido substitui o pendente e revoga os anteriores.** Resposta
  do autor: "sim, e invalida os convites anteriores".

- **D-5 — Listar e revogar entram nesta entrega.** Resposta do autor.

- **D-6 — Endereço da página de aceite é um template no config**
  (`INVITE_ACCEPT_URL`, com `{token}` e opcionalmente `{organization_id}`), por
  serviço. Sem a variável, a rota de criação responde `501`, como as de avatar sem
  storage. Resposta do autor: "sim".

- **D-7 — Validade escolhida na emissão, com padrão por organização.** Pedido do
  autor: "configurável na hora de emitir o convite, com um valor default por
  organização". Limites de 1 hora a 30 dias por `AskUserQuestion`. Padrão do
  serviço `INVITE_TTL_SECONDS`, 7 dias, com o padrão no código como
  `avatar.grant_ttl_seconds`.

## Lacunas detectadas

- **G-1** — Sem limite de taxa de envio: um participante autorizado dispara
  convites em série e cada `POST` manda um e-mail. É o mesmo buraco da pendência
  4 e do G-1 dos avatares, atrás de autenticação e de permissão.
- **G-2** — Endereço de aceite por serviço, não por aplicação. Num serviço com
  várias aplicações e vários frontends, o link vai sempre para o mesmo.
- **G-3** — O remetente é literal em `OtpService` (`No Reply <contact@vanin.dev>`),
  e o convite repete o literal. Levar para a configuração é mexer nos dois; vai
  para `pendencias.md`.
- **G-4** — `GET /auth/invite` diz ao portador do token se o e-mail tem conta na
  pool. O portador recebeu o e-mail naquele endereço, então não é oráculo de
  enumeração. Registrado para constar.
- **G-5** — O token viaja na query string da página do frontend: fica em
  histórico e em `Referer`. O frontend deve removê-lo da URL depois de ler
  (`history.replaceState`). Entra no handoff.
- **G-6** — Corrida entre dois `POST` para o mesmo e-mail: os dois revogam o
  anterior e inserem, e ficam dois pendentes. Um índice único parcial fecharia,
  mas a violação subiria como `500` (pendência 25). Aceito: o último e-mail
  enviado é o que o destinatário usa, e revogar continua possível.
- **G-7** — `ProvisionUser` não participa de transação externa (RC-12). Refatorar
  para aceitar `repo.Option` fecharia a janela entre as duas transações; não vale
  o custo agora.
- **G-8** — `utils.GenerateRandomDigits` e `GenerateRandomCharacters` usam
  `math/rand`, não `crypto/rand`. Fora do escopo — o segredo do convite não sai de
  lá —, mas o código do OTP sai. Vai para `pendencias.md`.
- **G-9** — **Quem tem o link põe aquela conta na organização** (D-2, RC-4). Um
  e-mail encaminhado ou um link vazado acrescenta a conta à organização sem senha.
  O efeito é limitado — a conta ganha uma participação com o profile que quem
  convidou podia dar, nada se lê da conta e nenhuma sessão é aberta —, mas é o custo
  aceito de não exigir login.
- **G-10** — Quem conhece só o `otp_id`, sem o segredo, distingue um convite aceito,
  revogado ou expirado de um pendente, porque os estados são conferidos antes do
  segredo para darem mensagens próprias. O `otp_id` só viaja dentro do link.
- **G-11** — `invite_ttl_seconds` não volta a `NULL` por payload: no dao, `nil` é
  "não mexa". Um padrão escrito só é trocado por outro.
- **G-12** — `ValidateConsumable` não conta tentativas. Com 256 bits de segredo,
  força bruta não é o risco; registrado porque o fluxo de seis dígitos conta, e o
  convite reaproveita o método que não conta.

## Evidências da sessão

O pedido fixa o par de rotas e a direção do segredo, origem de RF-1 a RF-3:

> "Essa funcionalidade deve ser uma rota de POST que dispara um email para um
> destinatário que tem como propósito fazer ele clicar em um link contendo as
> informações necessárias para aceitar o convite de maneira segura apenas do lado
> dele (do destinatário). Para aceite devemos ter que usar uma outra rota de aceite
> de convite."

O escopo por pool e os dois modos de aceite, origem de RF-4, RF-5 e RC-3:

> "organizações devem ser escopadas por user pool, portanto ao chamar um
> destinatário a uma organização o mesmo deve ser um usuário já existente a aquela
> pool ou deve ser criado na hora para aquela pool antes de aceitar o convite."

A proposta de mecanismo, avaliada em D-1:

> "organization/uuid/invite?token=base64(invite_uuid|otp_uuid|otp_token). Estava
> pensando em usar um otp de longa duração (mesma duração do convite) como chave
> validadora do aceite e ao invés de passar o código otp usar um token associado
> ao otp. Isso é prático e viável? se você tiver uma ideia melhor eu posso avaliar."

As respostas de 2026-11-15, origem das decisões:

> "D-1: Tentar orquestrar uma maneira de usar o OTP, mas também ter uma entidade de
> invite de se associa a OTP como token de convite. Se não for possível ou muito
> complicado, pode optar por usar apenas a entidade de invite."
>
> "D-2: sim, pode ficar. Mas é importante que no caso de um convite para um email
> que já existe na POOL não deve engatilhar um novo evento de register apenas a
> adição de um participante para a mesma organização."
>
> "D-4: sim, e invalida os convites anteriores" · "D-5: pode adicionar nessa
> entregar tbm" · "D-6: sim"
>
> "D-7: Sim, mas seria bom ser configurável na hora de emitir o convite, com um
> valor default por organização."

E por `AskUserQuestion`: D-1 "Convite + OTP"; D-3 "Não, só Manager e acima"; D-7
"1 hora a 30 dias"; D-2 "Só o token basta".

## Fontes consultadas

Os nove documentos de steering; o módulo OTP inteiro, que é o que a proposta
original reutiliza; `UpdateParticipant`, `CreateForUser` e `UpdateForCaller`,
que são as regras de escopo e de teto que o convite herda; `ProvisionUser` e
`RegisterWithOtp`, que o cadastro por convite copia; `ChangeEmail`, que é o
precedente de "provar posse do e-mail marca verificado"; o catálogo de grants e os
profiles semeados; `createEnum` da migração; o bloco `avatar` e `storage` do
config, que é o modelo da configuração nova; as views de resposta; a entidade de
sessão, cujo `login_type` é o enum `auth_method`; e, do frontend, `core-api.md` e
a tela de participantes, que registram a ausência do recurso.
