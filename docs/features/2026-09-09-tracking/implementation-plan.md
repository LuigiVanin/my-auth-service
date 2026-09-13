---
status: implemented
context: tracking
created: 2026-09-13
updated: 2026-09-13
implemented: 2026-09-09
reconstructed: true
drivers: >
  Ocupa a chave reservada que a feature de rotas de update deixou plantada e nunca
  preencheu, e entrega a primeira métrica de produto do serviço — cadastros por mês
  e por app numa pool, últimos logins de um usuário. De quebra, fecha a exposição a
  panic de todas as goroutines do projeto.
---

# Implementation Plan: tracking

> Reconstruído em 2026-09-13, depois da implementação. Ver o aviso no frontmatter
> de [requirements.md](requirements.md).

## Objetivo

Uma coluna `tracking jsonb` em `users` e `users_pool`, mantida só pelo serviço, com
um método `Track` por entidade; contadores agrupados na pool e eventos individuais
no usuário; escrita disparada fora da requisição.

## Decisões tomadas

- **Coluna dedicada, não `metadata` nem tabela.** A pergunta foi levantada pelo
  usuário e levada com as três opções. A coluna ganhou por três razões: a proteção
  deixa de ser regra nova e passa a ser a fronteira que o update dao já era; o
  `PUT` do cliente e o tracker param de escrever no mesmo campo; e o dado volta
  junto com a entidade, ao contrário da tabela. A tabela foi descartada porque a
  capacidade de `GROUP BY` que ela daria não tem consumidor — nenhum endpoint
  agrupa por app ou por mês (fonte: `AskUserQuestion`; usuário escolheu a coluna).

- **A chave reservada foi removida junto.** Com a coluna própria, a maquinaria de
  `FindReservedKey` ficou sem nada para guardar — era aspiracional, nada escrevia
  `record`. Saiu o pacote de constantes, a função, os dois call sites, os testes e
  os parágrafos de documentação. `MergeJsonPatch` ficou, porque mescla o `metadata`
  do cliente nas cinco rotas de update (fonte: consequência direta da decisão
  acima, flagrada na pergunta e aceita pelo usuário).

- **Eventos individuais no login, abrindo exceção à regra do próprio pedido.** O
  pedido dizia "nunca armazenar itens individuais" e também "guardando os últimos
  10 logins" — as duas coisas não coexistem. Levado como pergunta; o usuário
  escolheu eventos individuais. Registrado como exceção consciente, não esquecimento
  (fonte: `AskUserQuestion`).

- **Lock de linha com read-modify-write, divergindo da forma do `users_count`.** O
  precedente do contador existe para não perder escrita concorrente, e o lock honra
  esse princípio; o que não se sustentava era a *forma*. `users_count` é escalar e
  `+ 1` é expressão de uma linha; isto é documento aninhado com remoção FIFO, e o
  equivalente em SQL puro seria uma torre de `COALESCE(…) || jsonb_build_object(…)`
  — porque `jsonb_set` não cria níveis intermediários — mais um segundo comando com
  `jsonb_each`/`jsonb_object_agg` para podar em dez. O projeto não tem nenhum
  precedente de `jsonb_set`, e o `Track` roda em goroutine, então ninguém espera
  pelo lock (fonte: decisão de implementação, com a alternativa avaliada e
  descartada por legibilidade).

- **Período chaveado por objeto em `YYYY-MM`, e em UTC.** Ordem lexicográfica igual
  a ordem cronológica é o que permite a remoção escolher o mês mais antigo
  ordenando as chaves. UTC para o bucket não depender do fuso do processo (fonte:
  decisão de implementação).

- **`utils.Detach` em vez de `go` pelado.** O recoverer do Fiber cobre a pilha do
  handler, não a de uma goroutine que ele gera, então um panic ali derruba o
  processo. **Nenhum dos cinco `go` do projeto tinha `recover`** — incluindo o
  `go this.otpService.Invalidate(...)`, em dois lugares. Os cinco passaram pelo
  helper, e uma corrida de dados no closure do `AuthorizeService.Refresh`, que
  atribuía ao `err` da função envolvente, foi corrigida junto (fonte: descoberta
  durante a implementação; o nome `Detach` foi escolhido depois de o usuário
  recusar `Go` e sugerir `Fork`).

- **O signup dispara depois do `tx.Commit()`.** Antes dele a transação ainda segura
  a linha da pool por causa do incremento do `users_count`, e o `FOR UPDATE` da
  goroutine conflita — não é deadlock, mas é espera jogada fora. É a mesma
  invariante de ordem de lock que a feature anterior registrou (fonte: análise da
  interação de lock feita durante a implementação).

- **O login dispara em quatro pontos e não no funil `CreateNew`**, porque o funil
  também é alcançado pelo refresh, que foi excluído, e `Refresh` reaproveita o
  `LoginType` da sessão — de dentro do funil ele é indistinguível de um login. O
  custo está registrado como pendência (fonte: `AskUserQuestion` sobre os gatilhos
  + leitura do código de `CreateNew`).

- **O payload do login é construído de forma síncrona, antes da goroutine.**
  `session.User` é atribuído logo abaixo em todos os quatro sítios, e ler a struct
  de outra goroutine enquanto esta escreve seria corrida (fonte: descoberta durante
  a implementação).

- **`users.tracking` é `json:"-"`; o da pool serializa e a listagem apaga.** A
  assimetria é a diferença de conteúdo: o do usuário é histórico de ip e
  `entity.User` está embutido em login, cadastro, refresh e listagem. Com `json:"-"`
  esquecer de apagar não vaza nada, que é a diferença para apagar campo por campo.
  O da pool são contadores agregados e a organização que a enxerga é a dona dela
  (fonte: `AskUserQuestion` sobre as duas leituras).

- **`omitempty` não esconde coluna jsonb** — omite `json.RawMessage` só em tamanho
  zero, e `default:'{}'` são dois bytes. Serve apenas para omitir um valor que o
  código zerou de propósito, que é o caso da listagem de pools (fonte: descoberta
  durante a implementação).

## Dependências

- `shared/tracking/` — pacote novo, cálculo puro, sem dependência.
- `shared/utils/async.go` — `Detach`, novo.
- `UserService` ganhou logger e transaction manager; `UserPoolService` ganhou
  transaction manager; `RegisterService` ganhou `IUserPoolService`; `LoginService`
  ganhou `IUserService`. Todas mudanças de construtor, então `go test ./cmd/`.
- `tests/modules/mock/transaction.mock.go` — mock novo.

## Pendências deixadas

Itens 8 a 12 de [pendencias.md](../pendencias.md): o `login_type` errado no OTP, o
quinto caminho de login, a ausência de reconciliação e poda, o `Create` público, e
a falta de leitura agregada.

## O que sobrou verificado

Build, vet e as quinze suítes, inclusive sob `-race`. O detector encontrou corridas
no próprio código de teste — uma goroutine detached sobrevive ao teste que a
iniciou e lia campos da suíte que o teste seguinte reatribuía — corrigidas tirando
o estado compartilhado. Fica a nota de que o `recover` do `Detach` engole também o
panic de mock não esperado, então stub faltando em caminho detached não quebra
teste: a suíte de register stuba os dois `Track` e asserta por canal.
