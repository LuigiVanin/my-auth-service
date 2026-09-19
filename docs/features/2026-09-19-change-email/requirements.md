---
status: implemented
slug: change-email
extracted_at: 2026-09-19
gate: new-capability
modules: [authorize, session, otp, user]
sources:
  - session:2026-09-19 (pedido original, quatro AskUserQuestion respondidos)
  - code:app/modules/authorize
  - code:app/modules/core/{session,otp,user}
  - code:shared/permissions/grants.go
  - code:cmd/database/seeds/profiles.go
  - steering:service-layer.md
  - steering:repository-pattern.md
  - steering:guards-and-middlewares.md
  - steering:modules/profiles.md
  - pendencias:14 (fechada por esta feature)
  - pendencias:22
---

# Requirements — Troca de e-mail verificada e encerramento de sessão

## Contexto

O próprio usuário não troca o seu e-mail. A única forma é `PUT /core/users/{id}`,
que é rota de administração de pool: o comentário de `dto.UpdateUserSelf` exclui
`email` de propósito, porque "email e as duas flags de verificação decidem como o
usuário se autentica, então movê-los é ato de administração e não edição de
perfil". A constante `constants.ActionChangeEmail` existe desde sempre e não é
referenciada em lugar nenhum — é a [pendência 14](../pendencias.md).

Junto vem o encerramento de sessão. `AuthorizeController.Revoke` já existe e
responde `NOT_IMPLEMENTED`, nenhuma rota aponta para ele, e `ISessionService` não
tem método de encerrar sessão nenhuma. Ele entra nesta feature porque a troca de
e-mail **depende** dele: quando a identidade muda, os tokens emitidos sob a
identidade antiga não podem continuar valendo.

E aqui está a armadilha que o pedido antecipou. Até hoje `sessions.invalidated` era
escrito por um único lugar — o `InvalidateAllExcept` de `SessionService.CreateNew` —
e por isso só significa "nasceu uma sessão mais nova". O `/auth/refresh` passou a
**aceitar** sessão invalidada justamente por causa disso: recusar derrubava quem não
fez nada errado, porque outro cliente logou. Consequência direta: marcar
`invalidated` não mata mais o refresh token. É a [pendência 22](../pendencias.md).

## Atores

| Ator | Papel |
| --- | --- |
| Usuário autenticado | Pede a troca do próprio e-mail, confirma com o código, e é deslogado |
| Aplicação chamadora | Identificada pelo `AppGuard`; define a pool onde a unicidade do e-mail vale |
| Frontend | Descarta os tokens ao receber o `200` da confirmação e leva para o login |

## Requisitos funcionais

- **RF-1** — Um usuário autenticado pede a troca informando o **e-mail novo**, e
  recebe um código OTP **naquele endereço** — não no atual.
- **RF-2** — Ao informar o código, o e-mail novo passa a ser o da conta.
- **RF-3** — Concluída a troca, o usuário é deslogado da plataforma, de modo que
  possa entrar imediatamente com o e-mail novo.
- **RF-4** — Se o e-mail de destino já pertence a alguém na pool, a requisição é
  recusada com erro. A recusa acontece **duas vezes**: ao gerar o código e ao
  consumi-lo.
- **RF-5** — `POST /auth/revoke` encerra a sessão do token apresentado e
  impossibilita o uso do refresh token daquela sessão.

## Requisitos não funcionais

- **RNF-1** — A troca de e-mail e o encerramento das sessões valem juntos ou não
  valem: são uma unidade de trabalho.
- **RNF-2** — O caminho do `/auth/refresh` não muda. O comportamento que
  `TestRefreshAcceptsASupersededSession` fixa continua valendo: uma sessão superada
  por um login novo ainda renova.
- **RNF-3** — Nenhuma migração. `sessions.refresh_expires_at` já é coluna, e o enum
  Postgres `auth_action` já contém `CHANGE_EMAIL`.
- **RNF-4** — Toda rota nova é alcançável por grant, e nenhuma fica acessível só ao
  administrador da plataforma por esquecimento de catálogo.

## Regras e invariantes

- **RC-1** — O e-mail é único por pool (`users_email_users_pool_unique`), e a
  checagem no serviço existe para que a colisão vire `409 USER_ALREADY_EXISTS` em
  vez de violação de índice subindo como 500.
- **RC-2** — O OTP consumido tem que ser da ação `CHANGE_EMAIL`, da aplicação
  chamadora, e o e-mail guardado no seu metadata tem que ser o do corpo.
- **RC-3** — Confirmar a troca marca `verify_email = true` e grava
  `verified_email_date`. O código foi entregue no endereço novo e consumido, então
  a posse está provada; deixar `false` trancaria o usuário em toda rota `/core`
  pelo `EmailVerificationGuard` no instante seguinte à troca.
- **RC-4** — A rota de confirmação **não** carrega o `EmailVerificationGuard`.
  Exigir e-mail verificado para trocar de e-mail trancaria exatamente quem se
  cadastrou com o endereço errado, que é o caso principal da feature. Nenhuma rota
  de `/auth` carrega esse guard.
- **RC-5** — Encerrar uma sessão significa matar os **dois** tokens: o de acesso,
  que morre por `invalidated` porque `FindActive` filtra por ele, e o de refresh,
  que morre porque a janela `refresh_expires_at` vai para o passado.
- **RC-6** — O logout é idempotente: revogar duas vezes a mesma sessão não é erro.

## Escopo

### Dentro

A ação `CHANGE_EMAIL` no validador de OTP e no `OtpService`; a rota
`PUT /auth/change_email`; a primitiva de revogação no módulo `session`; a rota
`POST /auth/revoke`; as entradas de catálogo e os profiles semeados; e a correção
dos três defeitos de `AuthorizeService.VerifyEmail`, que é o fluxo que esta feature
copia.

### Fora (exclusão explícita)

- **A coluna `revoked_at`.** A [pendência 22](../pendencias.md) continua aberta: o
  epoch em `refresh_expires_at` é paliativo, não a separação definitiva entre
  "superada" e "encerrada". Decisão explícita do usuário nesta sessão.
- **Prender o OTP ao usuário que o pediu.** `entity.Otp.UserId` existe e é sempre
  `nil`; fechar custa mudar a assinatura de `IOtpService.GenerateConsumable`. Ver
  G-1.
- **Trocar e-mail sem estar logado.** Quem perdeu acesso à conta usa o fluxo de
  senha; recuperar a conta por um endereço que não se controla mais é outro
  problema.
- **Revogar a sessão de outro aparelho.** `POST /auth/revoke` não recebe corpo, e
  não existe rota que liste sessões, então não haveria de onde tirar o id.
- **Camada de template de e-mail.** O corpo continua o do único ponto de envio do
  serviço.

## Lacunas detectadas

- **G-1** — O OTP de `CHANGE_EMAIL` não fica preso a quem o pediu. Quem obtiver o
  par `otp_id` + código consegue apontar aquele endereço para a própria conta. O
  dano é o atacante ficar com um e-mail que não controla, e exige código roubado —
  baixo, mas real.
- **G-2** — O `409` da geração conta a um chamador **autenticado** se um endereço
  existe na pool. É o mesmo oráculo de enumeração da
  [pendência 1](../pendencias.md), atrás de autenticação.
- **G-3** — Se a corrida entre gerar e consumir escapar da re-checagem, a violação
  do índice único sobe como 500: o serviço não pode importar gorm para mapear o
  erro do driver.
- **G-4** — `/auth/verify_email` foi registrada no commit anterior **sem** entrada
  no catálogo de grants. Corrigido aqui de passagem.
- **G-5** — `LoginPermissions` não carregava `as::otp::CREATE`, então o profile
  padrão de toda pool declarava fluxos que não podia iniciar. Nada em `/auth` e
  `/otp` é enforçado hoje, então era latente.

## Evidências da sessão

O pedido descreve o fluxo inteiro, e é a origem dos RF-1 a RF-3:

> "um email deve ser enviado para o email novo com um código otp, ao inserir esse
> código na interface o novo email é configurado na conta e o usuário é deslogado
> da plataforma, dessa maneira ele pode já tentar o novo email para a conta dele"

O RF-4 veio literal:

> "É óbvio que deve ser testado se o email final já não existe na plataforma, nesse
> caso um erro deve ser lançado"

E o RF-5 veio com o diagnóstico do problema já pronto, que é o que gerou o RC-5:

> "a rota de auth/revoke que invalida uma sessão de usuário e impossibilita o uso
> de um refresh token (em tese marcar a sessão como invalidated era o suficiente,
> mas algum bugs ocorreram e creio que isso não seja mais verdade, se possível
> mudar o refresh token exp da sessão individual para 0 pode realizar isso)"

Quatro decisões foram levadas como pergunta e respondidas: o mecanismo de morte do
refresh token (expirar a janela, não coluna nova), o escopo do logout na troca
(todas as sessões do usuário naquele app), a forma do `/auth/revoke` (sem corpo,
sessão do próprio token) e o destino do bug invertido em `VerifyEmail` (corrigir
junto).

## Fontes consultadas

Steering de service, repository, models, controller, guards e profiles; o catálogo
de grants e os profiles semeados; a mudança não commitada de
`authorize.service.go` e o teste não rastreado `refresh.service_test.go` que a
fixa; os itens 1, 13, 14, 20 e 22 de `pendencias.md`.
