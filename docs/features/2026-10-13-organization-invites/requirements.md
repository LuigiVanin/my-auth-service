---
status: draft
slug: organization-invites
extracted_at: 2026-10-13
gate: new-capability
modules: [invite (novo, core), register, organization, participant, profile, user, user_pool]
sources:
  - session:2026-10-13 (pedido original; proposta de OTP de longa duração + `otps.token` avaliada em D-1)
  - code:app/modules/core/otp (GenerateConsumable, ValidateConsumable, entity.Otp.Token)
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
  - code:shared/email, shared/models/views.dto.go
  - steering:workflow.md, module-structure.md, service-layer.md, repository-pattern.md, models-layer.md, controller-layer.md, guards-and-middlewares.md, error-handling.md, modules/profiles.md
  - frontend:docs/core-api.md §4 e §9 ("Não há convite nem criação de participante")
  - frontend:src/pages/console/ParticipantsPage.tsx:112
  - pendencias:3, 4, 5, 11, 16, 23, 25
---

# Requirements — Convite de usuário para uma organização

> **Rascunho.** As decisões marcadas `D-*` aguardam resposta do autor. O resto foi
> extraído do pedido e do código e já pode ser lido como requisito.

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

A proposta original — um OTP com a duração do convite, e `otps.token` como
credencial no lugar do código — foi avaliada contra o módulo OTP e substituída por
uma tabela própria. As razões estão em D-1; o que a proposta acertou está lá
também, e fica.

## Atores

| Ator | Papel |
| --- | --- |
| Participante autorizado | Convida um e-mail para a organização em que está, escolhendo o profile que o convidado terá |
| Convidado (destinatário) | Recebe o e-mail, abre o link e aceita — logado, se já tem conta na pool; cadastrando-se, se não tem |
| Aplicação chamadora | Identificada pelo `AppGuard`; define a pool. Um convite só é lido e aceito por uma aplicação da pool da organização |
| Frontend | Página de aceite: lê o convite pelo token, decide entre login e cadastro, chama o aceite, chama o switch |
| Backend | Gera o segredo, guarda só o hash, envia o e-mail com o link, consome o token uma única vez |

## Requisitos funcionais

- **RF-1** — Um participante autorizado convida um e-mail para a organização em
  que está, escolhendo o profile que o convidado terá. O convite é um `POST`, e
  dispara um e-mail ao destinatário com um link de aceite.
- **RF-2** — O link contém um token que, sozinho, identifica o convite e prova que
  quem o apresenta recebeu o e-mail. Nada além do token é necessário para aceitar.
- **RF-3** — O destinatário aceita numa rota própria, distinta da de criação.
- **RF-4** — Se o destinatário já é usuário da pool da organização, aceita
  **autenticado**, e a participação é criada com o profile do convite.
- **RF-5** — Se ainda não é, é criado na hora **naquela pool**, com o e-mail do
  convite, e a participação é criada em seguida. O cadastro por convite devolve o
  mesmo que um cadastro normal: sessão, tokens e o usuário com o profile resolvido.
- **RF-6** — Antes de aceitar, o frontend consegue, só com o token, saber para
  qual organização é o convite, para qual e-mail, com qual profile, até quando
  vale, e se aquele e-mail já tem conta na pool — é o que decide entre "faça
  login" e "crie sua conta".
- **RF-7** — Um convite é consumido uma única vez e expira. Expirado, consumido ou
  substituído, deixa de aceitar.
- **RF-8** — Convidar de novo o mesmo e-mail para a mesma organização enquanto há
  um convite pendente **substitui** o pendente: token novo, e-mail novo, o anterior
  deixa de valer. É o "reenviar convite" sem precisar de rota de revogação (D-4).

## Requisitos não funcionais

- **RNF-1** — O segredo do link nunca é guardado em claro. O banco guarda um
  hash; vazar a tabela não entrega nenhum link válido.
- **RNF-2** — O token tem entropia suficiente para ser indeduzível — 32 bytes de
  `crypto/rand` — e é o **único** segredo. Nenhum id de banco precisa viajar na
  URL.
- **RNF-3** — Criar a participação e consumir o convite são uma unidade de
  trabalho: ou os dois valem, ou nenhum.
- **RNF-4** — A duração do convite e o endereço da página de aceite são
  configuração, não constantes.
- **RNF-5** — Toda rota nova tem entrada no catálogo de grants. Nenhuma fica
  alcançável só pelo administrador da plataforma por esquecimento.
- **RNF-6** — O módulo OTP não muda. O convite não é uma `auth_action`, não passa
  por `GenerateConsumable` e não toca o enum do Postgres.
- **RNF-7** — Nenhuma resposta devolve `entity.User`, `Organization` ou
  `Participant` cru: as views de `shared/models/views.dto.go`, como o teste de
  OpenAPI exige. O que vai ao portador do token não autenticado é um DTO mínimo,
  não a entidade.

## Regras e invariantes

- **RC-1** — **Quem convida não entrega mais do que tem.** O profile do convite
  tem que ser visível à organização (`FindByIdVisibleTo`) e caber em
  `Resolve(teto da organização, participação de quem convida)`, com `IsWithin` —
  a mesma comparação da terceira recusa de `UpdateParticipant`. Sem isso, convidar
  vira o caminho de escalonamento que `UpdateParticipant` fecha: dois
  participantes estreitos se convidam mutuamente com o profile de admin.
- **RC-2** — **`{id}` é a organização atual.** `POST /core/organizations/{id}/invites`
  recusa com `403` qualquer outro id, pela razão de `UpdateForCaller` e
  `UpdateParticipant`: a permissão foi conferida na organização atual.
- **RC-3** — **O convite pertence à pool da organização.** Ler ou aceitar por uma
  aplicação de outra pool responde `404` — para aquela aplicação o convite não
  existe. Usuário e organização compartilham a pool por construção: o usuário
  existente é procurado em `app.UsersPool.ID`, e o novo é provisionado nela. É a
  checagem que o comentário de `ParticipantService.Create` cobra do convite.
- **RC-4** — **O e-mail de quem aceita é o do convite.** No aceite autenticado,
  `user.email` tem que ser igual, sem diferenciar maiúsculas, a `invites.email`.
  Outro usuário logado apresentando um token alheio recebe `403`.
- **RC-5** — **Convidar quem já participa é `409`.** Na criação, se o e-mail já
  pertence a um usuário da pool **e** esse usuário já participa da organização,
  recusa. Só "já participa" é revelado — informação que o chamador já tem na
  listagem de participantes. Existir na pool sem participar **não** é revelado.
- **RC-6** — **Aceitar já participando não é erro.** Um usuário que já participa
  da organização e apresenta um convite válido recebe `200`, o convite é
  consumido, e a participação existente **não** muda de profile — mudar profile é
  `UpdateParticipant`, com as recusas dele. Um token já consumido responde `400`
  ("já usado"), como o OTP.
- **RC-7** — **Aceitar não troca a organização atual.** Como em `CreateForUser`,
  entrar não é mudar de escopo: a resposta carrega a organização e o cliente chama
  `PUT /core/organizations/switch`. Vale nos dois modos, inclusive para o usuário
  recém-criado, que nasce na própria organização como qualquer cadastro.
- **RC-8** — **Aceitar prova posse do e-mail.** O usuário novo nasce com
  `verify_email = true`, como em `RegisterWithOtp`; o usuário existente que ainda
  não verificou passa a verificado ao aceitar, pela razão do RC-3 da troca de
  e-mail: o token foi entregue naquela caixa postal e consumido.
- **RC-9** — **A busca é pelo hash.** `token_hash = sha256(segredo)`, único e
  indexado. Um segredo de 256 bits aleatórios não precisa do custo nem do sal do
  argon2 — o hash lento existe para códigos de seis dígitos — e é o que permite
  procurar a linha pelo próprio hash, sem id na URL.
- **RC-10** — **Pendente é um predicado, não uma coluna.**
  `accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now()`, em query
  dedicada do repositório — nunca em condição de struct, pelo motivo de
  `FindActive`: gorm descarta zero values e o predicado some em silêncio.
- **RC-11** — **O cadastro por convite segue as regras do cadastro por OTP.**
  Senha obrigatória se a aplicação aceita `WITH_PASSWORD`; sem senha se a
  aplicação é só OTP. O token do convite ocupa o lugar do OTP como prova do e-mail,
  e o e-mail vem do convite, nunca do corpo.
- **RC-12** — **Falha depois do provisionamento não perde o usuário.** O cadastro
  por convite são duas transações, porque `ProvisionUser` fecha a dele. Se a
  segunda falha, o usuário existe, o convite continua pendente, e o aceite
  autenticado resolve na tentativa seguinte.
- **RC-13** — **`token_hash`, `accepted_at` e `revoked_at` são colunas que o
  serviço possui.** `token_hash` é `json:"-"`; nenhuma das três entra num dao de
  atualização; as duas transições são métodos dedicados de repositório, e a de
  aceite é condicional ao estado pendente, para que dois aceites simultâneos
  terminem com uma participação só.

## Escopo

### Dentro

- Módulo core `invite` em `app/modules/core/invite/`: entidade `invites`,
  repositório, serviço e controller.
- `POST /core/organizations/{id}/invites` — cria e envia. Cadeia completa de
  guards, grant novo.
- `GET /auth/invite?token=` — descreve o convite ao portador do token (RF-6).
- `POST /auth/invite/accept` — aceite autenticado (RF-4).
- `POST /auth/register?method=invite` — cadastro mais aceite (RF-5), no módulo
  `register` (D-2).
- Catálogo de grants, profiles semeados (D-3), migração por `AutoMigrate`,
  configuração, e-mail, mocks, testes, OpenAPI, e o handoff para o frontend.

### Fora (exclusão explícita)

- **Listar e revogar convites.** A tela de participantes vai pedir os dois; o
  modelo já os comporta (`revoked_at`). Ficam para uma entrega própria (D-5).
- **Camada de template de e-mail.** O corpo continua texto montado no ponto de
  envio, como o OTP e como a troca de e-mail decidiram.
- **Endereço de aceite por aplicação.** Fica por serviço (D-6, G-2).
- **Limite de taxa** de envio, por chamador ou por IP (pendência 4, G-1).
- **Convidar para uma pool**, criando usuário sem organização — é o item do
  roadmap "criar usuário via console", outro fluxo com outra prova.
- **Reutilizar o módulo OTP** como mecanismo do token (D-1).
- **Transferir o dono** ou convidar como dono. O dono continua sendo só
  `owner_user_id`, congelado como antes.

## Decisões em aberto

- **D-1 — Token próprio em vez de OTP de longa duração + `otps.token`.**
  Recomendação: **tabela própria**, pelos seis pontos abaixo, todos lidos em
  `app/modules/core/otp`:
  1. `OtpExpirationDuration` é constante de 5 minutos dentro de
     `GenerateConsumable`. Expiração por ação exige outra assinatura ou um `switch`.
  2. O corpo do e-mail — "Your OTP is Here: …" — está dentro de
     `GenerateConsumable`. O convite precisa de link, nome da organização e de quem
     convidou. Outro `switch`.
  3. `otps.token` é um uuid guardado **em claro**, enquanto `code` é hasheado fora
     de `development`. Usar o token como credencial inverte a proteção que o módulo
     faz questão de ter.
  4. Um OTP é `contact + action + app`. Um convite é `organização + e-mail +
     profile + quem convidou`. Isso iria para o `metadata` livre (pendência 3), sem
     índice, sem listagem possível, sem estado de aceite.
  5. `auth_action` é enum do Postgres, e `createEnum` da migração só cria quando o
     tipo não existe. Uma ação `INVITE` exigiria `ALTER TYPE … ADD VALUE` à mão.
  6. `base64(invite_uuid|otp_uuid|otp_token)` carrega dois ids que não são
     segredo; só o terceiro protege alguma coisa. Um segredo só, procurado pelo
     hash, faz o mesmo trabalho com menos partes.

  O que a proposta acertou e **fica**: link no lugar de código digitado, longa
  duração, um segredo por convite, consumo único, a aplicação chamadora como
  escopo.

- **D-2 — Cadastro por convite como `POST /auth/register?method=invite`**, e não
  como rota do módulo `invite`. Recomendação: **register**. `ProvisionUser` é o
  único caminho que cria usuário (pendência 11), vive em `IRegisterService`, que é
  `fx.Private`, e `register` é módulo api — ele depende de `invite`, que é core,
  nunca o contrário. `RegisterValidator` ganha o terceiro `case`. A alternativa,
  um `ProvisionUser` exposto a um módulo core, inverte a direção das dependências.

- **D-3 — Quais profiles semeados recebem `as::organizations::invites::CREATE`.**
  `MANAGER_PROFILE` sim — é teto de `LOGIN_PROFILE` e `IsSubsetOf` recusa um filho
  que nomeie o que o pai não tem. **`LOGIN_PROFILE` precisa de decisão**: sem ele,
  uma organização nascida numa pool comum (teto `LOGIN_PROFILE`) não convida
  ninguém, e o recurso só existe para organizações com teto Manager ou Admin. Com
  ele, todo dono de organização convida, mas continua sem
  `as::organizations::participants::READ` — assimetria que já existe. Alargar
  `LOGIN_PROFILE` é mudança de permissão que a pendência 5 pede para decidir à
  parte, então fica como pergunta. `as::invite::READ` e `as::invite::accept::CREATE`
  entram nos dois, junto do conjunto de autoatendimento, pela mesma razão que
  `as::otp::CREATE` viaja com `change_email`.

- **D-4 — Convite repetido substitui o pendente** (RF-8), em vez de `409`. Sem
  rota de revogação nesta entrega, `409` deixaria um link perdido irrecuperável
  pela duração inteira do convite. Recomendação: **substituir**.

- **D-5 — Listar e revogar ficam fora.** Recomendação: **fora**, para a primeira
  entrega ser pequena; `revoked_at` já nasce na tabela para que entrem sem
  migração.

- **D-6 — Endereço da página de aceite é um template no config**
  (`INVITE_ACCEPT_URL`, com `{token}` e opcionalmente `{organization_id}`), por
  serviço. Comporta o formato proposto —
  `…/organizations/{organization_id}/invite?token={token}` — e um
  `…/invite?token={token}` mais curto, que basta, já que o token resolve tudo. Por
  aplicação fica para depois (G-2). Sem a variável, a rota de criação responde
  `501`, como as de avatar sem storage.

- **D-7 — Duração padrão de 7 dias**, `INVITE_TTL_SECONDS`, com o padrão no código
  como `avatar.grant_ttl_seconds`.

## Lacunas detectadas

- **G-1** — Sem limite de taxa de envio: um participante autorizado dispara
  convites em série e cada `POST` manda um e-mail. É o mesmo buraco da pendência
  4 e do G-1 dos avatares, atrás de autenticação e de permissão.
- **G-2** — Endereço de aceite por serviço, não por aplicação. Num serviço com
  várias aplicações e vários frontends, o link vai sempre para o mesmo.
- **G-3** — O remetente é literal em `OtpService` (`No Reply <contact@vanin.dev>`).
  O convite ou repete o literal ou o leva para a configuração. Pequeno o bastante
  para resolver junto.
- **G-4** — `GET /auth/invite` diz ao portador do token se o e-mail tem conta na
  pool. O portador recebeu o e-mail naquele endereço, então não é oráculo de
  enumeração. Registrado para constar.
- **G-5** — O token viaja na query string da página do frontend: fica em
  histórico e em `Referer`. O frontend deve removê-lo da URL depois de ler
  (`history.replaceState`). Entra no handoff.
- **G-6** — Corrida entre dois `POST` para o mesmo e-mail: os dois revogam o
  anterior e inserem, e ficam dois pendentes. Um índice único parcial fecharia,
  mas a violação subiria como `500` (pendência 25). Aceito: o último e-mail
  enviado é o que o destinatário usa.
- **G-7** — `ProvisionUser` não participa de transação externa (RC-12). Refatorar
  para aceitar `repo.Option` fecharia a janela entre as duas transações; não vale
  o custo agora.
- **G-8** — `utils.GenerateRandomDigits` e `GenerateRandomCharacters` usam
  `math/rand`, não `crypto/rand`. Fora do escopo, mas o código do OTP sai de lá.
  Vai para `pendencias.md`.

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

E a proposta de mecanismo, avaliada em D-1:

> "organization/uuid/invite?token=base64(invite_uuid|otp_uuid|otp_token). Estava
> pensando em usar um otp de longa duração (mesma duração do convite) como chave
> validadora do aceite e ao invés de passar o código otp usar um token associado
> ao otp. Isso é prático e viável? se você tiver uma ideia melhor eu posso avaliar."

## Fontes consultadas

Os nove documentos de steering; o módulo OTP inteiro, que é o que a proposta
original reutilizaria; `UpdateParticipant`, `CreateForUser` e `UpdateForCaller`,
que são as regras de escopo e de teto que o convite herda; `ProvisionUser` e
`RegisterWithOtp`, que o cadastro por convite copia; `ChangeEmail`, que é o
precedente de "provar posse do e-mail marca verificado"; o catálogo de grants e os
profiles semeados; `createEnum` da migração; o bloco `avatar` e `storage` do
config, que é o modelo da configuração nova; as views de resposta; e, do frontend,
`core-api.md` e a tela de participantes, que registram a ausência do recurso.
