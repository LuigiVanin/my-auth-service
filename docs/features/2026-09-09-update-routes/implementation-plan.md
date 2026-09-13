---
status: implemented
context: update-routes
created: 2026-09-13
updated: 2026-09-13
implemented: 2026-09-09
reconstructed: true
drivers: >
  Nenhuma entidade tinha rota de edição e a única registrada era um stub que
  respondia 500. Destrava a administração pelo console do frontend, e define o
  padrão de update — ponteiro em todo campo, mescla de JSON no serviço, escopo por
  organização — que toda entidade futura segue.
---

# Implementation Plan: update-routes

> Reconstruído em 2026-09-13, depois da implementação. Ver o aviso no frontmatter
> de [requirements.md](requirements.md).

## Objetivo

Entregar `PUT` para apps, users_pool, users (administração e `/me`), organizations
e profiles, com semântica de campo opcional que distingue ausente de zero, mescla
aditiva das colunas JSON, e escopo por organização que responde `404` em vez de
`403`. De quebra, um contador de cadastros na pool.

## Decisões tomadas

- **Ponteiro em todo campo de DTO de update**, e `omitnil` em vez de `omitempty`.
  Os dois funcionam — `hasValue` trata ponteiro como caso especial — mas `omitnil`
  diz a intenção (fonte: pedido explícito do usuário; verificação empírica do
  comportamento do validator na sessão).

- **A mescla é RFC 7386 (JSON Merge Patch), não deep merge qualquer.** `null` apaga
  a chave, array substitui inteiro, objeto desce um nível. Escolhida por ser padrão
  nomeado, o que dá ao nome da função (`MergeJsonPatch`) o poder de carregar a
  semântica em vez de exigir comentário (fonte: decisão de implementação; o pedido
  do usuário descrevia exatamente essa semântica sem nomeá-la).

- **Limpar a coluna inteira não é expressável, e isso é aceito.**
  `{"metadata": null}` deixa um `*json.RawMessage` em `nil`, indistinguível de
  ausente — o `encoding/json` zera o ponteiro antes de chegar no `UnmarshalJSON` do
  `RawMessage`. Verificado empiricamente na sessão antes de virar decisão. A
  exclusão é chave por chave (fonte: descoberta durante a implementação).

- **`repo.HasChanges(dao)` antes de responder 404 em zero linhas afetadas.** Um dao
  todo nil resolve para mapa vazio e o `Update` curto-circuita em `(0, nil)` sem
  tocar o banco — indistinguível de linha ausente. Sem esse branch, `PUT` com corpo
  `{}` responde 404 numa linha que existe. **`ProfileService.UpdateForOrganization`
  já tinha esse bug** e foi corrigido junto (fonte: leitura de
  `steering/repository-pattern.md`, seção "An update dao with every field nil").

- **A rota de administração de usuário não reusa `FindById`.** O branch de self
  daquele método devolve o usuário antes de checar a pool, que é o que faz
  `/core/users/me` funcionar na leitura; herdá-lo na escrita tornaria o self-update
  irrestrito. `UpdateForOrganization` usa `findInOwnedPool` (fonte: descoberta
  durante a implementação).

- **`as::users::UPDATE` separado de `as::users::me::UPDATE`.** Sem a separação,
  para um usuário comum editar o próprio nome seria preciso dar a ele o grant que
  edita todos os usuários de todas as pools da organização. Espelha a separação que
  o par de READ já tinha (fonte: decisão de implementação, derivada do modelo de
  permissão em `steering/modules/profiles.md`).

- **O contador usa `users_count + 1` no banco, não read-modify-write em Go**, e é a
  última escrita antes do commit da transação de provisionamento. A ordem importa:
  o `UPDATE` pega `FOR NO KEY UPDATE` na linha da pool, então cadastros simultâneos
  serializam ali até o commit, e deixá-lo por último encurta a janela de uma
  transação inteira para uma instrução (fonte: decisão de implementação).

- **`users_pool.description` adicionada.** O payload de criação já aceitava o campo
  e o descartava em silêncio, porque a coluna não existia. Corrigido na mesma
  passada por ser a mesma migração (fonte: descoberta durante a implementação,
  levada como pergunta e confirmada).

- **Os documentos de permissão do seed foram para `cmd/database/seeds/`.** O teste
  mantinha uma cópia à mão que **já tinha divergido em quatro grants**, e a feature
  ia acrescentar grants nos dois lugares. `init.go` é `package main` e nenhum teste
  consegue importá-lo — essa era a causa raiz da cópia (fonte: descoberta durante a
  implementação).

- **A ordem de registro de rotas virou teste.** `PUT /core/organizations/:id`
  registrado antes de `/switch` engole o literal, e como o `PermissionsGuard` casa
  em `ctx.Route().Path`, não dá 404: resolve para a chave errada e responde 403 a
  todo usuário cadastrado, que é quem tem `switch::UPDATE`. Verificado invertendo a
  ordem de propósito e confirmando que o teste falha (fonte: descoberta durante a
  implementação).

## Dependências

- `shared/utils/json.go` — `MergeJsonPatch`, novo, sem dependência externa.
- `shared/repository/update.go` — `HasChanges`, novo.
- `cmd/database/seeds/` — pacote novo, importado por `init.go` e pelo teste.
- Nenhuma dependência de terceiros adicionada.

## Pendências deixadas

- `LOGIN_PROFILE` sem `as::users::me::READ` — [pendencias.md](../pendencias.md) #5.
- `users_count` sem reconciliação — #10.
- `IUserRepository.Create` público — #11.
- Validação de corpo antes dos guards — #6.

## O que sobrou verificado

`go build`, `go vet` incluindo os testes, e as suítes. `tests/modules/register`
**não compilava na main** desde o refactor de organizations e foi reescrito.
`tests/modules/routes/` nasceu nesta feature para travar a ordem de registro e a
construção do documento OpenAPI.
