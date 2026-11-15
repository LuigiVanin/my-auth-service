# 05 — Busca e dados

Dois mecanismos de busca convivem na paleta e **não se misturam**: o filtro local, a cada
tecla, e a busca no servidor, só com o texto assentado. Este arquivo fixa quando cada um
roda, o que mostra enquanto espera, e como reaproveitar o cache das telas.

## 5.1 Filtro local

- Roda a cada tecla, sobre os destinos locais (páginas, ações, configurações).
- Casa pelo **rótulo ou por qualquer palavra-chave** do item.
- Sem distinção de caixa nem de acento (`Intl.Collator` com `sensitivity: "base"`, ou a
  função de comparação da sua biblioteca de i18n).
- **Substring**, não prefixo e não fuzzy. Fuzzy parece bom em demo e produz resultados
  inexplicáveis para o usuário ("por que *Organization settings* apareceu para `ogs`?").
  Se o produto quiser fuzzy, é uma troca da função de casamento, não da arquitetura.
- Com o campo vazio, mostra tudo.

**A função de casamento é uma só e é exportada pelo primitivo.** A camada 3 a importa para
filtrar os destinos e para decidir se deu zero (`RC-01`). O primitivo a usa como filtro
padrão para quem monta uma paleta simples.

## 5.2 Texto imediato × texto assentado

| | Texto imediato (`query`) | Texto assentado (`name`) |
| --- | --- | --- |
| O que é | `input.value.trim()` | `query` depois do debounce |
| Quem usa | filtro local, textos de vazio ("Nothing matches *x*") | toda request |
| Atraso | nenhum | 300 ms (ajustável; abaixo de 200 dispara demais, acima de 400 parece travado) |

**Regra do texto vazio.** `name = query === "" ? "" : debounced(query)`. Sem isso, ao
entrar num escopo (que zera `text`), o escopo novo recebe por 300 ms o texto do escopo
anterior como `name` e dispara uma request errada (`RC-03`). Com a regra, a primeira
request do escopo vai sem filtro na hora.

**Enquanto `query !== name`**, a UI está "entre" dois estados: mostra "Searching…" e não
"Nothing matches", mesmo que a request ainda não tenha saído.

## 5.3 A busca de reserva (nível raiz)

Dispara quando **todas** valem:

1. estamos na raiz (`path` vazio);
2. `name !== ""`;
3. `name` não casa com **nenhum** destino local (pela mesma função de casamento);
4. o usuário tem permissão de leitura do recurso.

Para cada recurso da reserva (os 2–3 principais do produto, decididos em `01`), uma
request `name=<name>&limit=5`. As requests são **habilitadas** pela condição, não
disparadas imperativamente: com um cliente de cache declarativo, isso é `enabled:
missesLocally && can(grant)`; sem ele, é um `useEffect`/watcher com cancelamento.

O que a lista mostra, em ordem de prioridade:

| Situação | Lista |
| --- | --- |
| `visible` (destinos filtrados por `query`) não vazio | os destinos |
| vazio, mas nenhum recurso permitido | vazio: "Nothing matches that. Try a page name." |
| vazio, algum recurso com erro | vazio: a mensagem do erro |
| vazio, `query !== name` ou alguma request no ar | vazio: "Searching…" |
| vazio, todas responderam, soma dos resultados > 0 | os resultados, agrupados por recurso (seção = nome do recurso no plural) |
| vazio, todas responderam, zero resultados | vazio: "Nothing matches “x” in pages, App Clients or User Pools." |

**Por que `visible` usa `query` e a condição de disparo usa `name`.** Enquanto o usuário
digita "cust", a lista local esvazia na hora (feedback imediato), mas a request só sai
quando "cust" assenta. Se a condição usasse `query`, cada tecla intermediária que não casa
("c", "cu") dispararia request antes do debounce decidir.

**Número fixo de buscas.** A lista de recursos da reserva é estática. Com hooks (React,
Vue composables), isso é o que garante um número fixo de chamadas por render: a permissão
liga ou desliga cada busca, nunca muda quantas existem.

## 5.4 A busca dentro de um escopo (nível 3)

- Uma request: `scope.query({ name, limit: 10 })`. Com `name === ""`, a primeira página
  sem filtro (`RF-11`).
- **Mantenha o resultado anterior enquanto o novo carrega** dentro do **mesmo** escopo
  (`keepPreviousData` ou equivalente). Sem isso a lista pisca vazia a cada tecla
  assentada.
- **Nunca** mantenha o resultado anterior **entre escopos** (`RC-04`). A lista do escopo é
  um componente com chave de identidade = id do escopo, então trocar de escopo remonta e
  começa sem dado.

Textos de vazio:

| Situação | Texto |
| --- | --- |
| erro | a mensagem do erro |
| carregando, ou `query !== name` | "Searching…" |
| `query` não vazio, zero resultados | "No App Clients match “x”." |
| `query` vazio, zero resultados | "No App Clients yet." |

## 5.5 Reaproveitar o cache das telas

A paleta **não tem camada de dados própria**. O descritor de escopo recebe a mesma função
que a tela de listagem usa para montar a query (`appListQuery(params)`), com os mesmos
parâmetros. Resultado:

- Mesma chave de cache → abrir a paleta depois de visitar a listagem não faz request; e
  buscar na paleta aquece o cache da listagem.
- Mesmo cliente HTTP → token, refresh em 401, cancelamento por `signal`, retry.
- Mesma tipagem → o descritor recebe a linha tipada e a converte; nenhum `any`.

A conversão linha → resultado acontece numa transformação **no nível da query** (`select`,
`map` do resultado), não no render, para que o cache guarde a linha crua e a lista receba
só o formato da paleta.

**Antipadrão:** montar `/core/apps?name=${q}` dentro da paleta. Cria um segundo caminho
de HTTP fora das políticas do projeto e um cache duplicado.

## 5.6 Limites e cancelamento

| Parâmetro | Valor | Por quê |
| --- | --- | --- |
| Reserva, por recurso | 5 | Dois ou três recursos cabem numa tela sem rolar |
| Dentro do escopo | 10 (o limite padrão das listagens) | Mesmo cache da tela; mais que isso exige paginação, que fica fora |
| Debounce | 300 ms | Ver 5.2 |
| Cancelamento | `AbortSignal` da query | Texto que mudou cancela a request anterior; sem isso, uma resposta lenta chega depois da rápida e sobrescreve |

Se o cliente de cache não cancela sozinho, implemente a regra manual: toda resposta
carrega o `name` que a originou; descarte se `name` atual for diferente.

## 5.7 Permissão

A permissão é checada **antes** de:

- habilitar a request (uma query desabilitada não roda);
- mostrar a dica `Tab` num item;
- aceitar o `Tab` (defesa em profundidade — a dica pode estar errada).

Checar na UI é conveniência: evita mostrar um caminho que termina em 403. A autorização
real é do servidor, em cada listagem.

## 5.8 Erros

- Erro de uma busca da reserva **não** esconde os resultados das outras; aparece só quando
  a lista ficaria vazia.
- Erro dentro de um escopo substitui a lista pela mensagem.
- A mensagem vem da função do projeto que converte erro de API em texto (`describeApiError`
  ou equivalente); a paleta não interpreta status HTTP.
- `401` nunca chega à paleta (a camada HTTP renova o token ou derruba a sessão).
- Sem botão "Tentar de novo": digitar de novo refaz a busca.
