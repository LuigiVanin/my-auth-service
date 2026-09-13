---
status: reconstructed
slug: tracking
extracted_at: 2026-09-13
implemented_at: 2026-09-09
gate: new-capability
modules: [user, user_pool, register, login, session, authorize]
reconstructed: true
reconstruction_note: >
  Escrito DEPOIS da implementação, a partir da sessão que a produziu. Vale aqui o
  mesmo aviso de update-routes: documento reconstruído descreve o código, não o que
  foi pedido e decidido. Mantido pelo histórico, marcado para não virar exemplo.
sources:
  - session:2026-09-09 (pedido original, quatro AskUserQuestion respondidos)
  - code:shared/tracking/document.go
  - code:shared/utils/async.go
  - code:app/modules/core/{user,user_pool}/services
  - code:app/modules/{register,login}/services
  - steering:models-layer.md
  - steering:repository-pattern.md
  - pr:9
---

# Requirements — Tracking por entidade

## Contexto

A feature de rotas de update deixou plantada uma chave reservada `record` dentro de
`users.metadata`, descrita como "onde o histórico de login vai ser escrito". Nada
escrevia nela. Esta é a feature que ia ocupar aquela chave, e ela cresceu: não é só
histórico de login, é métrica de tracking por entidade, com um método `Track` no
serviço de cada uma.

## Atores

| Ator | Papel |
| --- | --- |
| O próprio serviço | Único escritor do tracking; nenhum payload alcança a coluna |
| Administrador de pool | Lê contadores de cadastro da pool e o histórico de login dos usuários dela |
| Usuário | Lê os próprios últimos logins |

## Requisitos funcionais

- **RF-1** — `usersPoolService.Track(tag, payload)` e
  `userService.Track(tag, payload)`, um método no serviço da entidade a que o dado
  se refere.
- **RF-2** — A tag seleciona o que está sendo registrado: `signup` na pool,
  `login` no usuário.
- **RF-3** — A pool conta cadastros por mês e por app, com no máximo dez meses; o
  mais antigo sai quando entra um novo.
- **RF-4** — O usuário guarda os últimos dez logins.
- **RF-5** — O `Track` roda em paralelo, antes do retorno do serviço, para não
  somar latência ao cadastro nem ao login.

## Requisitos não funcionais

- **RNF-1** — Nenhum cadastro individual é guardado na pool: apenas contador e os
  parâmetros que agrupam (app e data).
- **RNF-2** — Dois tracks concorrentes na mesma linha não podem perder um ao outro.
- **RNF-3** — Falha de tracking não pode derrubar nem afetar a resposta da
  requisição que o originou.

## Regras e invariantes

- **RC-1** — A coluna de tracking não é escrevível por payload nenhum.
- **RC-2** — O tracking de signup é escrito **depois** do commit da transação de
  provisionamento, nunca dentro dela.
- **RC-3** — O histórico de ip de um usuário não pode vazar nas respostas que
  embutem a entidade de usuário — login, cadastro, refresh e listagem.
- **RC-4** — Uma tag que a entidade não mantém é erro, não no-op.

## Escopo

### Dentro

As duas colunas; o pacote de cálculo do documento; os dois `Track`; os disparos; o
helper de goroutine com `recover`; as rotas de leitura.

### Fora (exclusão explícita)

- **Refresh de token não conta como login** — é renovação automática do cliente e
  inflaria o contador.
- Tabela dedicada de tracking — não há consumidor de leitura agregada.
- Poda por tempo e reconciliação — nada corrige o tracking se ele ficar errado.

## Lacunas detectadas

- **G-1** — O disparo de login está em quatro pontos e não no funil
  `SessionService.CreateNew`, porque o funil também é alcançado pelo refresh, que
  foi excluído. Um quinto caminho de login pode esquecer. Ver
  [pendencias.md](../pendencias.md) #9.
- **G-2** — `LoginWithOtp` grava `login_type` como `WITH_PASSWORD`; o bug é
  anterior e agora aparece também no evento de tracking. #8.
- **G-3** — `sessions` já é a fonte de verdade dos logins, com ip, user agent, app
  e data por sessão. O tracking do usuário é desnormalização deliberada: sobrevive
  à invalidação e à poda de sessões e volta junto com a entidade.

## Evidências da sessão

O pedido definiu a forma do dado e a regra de agrupamento:

> "esse comportamente deve ser sempre igual no tacking, nunca armazenar itens
> individuais, apenas um contador e parâmetros gerais que vão ser agrupados, como
> app id e data"

E a execução em paralelo, com a pergunta embutida:

> "como a operação de Track pode não ser performática e não impacta no retorno do
> recurso de signup, ela deve ser feito de maneira paralela antes do retorno do
> service de signup como forma uma goroutine, viáveil?"

A resposta foi que sim, com a ressalva de que panic em goroutine derruba o
processo, o que originou o helper.

O usuário levantou explicitamente a questão de onde guardar:

> "Veja se é válido criar uma nova coluna na tabela para esse tracking ou até uma
> nova tabela, ou se só expandir no metadados é o suficiente"

Quatro perguntas foram respondidas na sessão: onde guardar, a forma do login
(eventos individuais, abrindo exceção consciente ao RNF-1), quais caminhos
disparam, e como tratar a goroutine.

## Fontes consultadas

`shared/permissions` como precedente de pacote de cálculo puro em `shared/`; o
precedente do `users_count` para escrita concorrente; a documentação do gorm para
`clause.Locking`; o comportamento do `encoding/json` com ponteiro e `null`.
