# 09 — Caminhos de adoção

Três pontos de partida. Em todos, a ordem é a mesma: inventário → nível 1 redondo →
nível 2 → nível 3. Pular o nível 1 esconde os defeitos de foco atrás da busca.

## 9.1 Protocolo para um agente de IA

Se este playbook foi colado num projeto para um agente implementar, siga nesta ordem e
**pare para perguntar** nos pontos marcados.

1. **Inventário do projeto** (só leitura; não crie nada ainda):
   - Framework de UI e versão; biblioteca de componentes acessíveis, se houver (React
     Aria, Radix, Headless UI, Ark, Vuetify, Angular CDK…); biblioteca de paleta já
     instalada (`cmdk`, `kbar`, `ninja-keys`…).
   - Como a navegação é definida: existe uma lista de rotas/itens como **dado**? Onde?
   - Camada de dados: cliente de cache (TanStack Query, SWR, Apollo, RTK Query, Pinia
     Colada…) ou chamadas diretas? Onde ficam as queries de listagem e com quais
     parâmetros?
   - Permissões: existe `can(grant)` ou equivalente no cliente? Onde?
   - Debounce: existe hook/utilitário?
   - Atalhos globais: existe um mecanismo? Alguma tecla já ocupa `⌘K`?
   - Conversão de erro de API em texto: existe?
   - Design system: tokens de cor, componente `kbd`, diálogo/modal padrão.
   - Testes: há navegador headless configurado? Storybook ou página de demonstração?
   - Documentação: onde o projeto registra componentes, rotas, dívida, features.
2. **Inventário da API** — preencher a tabela de `02` §2.6, lendo os endpoints de listagem
   (filtro por nome? por pai? permissão? envelope?).
3. **Lacunas de `01`** — responder as sete por escrito. ⏸ **Perguntar** o que não dá para
   inferir do código: tecla de entrar (Tab?), o que mostrar com campo vazio num escopo,
   destino de recursos sem rota, vocabulário do produto.
4. **Plano por nível** — listar arquivos a criar/alterar em cada nível, com o mapeamento
   das peças de `03` para a stack encontrada (tabela de 9.4). ⏸ **Apresentar o plano e
   esperar aprovação** antes de escrever código.
5. **Nível 1** — primitivo em peças + integração com destinos locais. Rodar os cenários
   P1–P12 de `08` (adaptados). Documentar.
6. **Nível 2** — escopos dos recursos da reserva + a lógica de reserva. Rodar A1–A3, A11,
   A12.
7. **Nível 3** — `Tab`, caminho, subescopos, destino sem rota. Rodar o resto de `08`.
8. **Fechamento** — documentação do projeto, dívida de backend registrada, relatório do
   que foi verificado e do que não foi.

Nunca monte URLs de API dentro da paleta; nunca copie a função de casamento; nunca
intercepte `Tab` sem a resposta booleana da lista. Esses três são os erros que um agente
comete quando "simplifica".

## 9.2 Do zero

Não há paleta nem diálogo de busca.

1. Escolher a base do diálogo: a biblioteca de componentes acessíveis que o projeto já
   usa. Se não usa nenhuma, prefira uma que ofereça **diálogo com foco contido + campo
   com lista e foco virtual** (padrão "combobox/autocomplete"); construir foco virtual à
   mão é a parte mais cara e mais fácil de errar.
2. Construir o primitivo em peças (`03` §3.1) com o filtro padrão.
3. Integração: destinos locais a partir da navegação, atalho `⌘K`, `Enter` navega.
4. Testar P1, P2, P8, P10–P12. Entregar como nível 1.
5. Seguir para `02` (inventário) e os níveis 2 e 3.

## 9.3 De uma paleta estática existente

Há uma paleta que lista páginas/ações e filtra no cliente (o caso de origem).

1. **Não jogue fora.** Ela vira a "montagem pronta" do primitivo e continua servindo a
   quem só quer lista local.
2. Extrair as peças (`03` §3.1) do componente fechado. Critério: o componente atual deixa
   de ter `items` como prop obrigatória da raiz; `items` passa a ser da peça List.
3. Extrair a função de casamento para um módulo exportado; usar nas duas pontas.
4. Acrescentar `filter={null}` (ou equivalente) na peça Search, `path`/`onPopPath` na
   Input, `onTab`/`isLoading`/`emptyState` na List, `hint` no item.
5. Reescrever a integração sobre as peças **com o comportamento idêntico ao anterior**,
   e rodar P1–P12 antes de tocar em servidor. Este passo intermediário é o que isola
   defeitos de refatoração dos defeitos de busca.
6. Níveis 2 e 3.

Armadilha típica: a paleta antiga filtrava **só por rótulo** e a nova filtra por rótulo
e palavras-chave. Um teste que esperava "nada" para uma palavra-chave passa a ver um
item. Isso é correção, não regressão — confira as palavras-chave da navegação.

## 9.4 De uma biblioteca pronta (`cmdk`, `kbar`, Headless UI Combobox, Radix, Ark…)

Essas bibliotecas entregam o nível 1 pronto e nenhuma entrega o 3. Dois caminhos:

- **Manter a biblioteca como camada 1** se ela expõe: texto controlado, filtro
  desligável, item com `value`/`data-*` lido no DOM, `keydown` no campo, e remontagem
  limpa. Então as camadas 2 e 3 deste playbook se aplicam por cima.
- **Trocar por peças próprias** sobre a biblioteca de acessibilidade do projeto se ela
  esconder o filtro ou o foco (as que fazem "fuzzy mágico" dentro costumam esconder).

Tabela de equivalência — o que procurar em cada biblioteca para cada peça/mecanismo:

| Conceito do playbook | O que procurar | Observação |
| --- | --- | --- |
| Root | o diálogo/modal da biblioteca com `open`/`onOpenChange` | precisa desmontar o conteúdo ao fechar; se mantém montado, zere o estado no `onOpenChange(false)` |
| Search (texto controlado) | `value`/`onValueChange` no root ou no input | obrigatório para zerar o texto ao entrar num escopo |
| filtro desligável | `shouldFilter={false}`, `filter={null}`, `filter={() => 1}` | obrigatório para RC-01 |
| marca de destaque no DOM | `data-selected`, `aria-selected`, `data-highlighted`, `data-focused`, `data-active` | é daqui que o `Tab` lê; **não** do `aria-activedescendant` |
| chave do item no DOM | `data-value`, `data-key`, `id` | se não houver, acrescente `data-key` no render do item |
| `onTab` | não existe em nenhuma; é `onKeyDown` no input + leitura do DOM | `preventDefault` + `stopPropagation` só quando aceito |
| reiniciar ao trocar de escopo | `key={generation}` no componente raiz da busca (React), `:key` (Vue), `*ngIf` com toggle ou `ViewContainerRef.clear()` (Angular) | ver `04` §4.3-C; algumas bibliotecas limpam sozinhas ao trocar `value` — teste P9 diz |
| seção/cabeçalho | `Group`/`Section` com `heading` | agrupa a reserva por recurso |
| vazio | `Empty`/`renderEmptyState` | precisa aceitar conteúdo, não só string, para o spinner |
| `keepPreviousData` | `placeholderData: keepPreviousData` (TanStack), `keepPreviousData: true` (SWR), `fetchPolicy` + `previousData` (Apollo) | só dentro do mesmo escopo |
| `enabled` | `enabled` (TanStack), chave `null` (SWR), `skip` (Apollo/RTK) | liga a reserva pela condição, nunca dispara imperativamente |

## 9.5 Sem cliente de cache

Se as listagens são chamadas diretas (`fetch` num `useEffect`, serviço com Promise), a
paleta ainda pode nascer, com um adaptador mínimo no `defineScope`:

- `useResults` vira um hook com `useState` + `useEffect` que: ignora `enabled: false`;
  dispara com `AbortController`; cancela no cleanup; guarda `{ results, isLoading,
  error }`; mantém o resultado anterior enquanto o novo carrega **só se o escopo é o
  mesmo** (o `key` da lista garante).
- Toda resposta carrega o `name` que a originou e é descartada se o `name` atual mudou.
- Registre na dívida: "migrar as listagens para um cliente de cache; a paleta deixa de
  precisar do adaptador".

## 9.6 Sem definição de navegação como dado

Se o menu é JSX solto, extraia primeiro uma lista `{ label, to, icon, grants, keywords,
search? }` e faça o menu **e** a paleta lerem dela. Sem isso, a paleta mostra uma página
que o menu não mostra (ou o contrário) na primeira mudança de rota.

## 9.7 Backend sem filtro por nome

A paleta é entregue com os recursos que têm filtro; os outros ficam fora e vão para a
dívida (`02` §2.7). **Não** filtre no cliente a primeira página de um recurso sem filtro:
parece funcionar com 8 registros e mente com 80.
