# 03 — Arquitetura do frontend

Três camadas, com uma fronteira clara entre elas. A separação é o que permite que a
primeira camada não saiba o que é um "App Client", e que a terceira não saiba como o
diálogo captura o `Tab`.

```
┌──────────────────────────────────────────────────────────────┐
│ 3. Integração da aplicação  (ConsoleCommandPalette)          │
│    - monta destinos locais a partir da navegação             │
│    - decide: lista local | busca de reserva | lista do escopo│
│    - estado: caminho (pilha de escopos) + texto              │
│    - gate por permissão                                      │
├──────────────────────────────────────────────────────────────┤
│ 2. Escopos do domínio  (command-scopes)                      │
│    - um descritor por recurso pesquisável                    │
│    - sabe a query de listagem, a permissão, o mapeamento     │
│      linha → resultado (rótulo, descrição, destino, subescopo)│
├──────────────────────────────────────────────────────────────┤
│ 1. Primitivo de UI  (CommandPalette.*)                       │
│    - diálogo, campo com caminho, lista agrupada, rodapé      │
│    - filtro padrão substituível                              │
│    - teclado: setas, Enter, Esc, Tab (delegado), Backspace   │
│    - não conhece domínio, servidor nem permissão             │
└──────────────────────────────────────────────────────────────┘
```

## 3.1 Camada 1 — o primitivo, em peças

O primitivo é um **conjunto de peças que se compõem** (Compound Components, slots, ou o
equivalente da sua stack), não um componente fechado com uma prop `items`. É isso que
permite à camada 3 injetar o caminho, o estado de carregamento e os resultados do
servidor sem o primitivo saber o que são.

| Peça | Responsabilidade | O que recebe |
| --- | --- | --- |
| **Root** | O overlay e o diálogo. Abre e fecha. **Desmonta o conteúdo ao fechar** | `isOpen`, `onOpenChange` |
| **Search** | Liga o campo à lista: texto controlado, filtro, foco virtual. Guarda o handler de `Tab` registrado pela lista e sabe **reiniciar** a si mesmo ao trocar de escopo | `filter` (padrão, ou `null` para desligar), `inputValue`, `onInputChange` |
| **Input** | O campo. Desenha os segmentos do caminho antes do texto. Intercepta `Tab` e `Backspace` | `path: string[]`, `onPopPath`, `placeholder` |
| **List** | A lista agrupada por seção. Registra `onTab` no Search. Fecha o Root depois de `onAction` | `items`, `onAction(id)`, `onTab(id) → boolean`, `emptyState`, `isLoading` |
| **Footer** / **Hint** | Dicas de teclado | `keys`, texto |
| **Montagem pronta** | Root + Search + Input + List + Footer com o filtro padrão, para o caso simples de lista local | `items`, `onAction`, `placeholder`, `emptyMessage` |

Duas decisões que parecem detalhe e não são:

- **Root e Search são peças separadas** para que o estado da camada 3 (caminho, texto)
  possa viver **entre** eles — dentro do diálogo, portanto desmontado ao fechar (`RC-05`),
  mas acima do Search, portanto preservado quando o Search se reinicia (`04`).
- **O filtro padrão existe e pode ser desligado.** Uma lista local simples usa o padrão
  (rótulo + palavras-chave). A camada 3 passa `null` e entrega a lista já filtrada,
  porque precisa saber o resultado do filtro para decidir a busca de reserva (`RC-01`).

## 3.2 Camada 2 — escopos

Um **descritor** por recurso pesquisável. É um objeto, não um componente, para poder ser
referenciado pela navegação (o item "App Clients" do menu aponta para o escopo
`appsScope`) e devolvido por um resultado (uma pool devolve o escopo "usuários desta
pool"). Detalhado em `06`.

O descritor **esconde o tipo da linha do servidor**: quem consome só vê resultados no
formato de item da paleta. Isso é o que permite ter uma lista de escopos heterogênea
(`[appsScope, poolsScope]`) sem genéricos vazando para a camada 3.

## 3.3 Camada 3 — integração

Um componente da aplicação que:

1. monta os destinos locais a partir da definição de navegação (a mesma que alimenta o
   menu), já filtrados pela permissão;
2. mantém o estado: `path: Scope[]` e `text: string`;
3. deriva `query = text.trim()` e `name = query === "" ? "" : debounce(query)`;
4. renderiza **uma** de duas listas:
   - `scope = path[last]` existe → **lista do escopo**: resultados de
     `scope.useResults(name)`;
   - não existe → **lista raiz**: destinos locais filtrados por `query`; se vazia e
     `name` não casa com nada local, a busca de reserva.
5. trata `onAction` (navegar, ou a ação especial do destino) e `onTab` (entrar no
   escopo, se existir e for permitido).

A lista do escopo é renderizada com uma **chave de identidade** igual ao id do escopo,
para que trocar de escopo troque o componente inteiro (`RC-04`).

## 3.4 Modelo de estado

```
PaletteState = {
  path: Scope[]        // pilha; vazio = raiz
  text: string         // texto imediato
}

derived:
  query  = text.trim()
  name   = query === "" ? "" : debounced(query)   // texto assentado
  scope  = path.at(-1)                            // undefined na raiz
  mode   = scope ? "scope"
         : visibleLocal(query).length > 0 ? "local"
         : "fallback"
```

Transições:

| Evento | Pré-condição | Efeito |
| --- | --- | --- |
| digitar | — | `text = valor` |
| `Tab` no item `i` | `i.scope` existe e `can(i.scope.grant)` | `path.push(i.scope)` (ou `push(origem, i.scope)` vindo da reserva), `text = ""`, Search reinicia |
| `Backspace` com `text === ""` | `path.length > 0` | `path.pop()`, Search reinicia |
| `Enter` no item `i` | — | `i.action()` ou `navigate(i.href)`, depois fechar |
| `Esc` / clique fora | — | fechar → Root desmonta → estado some |

Não há transição "limpar". Fechar destrói o estado.

## 3.5 Onde cada coisa mora

| Coisa | Mora em | Não mora em |
| --- | --- | --- |
| Caminho e texto | Camada 3, **dentro** do Root | Store global, URL, contexto da aplicação |
| A função de casamento | Camada 1, exportada; camada 3 importa a mesma | Uma cópia na camada 3 |
| Quais recursos são pesquisáveis | A definição de navegação (campo `search` apontando para um escopo) | Uma lista paralela na paleta |
| Como buscar um recurso | O descritor de escopo, reaproveitando a query da tela de listagem | Uma URL montada na paleta |
| Permissão | Checada na camada 3 **antes** de renderizar a dica e de habilitar a query; o descritor só declara qual grant | Dentro do primitivo |
| O destino de um resultado | O descritor (`href` ou ação) | A camada 3, por tipo de id |
| Abrir um modal via URL | A tela de destino lê o parâmetro | A paleta, com estado global |

## 3.6 Fluxo de dados

```
teclado ──▶ Input.onChange ──▶ text ──▶ query ──▶ filtro local ──▶ visible
                                   │
                                   └──▶ debounce ──▶ name ──┬──▶ (raiz, visible vazio) reserva: N queries, limit 5
                                                             └──▶ (escopo) scope.useResults(name), limit 10
                                                                      │
                                            cache compartilhado com a tela de listagem
                                                                      │
                                                             resultados ──▶ List.items
```

A seta que importa: **o texto imediato nunca vira request**, e **o texto assentado nunca
filtra a lista local**. Os dois existem lado a lado de propósito.

## 3.7 Dependências que a camada 3 precisa encontrar no projeto

Antes de codar, confirme que existem (ou crie, fora da paleta):

- a definição de navegação como dado (lista de `{ label, to, icon, grants, keywords }`),
  não só como JSX de menu;
- um `can(grant)` síncrono acessível em qualquer componente autenticado;
- um hook/função de debounce de valor;
- a camada de dados das listagens (uma função por recurso que devolve "como buscar",
  com chave de cache), para o descritor reaproveitar;
- um atalho global de teclado que **não** rouba a tecla de campos de texto e menus
  abertos;
- a função que converte erro de API em texto de UI.
