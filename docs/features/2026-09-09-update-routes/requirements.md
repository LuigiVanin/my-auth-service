---
status: reconstructed
slug: update-routes
extracted_at: 2026-09-13
implemented_at: 2026-09-09
gate: new-capability
modules: [app, user, user_pool, organization, profile, register]
reconstructed: true
reconstruction_note: >
  Escrito DEPOIS da implementação, a partir da sessão que a produziu e do código
  mergeado no PR #9. É o antipadrão que este processo existe para evitar — um
  documento reconstruído descreve o código, não o que foi pedido e decidido, e os
  dois divergem exatamente onde documentar importa. Mantido porque o histórico
  vale mais que a consistência, e marcado para que ninguém o leia como exemplo do
  fluxo correto. Da próxima feature em diante o par nasce antes do código.
sources:
  - session:2026-09-09 (pedido original, quatro AskUserQuestion respondidos)
  - code:app/modules/core/{app,user,user_pool,organization,profile}
  - code:shared/utils/json.go
  - code:shared/permissions/grants.go
  - steering:models-layer.md
  - steering:repository-pattern.md
  - pr:9
---

# Requirements — Rotas de update das entidades

## Contexto

Quase nenhuma entidade tinha rota de edição. `PUT /core/apps/:id` estava
registrada mas o handler devolvia `Not Implemented Yet`, e o `UpdateApp` daquele
módulo declarava todos os campos obrigatórios e sem ponteiro — exatamente o que não
serve para um update. `users_pool`, `users` e `organizations` não tinham rota de
edição nenhuma.

`profiles` e `participants` já tinham `PUT`, e
`ProfileService.UpdateForOrganization` era a implementação de referência.

## Atores

| Ator | Papel |
| --- | --- |
| Participante de uma organização | Edita apps, pools e organizações daquela organização |
| Administrador de pool | Edita usuários de pools que sua organização possui |
| Usuário comum | Edita a si mesmo por `/core/users/me` |

## Requisitos funcionais

- **RF-1** — `PUT` em `apps`, `users_pool`, `users`, `organizations` e `profiles`.
- **RF-2** — Todo campo do corpo é opcional, e apenas o que foi enviado é escrito.
- **RF-3** — Enviar o valor zero (`false`, `0`, `""`) escreve o valor zero, e não é
  tratado como ausência.
- **RF-4** — Colunas JSON abertas são mescladas com o que está guardado, com
  prioridade para a requisição.
- **RF-5** — Apagar chave dentro de uma coluna JSON só é possível enviando a chave
  explicitamente como `null`.
- **RF-6** — `users_pool` mantém um contador de cadastros, incrementado em todo
  caminho que cria usuário.
- **RF-7** — `PUT /core/users/me` edita o próprio usuário, sem checagem de escopo.

## Requisitos não funcionais

- **RNF-1** — A mescla de JSON acontece no serviço, nunca no repositório.
- **RNF-2** — O contador não pode perder escrita concorrente.
- **RNF-3** — Toda rota nova é alcançável por grant, e nenhuma fica acessível só
  ao administrador da plataforma por esquecimento de catálogo.

## Regras e invariantes

- **RC-1** — Só quem participa da organização que criou a entidade pode editá-la.
  Para usuário, o alvo tem que pertencer a uma pool que a organização do editor
  possui.
- **RC-2** — "Não é sua" responde `404` e nunca `403`, para o id não virar oráculo.
- **RC-3** — Uma coluna ausente do update dao não pode ser escrita por payload
  nenhum. O dao é fronteira de segurança, não conveniência.
- **RC-4** — Trocar `default_profile_id` de uma pool passa pelo mesmo teto de
  permissão que a criação: não se concede mais do que se tem.
- **RC-5** — `PUT /core/organizations/{id}` só aceita a organização atual, porque
  as permissões que autorizaram a requisição são as de lá.

## Escopo

### Dentro

Rotas de update das cinco entidades; a mescla RFC 7386; o contador da pool; os
grants novos; `users_pool.description`, que o payload de criação já aceitava e
descartava.

### Fora (exclusão explícita)

- `users_pool_id` e `parent_app_id` do app — mover o app deixaria seus usuários
  para trás.
- `public_key` e `secret_key` — derivada e write-once.
- `profile_id` e o dono de uma organização — teto de permissão e transferência,
  cada um um fluxo próprio.
- Senha — continua no fluxo de esqueci-a-senha.
- `key` de profile — é o handle que o seed resolve.

## Lacunas detectadas

- **G-1** — `LOGIN_PROFILE` ganhou `as::users::me::UPDATE` e continua sem
  `as::users::me::READ`: o usuário edita a si mesmo e não consegue se ler. Deixado
  de fora por ser alargamento de leitura, não de update. Ver
  [pendencias.md](../pendencias.md) item 5.
- **G-2** — `constants.ActionChangeEmail` existe e não é usado: não há fluxo
  verificado de troca de e-mail, então a rota de administração é a única forma.
- **G-3** — `IUserRepository.Create` é público, então nada estrutural garante que
  o contador conte todos os cadastros.

## Evidências da sessão

O pedido original nomeou o problema com precisão, e é a origem do RF-2 e do RF-3:

> "golang não tem um valor nativo para null em seus tipo […] nas nossas rotas de
> update os atributos de nossos DTOs de update devem todos ter `*` […] isso serve
> para diferenciar o 0 do nada, o "" do nada"

E o RNF-1 veio literalmente do pedido:

> "Creio que essa junção de mapas deve ficar no nível de serviço de cada feature e
> não no repositório, pois o repo deve se preocupar em apenas atualizar a entidade
> com os respectivos campos fornecidos"

O RC-1 também:

> "apenas um usuário participante de uma organização que CRIOU aquele app ou pool
> de usuários tem a permissão de realizar modificações naquela entidade"

Quatro decisões foram levadas como pergunta e respondidas na sessão: o escopo de
entidades, os campos editáveis do usuário, o destino do `description` da pool, e a
semântica do contador.

## Fontes consultadas

Steering de models, repository, controller e service; o catálogo de grants; as
specs de organizations e de scoped-profiles para o modelo de permissão.
