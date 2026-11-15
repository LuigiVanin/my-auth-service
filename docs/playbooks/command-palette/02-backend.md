# 02 — O que o backend precisa oferecer

A paleta não tem endpoint próprio. Ela **reaproveita as listagens que as telas já
usam**, e por isso a qualidade da busca é exatamente a qualidade dessas listagens. Este
arquivo descreve o contrato mínimo, o que é desejável, e como registrar o que faltar.

Os exemplos são REST porque é o caso mais comum. O contrato vale igual para GraphQL
(uma query de listagem com argumentos `name`, `limit`, `parentId`) ou RPC: o que importa
é a semântica, não a forma.

## 2.1 Contrato mínimo por recurso pesquisável

Para cada recurso que vai aparecer na paleta (nível 2 ou 3), a listagem precisa de:

| Capacidade | Por quê | Exemplo |
| --- | --- | --- |
| **Filtro por nome, parcial, sem caixa** | É o único campo que o usuário digita na paleta | `GET /apps?name=cust` → `ILIKE '%cust%'` |
| **Limite de página** | A paleta pede 5 (reserva) ou 10 (escopo); sem limite ela baixa a tabela inteira | `?limit=5` |
| **Ordenação estável** | Sem `ORDER BY` determinístico, a mesma busca devolve ordens diferentes e a lista "pula" entre duas teclas | `ORDER BY updated_at DESC, id` |
| **Escopo pela sessão** | A listagem já filtra pelo tenant/organização do usuário; a paleta não manda esse id | `WHERE organization_id = <da sessão>` |
| **Autorização própria** | O backend recusa quem não pode ler; a UI só esconde | `403` com código estável |
| **Resposta com envelope** | A paleta precisa saber onde estão as linhas e, de preferência, o total | `{ total, data: [...] }` |

Se a listagem devolve as linhas num array cru, também serve; o envelope só evita um
adaptador por recurso.

### O que o filtro por nome deve fazer

- **Substring, não prefixo.** "ustom" encontra "Customers". Prefixo frustra quem lembra
  do meio do nome.
- **Sem distinção de caixa.** `ILIKE`, `LOWER(name) LIKE LOWER(?)`, collation
  case-insensitive, ou equivalente.
- **Idealmente sem distinção de acento**, se o produto tem nomes acentuados (`unaccent`
  no Postgres, collation `*_ai_ci` no MySQL). O filtro local da paleta já ignora acento;
  se o servidor não ignorar, o usuário vê comportamentos diferentes no nível 1 e no 2.
- **Texto vazio ou ausente = sem filtro.** A paleta manda a primeira página sem nome ao
  entrar num escopo (`RF-11`).
- **Escape de curingas.** `%` e `_` digitados pelo usuário não podem virar curinga do
  `LIKE`.

### Semântica do limite

Documente o que `limit=0`, negativo ou acima do máximo fazem. O caso que já causou
confusão: `limit=-1` caindo no default de 10 em vez de "sem limite". A paleta sempre
manda um valor positivo pequeno, mas quem ler o contrato precisa saber.

## 2.2 Recursos aninhados (nível 3)

Um segundo nível do caminho ("usuários **desta** pool") exige uma listagem filtrável
**pelo id do pai**, além do nome:

```
GET /users?pool_id=<id>&name=ali&limit=10
```

Regras:

- O id do pai é **obrigatório** quando a listagem não faz sentido sem ele (usuários sem
  pool). Responda `400` com mensagem clara, não uma lista vazia — a lista vazia esconde
  um bug de integração.
- O pai precisa pertencer ao tenant da sessão. Um id de fora responde `404` (ou `403`,
  mas escolha um e documente); a paleta trata como erro, não como "nenhum resultado".
- Se a relação é exposta como sub-rota (`/pools/{id}/users`), serve igual. O que importa
  é que exista **um** parâmetro que estreite pelo pai e que o filtro por nome funcione
  junto.

## 2.3 Permissões

A paleta precisa, no cliente, de uma forma **síncrona** de responder "este usuário pode
ler este recurso?". Normalmente isso vem do documento de permissões carregado no login
(uma lista de grants como `apps:read`). Dois requisitos no backend:

- O documento de permissões do usuário chega ao cliente **já resolvido** (curingas
  expandidos, herança aplicada), para a UI comparar por igualdade.
- Cada listagem exige exatamente a permissão que a UI vai checar. Se a listagem de
  usuários exige `users:read` **e** `pools:read`, a UI precisa checar as duas — registre
  a combinação no contrato.

## 2.4 Erros

A paleta mostra erros dentro da lista vazia, com um texto curto. Para isso o backend
precisa de:

- Um formato de erro estável (Problem Details ou equivalente) com um **código** legível
  por máquina (`NOT_FOUND`, `FORBIDDEN`), não só o status HTTP.
- `401` tratado pela camada HTTP do cliente (refresh de token) **antes** de chegar à
  paleta — a paleta nunca navega para o login nem trata sessão.

## 2.5 Desempenho

Cada tecla assentada no nível 2 dispara uma request **por recurso** (duas ou três em
paralelo). Isso é aceitável até algumas centenas de usuários simultâneos; acima disso,
ou com tabelas grandes:

- Índice para busca por substring: trigram (`pg_trgm` + GIN) no Postgres, `FULLTEXT` no
  MySQL, ou um índice de prefixo se decidirem trocar substring por prefixo.
- Um limite máximo no servidor (`limit ≤ 50`), para a paleta ou qualquer cliente não
  pedir mil linhas.
- Opcional: um **endpoint de busca global** (`GET /search?q=&limit=`) que devolve os
  recursos principais numa resposta só, já filtrados pelas permissões do caller. Reduz
  a busca de reserva a uma request. Não é pré-requisito; é a evolução natural quando a
  reserva começar a pesar.

## 2.6 Inventário — preencha antes de codar

Uma linha por recurso que o produto tem. É daqui que sai a lista de escopos (`06`) e a
dívida de backend.

| Recurso | Endpoint de listagem | Filtro por nome? | Filtro por pai? (qual) | Permissão de leitura | Rota de detalhe no frontend? | Entra na paleta? |
| --- | --- | --- | --- | --- | --- | --- |
| App Clients | `GET /core/apps` | sim (`name`) | `pool_id` | `apps:read` | `/console/apps/:id` | nível 2 e 3 |
| User Pools | `GET /core/users_pool` | sim | — | `users_pool:read` | `/console/pools/:id` | nível 2 e 3 |
| Users | `GET /core/users` | sim (`name`; `email` separado) | `pool_id` (obrigatório) | `users:read` | **não** (modal na pool) | nível 3, sob pool |
| Profiles | `GET /core/profiles` | sim | — | `profiles:read` | `/console/profiles/:id` | nível 3 |
| Organizations | `GET /core/organizations` | **não** | — | `organizations:read` | `/console/organizations` | **fora** até ter filtro |
| Participants | `GET /core/organizations/{id}/participants` | **não** | id da org (na rota) | `participants:read` | — | **fora** até ter filtro |

As linhas acima são o inventário do projeto de origem, deixadas como exemplo preenchido.
Substitua pelas suas.

## 2.7 O que registrar como dívida de backend

Para cada "não" na coluna "Filtro por nome?" e cada relação pai → filho que faltar,
registre no road-map do backend uma linha no formato:

> `GET /core/organizations`: filtro `name`, parcial e sem caixa, exposto no DTO de query
> e na documentação da API. A paleta de comandos usa para buscar dentro de Organizations
> com `Tab`, como já faz com App Clients e User Pools.

E, se a busca por e-mail ou id fizer falta:

> `GET /core/users`: um parâmetro de busca única (`q`) que case nome **ou** e-mail, em vez
> de `name` e `email` separados. A paleta busca usuários só por nome hoje.

A paleta é construída de modo que **ligar um recurso novo depois** seja um descritor de
escopo (`06`) e nada mais — então a dívida não bloqueia a entrega dos recursos que já
têm filtro.
