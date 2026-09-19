---
status: implemented
context: change-email
created: 2026-09-19
updated: 2026-09-19
implemented: 2026-09-19
drivers: >
  O usuário não troca o próprio e-mail — só a rota de administração de pool troca,
  e a constante da ação existia sem fluxo. Destrava o autoatendimento de conta, e
  de quebra entrega a primitiva de encerrar sessão, que não existia e sem a qual
  trocar de e-mail deixaria tokens vivos sob a identidade antiga.
---

# Implementation Plan: change-email

## Objetivo

Entregar `PUT /auth/change_email` — usuário autenticado pede, recebe o código no
endereço **novo**, confirma, e sai com o e-mail trocado, verificado e todas as
sessões daquele app encerradas — e `POST /auth/revoke`, o logout que não existia e
que é a primitiva da qual a troca depende. Sem migração e sem tocar no caminho do
`/auth/refresh`.

## Decisões tomadas

- **Revogar expira a janela de refresh em vez de ganhar coluna própria.** Uma
  query dedicada grava `invalidated = true` **e** `refresh_expires_at` no epoch. O
  `Refresh` não muda: a checagem `RefreshExpiresAt.Compare(time.Now()) < 0` que já
  existe recusa, e o acesso já morria por `FindActive`. A alternativa — `revoked_at`,
  que é o que a [pendência 22](../pendencias.md) prescreve — foi levada como
  pergunta e recusada em favor da mais barata, com a pendência mantida aberta
  (fonte: AskUserQuestion respondido nesta sessão; o próprio pedido sugeria "mudar
  o refresh token exp da sessão individual para 0").

- **Epoch e não `time.Now()`.** Nenhuma sessão legítima termina em 1970, então a
  linha continua dizendo que foi encerrada de propósito e não que venceu sozinha.
  É o único sinal que distingue os dois enquanto não existir `revoked_at` (fonte:
  decisão de implementação).

- **A troca derruba todas as sessões do usuário naquele app**, e não só a atual. A
  identidade mudou, então nenhum token emitido sob o e-mail antigo continua
  valendo, e "ele pode já tentar o novo email" vale em todo aparelho (fonte:
  AskUserQuestion respondido).

- **`POST /auth/revoke` não recebe corpo.** Lê `session_id` do `ctx.Locals` que o
  `AuthGuard` já escreveu. Sem id no corpo não há como revogar sessão de terceiro,
  e não existe rota que liste sessões de onde um id viria (fonte: AskUserQuestion
  respondido).

- **Confirmar a troca marca `verify_email = true`.** A posse do endereço novo está
  provada pelo código consumido; deixar `false` trancaria o usuário em toda rota
  `/core` pelo `EmailVerificationGuard` logo depois da troca (fonte: decisão de
  implementação, registrada como RC-3).

- **A rota de confirmação não carrega o `EmailVerificationGuard`.** Exigir e-mail
  verificado para trocar de e-mail trancaria justamente quem se cadastrou com o
  endereço errado. Nenhuma rota de `/auth` carrega esse guard (fonte: leitura de
  steering/guards-and-middlewares.md, seção EmailVerificationGuard).

- **A disponibilidade do e-mail é checada duas vezes**, ao gerar o OTP e ao
  consumi-lo. A segunda fecha a janela entre os dois passos; o índice único
  continua sendo a rede de baixo, mas ela viraria 500 em vez de 409 (fonte:
  decisão de implementação, registrada como G-3).

- **`e.ThrowUserAlreadyExists` e não `e.ThrowBadRequest`.** O ramo `REGISTER` do
  `OtpService` usa o segundo, que é inconsistente com os três outros pontos de
  colisão do serviço — a inconsistência não se propaga (fonte: leitura de
  steering/error-handling.md, "would a client write a different branch for this?").

- **A troca é uma transação, e as duas escritas vão por repositório.** E-mail
  trocado sem as sessões caírem não entrega o RF-3. Dentro da unidade de trabalho
  o `repo.Option` é o que carrega a transação, então escreve-se pelo repositório
  inclusive o de outro módulo; as **leituras** continuam pelo serviço. É a tabela
  de três casos de steering/service-layer.md, com `RegisterService.ProvisionUser`
  como referência (fonte: leitura de steering/service-layer.md, "Own the
  transaction").

- **A revogação em massa fica só no repositório, não no `ISessionService`.** Ela
  só acontece dentro da transação da troca. `Revoke` de uma sessão, esse sim, vai
  ao serviço, porque `/auth/revoke` é mutação única (fonte: a mesma tabela de três
  casos).

- **Três defeitos de `AuthorizeService.VerifyEmail` corrigidos antes da feature.**
  É o fluxo que esta copia: a condição `if !user.VerifyEmail { return nil }` está
  invertida e faz a rota nunca verificar ninguém; o OTP nunca é invalidado no
  sucesso, ao contrário do `ResetPassword`; e a resposta devolve o usuário
  desatualizado porque o serviço reatribui `user` localmente enquanto o controller
  serializa o ponteiro do `Locals` (fonte: descoberto durante a exploração, levado
  como pergunta e confirmado).

- **`/auth/verify_email` ganhou a entrada de catálogo que faltava.** Foi registrada
  no commit anterior sem chave, o que a deixava alcançável só pelo administrador da
  plataforma. Corrigida de passagem porque o catálogo estava sendo tocado de todo
  jeito (fonte: descoberta durante a exploração).

- **`LoginPermissions` ganhou `as::otp::CREATE` junto com as chaves novas.** Sem
  ele o profile padrão de toda pool declararia "pode trocar o e-mail" sem "pode
  pedir o código". Nada em `/auth` e `/otp` é enforçado hoje, então as quatro são
  expressivas e não efetivas (fonte: leitura de steering/modules/profiles.md).

## Dependências

- Nenhuma dependência de terceiros adicionada.
- Nenhuma migração: `sessions.refresh_expires_at` já é coluna e o enum
  `auth_action` já contém `CHANGE_EMAIL`.
- `NewAuthorizeService` ganha `ur.IUserRepository` e `repo.ITransactionManager`,
  então `go test ./cmd/` é obrigatório e os `nil` posicionais de
  `tests/modules/authorize/refresh.service_test.go` acompanham.
- `MockTransactionManager` e `MockUserRepository` já existem em
  `tests/modules/mock/`.

## Pendências deixadas

- `revoked_at` continua sendo o fim certo — [pendencias.md](../pendencias.md) #22,
  reescrita para registrar que o epoch é o paliativo.
- OTP de `CHANGE_EMAIL` não preso a quem o pediu — #23, o G-1 dos requisitos.
- O `409` como oráculo de existência para chamador autenticado — #24, G-2.
- Violação de índice único subindo como 500 — #25, G-3.
- O item **14** foi fechado e apagado: o fluxo verificado de troca de e-mail agora
  existe.

## O que divergiu do plano

- **`VerifyUserEmail` passou a manter o usuário em memória consistente** com a
  linha gravada, em vez de o `VerifyEmail` reler. É o mesmo `NOTE` que o
  `SetPassword` já carregava, e resolve o terceiro defeito para os dois chamadores
  de uma vez (fonte: descoberta durante a implementação).

- **`RevokedRefreshWindow` foi exportado.** O teste que prova que um refresh
  revogado é recusado precisa montar a linha que o revoke produz, e manter uma
  segunda cópia do epoch no teste é exatamente o antipadrão do documento em dois
  lugares (fonte: descoberta durante a implementação).

- **O `captureQuery` do teste de queries dedicadas não serve para uma escrita** —
  um UPDATE nunca chega ao callback de query. O arquivo já tinha `captureUpdate`
  para isso (fonte: descoberta durante a implementação).

- **A descrição OpenAPI de `/otp/generate_consumable` foi corrigida junto.** Ela
  anunciava `TWO_FA` e `REGEN_APP_SECRET_KEY` como ações aceitas, e as duas
  respondem 422 — nenhum fluxo as consome. Agora diz quais valem (fonte:
  descoberta durante a implementação).

- **Os números de `pendencias.md` viraram identidade declarada.** O item 14 foi
  fechado e os três novos entraram em seções do meio do arquivo mantendo o número
  de nascimento, porque código e steering citam itens pelo número — o comentário
  de `SessionRepository` cita o 22 (fonte: decisão de implementação).

## O que sobrou verificado

`go build ./...`, `go vet ./...` (que compila os testes e é o que pegou as quatro
quebras de mock e as três chamadas posicionais de `NewAuthorizeService`),
`go test ./cmd/` para o grafo FX, e a suíte inteira.

Testes novos: `change_email.service_test.go` (sete casos, incluindo o que prova que
as duas escritas compartilham o mesmo `repo.Option`), `revoke.service_test.go`,
`verify_email.service_test.go` (os dois lados da condição corrigida), os dois
DryRun em `dedicated_queries_test.go`, e `TestRefreshRefusesARevokedSession` ao
lado do `TestRefreshAcceptsASupersededSession` que já existia — os dois juntos são
a prova de que revogar mata o refresh token sem derrubar quem foi apenas superado.

Sem migração e sem `make fresh`: nenhuma entidade mudou. Os documentos semeados
mudaram, então um ambiente já semeado precisa de `make seed reset && make seed init`
para enxergar os grants novos.
