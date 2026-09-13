# Rotas de update e contador de cadastros na users_pool

> **Status: implementado.** Rotas, mescla de JSON, grants e contador estão no
> código. Os pontos abertos estão no fim.

## O problema

Quase nenhuma entidade tinha rota de edição. `PUT /core/apps/:id` estava
registrada mas o handler devolvia `Not Implemented Yet`, e o `UpdateApp` daquele
módulo declarava todos os campos obrigatórios e sem ponteiro — que é exatamente o
que não serve para um update. `users_pool`, `users` e `organizations` não tinham
rota de edição nenhuma. `profiles` e `participants` já tinham `PUT`, e
`ProfileService.UpdateForOrganization` é a implementação de referência que o resto
seguiu.

## As três garantias

### 1. Ponteiro em todo campo de DTO de update

Go não tem null nos tipos de valor, então `*T` é a única forma de separar "não
enviado" de "enviado com o valor zero". Sem isso, `{"private": false}` e um corpo
sem `private` são o mesmo struct, e o `PUT` ou não consegue escrever `false`, `0`
e `""`, ou sobrescreve toda coluna que o chamador não mencionou.

A regra, os detalhes de validação (`omitnil`, o `min=1` junto do `dive`) e a
relação com o dao estão em
[docs/steering/models-layer.md](../../../steering/models-layer.md).

### 2. Colunas JSON abertas são mescladas, nunca sobrescritas

`utils.MergeJsonPatch` implementa RFC 7386 (JSON Merge Patch): objetos são
mesclados chave por chave, `null` apaga a chave, e qualquer outro valor substitui.
Array é valor, não documento — é substituído inteiro.

A mescla fica no **serviço**. O repositório escreve as colunas que recebe;
transformar dois documentos em um é processamento do que o chamador mandou. Um
repositório que mesclasse também teria que ler a linha antes, virando um
read-modify-write que não é dele.

Duas limitações, ambas propositais e verificadas empiricamente:

- **Limpar o objeto inteiro não é expressável.** `{"metadata": null}` deixa um
  campo `*json.RawMessage` em `nil`, indistinguível de ausente: o `encoding/json`
  zera o ponteiro antes de chegar no `UnmarshalJSON` do `RawMessage`, e o Fiber v3
  usa `encoding/json`. Então `null` na coluna quer dizer "não toque", e a exclusão
  é chave por chave.
- **Patch que não é objeto é recusado**, não aplicado. A RFC substituiria o
  documento pelo escalar; toda coluna jsonb aqui é documentada como objeto, então
  o helper devolve `ErrPatchNotAnObject` e o serviço responde `400`.

### 3. Só quem participa da organização que criou a entidade pode editá-la

Cada rota reusa o lookup escopado que já existia, e nenhuma delas responde `403`
para "não é sua" — sempre `404`, senão o id vira oráculo para as entidades das
outras organizações.

| Rota | Regra de escopo | De onde vem |
| --- | --- | --- |
| `PUT /core/apps/:id` | `app.organization_id == organização atual` | `AppService.FindById` |
| `PUT /core/users_pool/:id` | `pool.organization_id == organização atual` | `UserPoolService.FindByIdInOrganization` |
| `PUT /core/users/:id` | a pool do alvo pertence à organização atual | `UserService.assertPoolInOrganization` |
| `PUT /core/users/me` | nenhuma: o alvo é o próprio chamador | `AuthGuard` |
| `PUT /core/organizations/:id` | `id == organização atual` | igual ao `UpdateParticipant` |
| `PUT /core/profiles/:id` | visível para a organização atual | já existia |

**A rota de administração de usuário não tem escape para o próprio usuário.**
`UserService.FindById` tem um branch que devolve o usuário antes de checar a pool
quando o alvo é o chamador — é o que faz `/core/users/me` funcionar na leitura.
Herdar esse branch na escrita tornaria o self-update irrestrito, então
`UpdateForOrganization` usa `findInOwnedPool` e não `FindById`.

## A chave reservada `record` — substituída

Esta seção descrevia uma chave `record` reservada dentro de `users.metadata`, com
`constants.UserReservedMetadata` e `utils.FindReservedKey` recusando um payload que
a nomeasse. **Ela foi removida.** O histórico de login que ia ocupá-la virou coluna
própria — `users.tracking` — e a proteção passou a ser a fronteira que já existia:
uma coluna ausente do update dao nunca é escrita, qualquer que seja o payload.

Ver [2026-09-09-tracking.md](../2026-09-09-tracking/spec.md), que explica por que a coluna
ganhou da chave no blob.

`utils.MergeJsonPatch` continua no lugar: é o que mescla o `metadata` do cliente nas
cinco rotas de update desta feature.

## Grants

Quatro chaves novas em `shared/permissions/grants.go`:

| Grant | Rota |
| --- | --- |
| `as::users::UPDATE` | `PUT /core/users/:id` |
| `as::users::me::UPDATE` | `PUT /core/users/me` |
| `as::users_pool::UPDATE` | `PUT /core/users_pool/:id` |
| `as::organizations::UPDATE` | `PUT /core/organizations/:id` |

`as::apps::UPDATE` e `as::profiles::UPDATE` já existiam.

Acrescentar chave ao catálogo é mudança de permissão: o profile `Admin` de toda
organização é `{"grants": ["as::*::*"]}`, então todo dono de organização alcança as
quatro no instante em que elas entram — limitado pelo teto da organização, o que
mantém uma org com teto LOGIN de fora. A saída de `GET /core/grants` muda junto.

O porquê da separação `as::users::UPDATE` / `as::users::me::UPDATE` está em
[docs/steering/modules/profiles.md](../../../steering/modules/profiles.md).

## O contador da users_pool

`users_pool.users_count`, `not null default 0`, **monotônico**: cresce no cadastro
e nunca decrementa. Conta cadastros feitos até agora, não linhas existentes — vai
divergir de `COUNT(*)` na primeira exclusão de usuário, e isso é o comportamento
escolhido.

`UserPoolRepository.IncrementUsersCount` faz `users_count = users_count + 1` no
banco. Não pode passar pelo dao de ponteiros — `count + 1` é expressão, não valor —
e não pode ser read-modify-write em Go, que perderia todo cadastro que caísse entre
as duas instruções. Usa `UpdateColumn`, então um cadastro não move o `updated_at`
da pool: aquela coluna quer dizer "a pool foi configurada".

`RegisterService.ProvisionUser` incrementa **dentro da transação que já é dele**, e
como **última instrução antes do commit**. A razão da ordem:
`UPDATE users_pool SET users_count = users_count + 1` pega `FOR NO KEY UPDATE` na
linha da pool. Isso não conflita com o `FOR KEY SHARE` que as FKs dos inserts de
`organizations` e `users` já pegam na mesma linha, mas conflita com outro
incremento concorrente — então cadastros simultâneos na mesma pool serializam
naquela linha até o commit, e deixar o incremento por último encurta a janela de
uma transação inteira para uma instrução.

**Invariante para o futuro:** qualquer coisa que atualize `users_pool` **e** insira
em `users`/`organizations` na mesma transação tem que tomar a escrita em
`users_pool` por último.

O seed cria um usuário admin à mão, então o literal da pool em
`cmd/database/init.go` já nasce com `UsersCount: 1`.

## Ordem de registro das rotas

Duas rotas novas caem na classe de problema do `/core/grants` antes de
`/core/profiles/:id`: o Fiber casa por ordem de registro, por método, e uma rota
com parâmetro engole o literal de mesma forma.

- **`PUT /core/organizations/:id` depois de `PUT /core/organizations/switch`.**
  Registrada antes, `ctx.Route().Path` passa a ser `/core/organizations/:id`,
  `as::organizations::switch::UPDATE` deixa de casar, e **todo usuário cadastrado
  perde a troca de organização** — LOGIN_PROFILE tem `switch::UPDATE`, não
  `organizations::UPDATE`.
- **`PUT /core/users/me` antes de `PUT /core/users/:id`.**

`tests/modules/routes/route_order_test.go` trava as duas lendo
`server.GetRoutes()` sobre os controllers de verdade. Um `PermissionsGuard` que
casa em `ctx.Route().Path` não dá 404 num literal engolido — ele resolve para a
chave de permissão errada e responde 403 para quem tem o grant do literal, que é
por que isso precisa de teste e não de comentário.

## O que ficou de fora, e por quê

| Campo | Decisão |
| --- | --- |
| `apps.users_pool_id`, `apps.parent_app_id` | Fora do DTO **e do dao**. Os usuários de um app moram na pool dele, então mover o app deixaria todos atrás |
| `apps.public_key`, `apps.secret_key` | Nunca editáveis. A pública é derivada do id, a secreta é write-once na criação |
| `users_pool.users_count` | Fora do dao. É contador de cadastro, não pode ser settável por HTTP |
| `users.password_hash` | Nunca vem de payload; só é derivado pelo hash service no fluxo de esqueci-a-senha |
| `organizations.profile_id` | É o teto da organização. Mover teto por rota precisa do clamp que o ponto aberto 11 de [2026-08-23-scoped-profiles.md](../2026-08-23-scoped-profiles.md) deixou aberto |
| `organizations.owner_user_id` | Transferir organização tem que mover a participação na mesma transação: é fluxo próprio |
| `profiles.key` | Já era imutável; é o handle que o seed e os payloads resolvem |

## Riscos aceitos

- **`apps.token_type` continua editável, e trocá-lo invalida o formato de todo
  token vivo daquele app** — o `AuthorizeService` ramifica em `app.TokenType`.
  Invalidar as sessões do app na troca (`SessionRepository.InvalidateAllExcept`
  existe) é feature própria. Está na description da rota.
- **`apps.private` virando `true` passa a exigir `X-Secret-Key` no mesmo app com
  que o chamador talvez esteja autenticado.** Dá para se trancar para fora. Está na
  description da rota.
- **`users.email` não tem fluxo verificado.** `constants.ActionChangeEmail` existe
  mas não é usado em lugar nenhum, então `PUT /core/users/:id` é a única forma de
  trocar email hoje, e é uma rota de administração — o próprio usuário não troca o
  seu. Colisão dentro da pool responde `409`, checada no serviço para não sair como
  violação de índice único num 500.

## Pontos abertos

1. **`LOGIN_PROFILE` pode editar a si mesmo mas não pode se ler.** Ele ganhou
   `as::users::me::UPDATE` e não tem `as::users::me::READ`, então
   `GET /core/users/me` responde 403 para um usuário cadastrado. É lacuna anterior
   a esta feature — o profile nunca teve a leitura — e alargar a leitura não é
   mudança de rota de update, por isso ficou fora. Uma linha no seed resolve.
2. **`IUserRepository.Create` é público**, então nada estrutural impede um quinto
   chamador de criar usuário fora do `ProvisionUser` e furar o contador. Se o
   número precisar ser exato, o caminho é fechar a criação atrás do serviço.
3. **`users_pool` não tem rota de exclusão**, então `users_count` nunca é
   reconciliado. Se algum dia precisar bater com `COUNT(*)`, a alternativa sem
   coluna é uma query dedicada com subselect e um campo `gorm:"->"`.
4. **A validação de corpo continua rodando antes dos guards** em todas as rotas
   novas, seguindo a ordem existente — a inconsistência já registrada em
   [controller-layer.md](../../../steering/controller-layer.md) vale para elas também.
