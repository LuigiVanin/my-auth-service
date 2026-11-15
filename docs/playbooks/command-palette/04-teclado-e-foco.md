# 04 — Teclado e foco

A paleta é um componente de teclado. Tudo o que o mouse faz é um atalho para algo que o
teclado já faz. Este arquivo fixa o contrato e explica os três mecanismos que, em
qualquer biblioteca de acessibilidade, vão tentar quebrá-lo.

## 4.1 O contrato

| Tecla | Na raiz | Dentro de um escopo | Observações |
| --- | --- | --- | --- |
| `⌘K` / `Ctrl+K` | abre; se aberta, fecha | fecha | Atalho global. Não dispara dentro de campos de texto de outras telas? **Dispara** — é o único atalho com modificador que sempre vale |
| `↑` `↓` | move o destaque | move o destaque | Foco real fica no campo (foco virtual) |
| `Enter` | ação do item destacado | ação do resultado destacado | Fecha a paleta depois. Sem item destacado, nada |
| `Tab` | entra no escopo do item destacado, **se** ele tiver um | idem (subescopo) | Sem escopo: comportamento padrão do diálogo (foco segue). **Nunca** `preventDefault` sem entrar |
| `Shift+Tab` | padrão do diálogo | padrão | Não interceptado |
| `Backspace` | apaga texto | com texto: apaga; **vazio: sobe um nível** | Subir reinicia o Search (4.4) |
| `Esc` | fecha | fecha (não sobe um nível) | Subir é `Backspace`; `Esc` sempre fecha, por previsibilidade |
| clique no item | = `Enter` | = `Enter` | Clique nunca entra num escopo; é preciso um controle visível se quiser isso com mouse |
| clique fora | fecha | fecha | |

Decisões que valem explicar no registro da feature:

- **`Enter` age, `Tab` entra.** Alternativa considerada: `Enter` entra quando o item tem
  escopo. Rejeitada porque torna `Enter` imprevisível (às vezes navega, às vezes não) e
  tira o caminho mais curto para a página de listagem.
- **`Esc` não sobe nível.** Quem está dois níveis abaixo e quer sair aperta `Esc` uma vez.
  Subir é raro e tem `Backspace`.
- **`Shift+Tab` fica livre.** É a saída de foco para quem usa teclado; o diálogo precisa
  dela.

## 4.2 Foco virtual

Em toda implementação acessível de "campo + lista", o foco do DOM **fica no campo** e o
item destacado é marcado de outro jeito: o campo carrega `aria-activedescendant` com o
id do item, e o item recebe uma marca (atributo `data-focused`, classe, `aria-selected`).
Isso é o que faz o leitor de tela anunciar o item sem o cursor sair do campo.

Consequência: **o `keydown` de `Tab` chega ao campo, não ao item.** Para descobrir em
qual item o usuário está, o campo precisa olhar para a lista. Há duas fontes possíveis, e
elas divergem no pior momento (4.3).

## 4.3 Os três mecanismos que quebram o `Tab`

Estes três apareceram na implementação de origem com React Aria. Os nomes mudam de
biblioteca para biblioteca; o mecanismo não. Para cada um: sintoma, causa, correção, e
como verificar na sua stack.

### A. A biblioteca não entrega o `Tab` ao item

**Sintoma.** `Tab` no item destacado move o foco para o próximo elemento focável (o
rodapé, o botão de fechar) em vez de entrar no escopo.

**Causa.** O componente de autocomplete repassa setas e `Enter` ao item, mas deixa `Tab`
passar para o gerenciador de foco do diálogo (focus trap), que o consome num listener
registrado em `document`.

**Correção.** Interceptar `Tab` no `keydown` do **campo**, resolver o item destacado,
perguntar à lista se ele tem escopo (`onTab(id) → boolean`). Só se a resposta for `true`:
`preventDefault()` **e** `stopPropagation()`. O `stopPropagation` é obrigatório porque o
focus trap escuta em `document`; sem ele, o foco ainda pula.

**Como verificar.** Abra a paleta, destaque um item com escopo, aperte `Tab`: o caminho
tem que ganhar um segmento **e** `document.activeElement` tem que continuar sendo o
campo. Destaque um item sem escopo, aperte `Tab`: o foco tem que ir para o próximo
focável do diálogo (não pode ficar preso).

### B. O atributo de foco está atrasado em relação à tela

**Sintoma.** Digitar "pool" e apertar `Tab` imediatamente entra no item errado, ou em
nenhum — mas esperar meio segundo e apertar funciona.

**Causa.** Ao digitar, a biblioteca destaca o primeiro resultado na tela na hora, mas
**atrasa** a escrita de `aria-activedescendant` (500 ms no caso de origem) para não
interromper o anúncio da letra digitada pelo leitor de tela. O `Enter` interno da
biblioteca usa uma referência própria, já atualizada; o seu `keydown` de `Tab`, lendo o
atributo, vê o valor antigo.

**Correção.** Ler o item destacado **da lista**, pela marca visual (`[data-focused]`,
`[aria-selected=true]`, a classe de destaque), não do `aria-activedescendant`. A lista
está acessível pelo `aria-controls` do campo:

```
listId = input.getAttribute("aria-controls")
key    = document.getElementById(listId)?.querySelector("[data-focused]")?.dataset.key
```

Se a sua biblioteca não expõe nenhuma marca no DOM, exponha uma você mesmo no render do
item (`data-focused={isFocused}`).

**Como verificar.** Digite três letras com 50–60 ms entre elas e aperte `Tab` sem pausa.
Tem que entrar no item que estava destacado na tela. Um teste manual "digito e aperto"
geralmente é lento demais para pegar isso; use um script (`08`).

### C. A biblioteca guarda o item focado da lista antiga

**Sintoma.** Depois de entrar num escopo (ou subir um), a **primeira** seta para baixo não
faz nada; a segunda destaca o primeiro item.

**Causa.** O gerenciador de foco virtual guarda o id do último item destacado. Ao trocar a
lista inteira (novo escopo), esse id aponta para um elemento que não existe mais. A
primeira seta serve só para a biblioteca descobrir que o item sumiu e limpar o estado.
Normalmente não há API pública para limpar isso.

**Correção.** **Remontar** o componente de autocomplete (campo + lista) a cada troca de
escopo: uma chave de identidade incrementada por `Tab` aceito e por `Backspace` que sobe.
O estado da aplicação (caminho, texto) fica **acima** desse componente, então sobrevive à
remontagem; o campo volta focado pelo `autoFocus`.

Isso fica **no primitivo** (a peça Search expõe `restart()` e a peça Input chama depois de
`Tab` e `Backspace`), não na camada 3 — quem usa a paleta não deve precisar saber.

**Como verificar.** Entre num escopo, aperte `↓` **uma** vez: o primeiro item tem que
ficar destacado. Faça o mesmo depois de `Backspace` subindo um nível.

## 4.4 A sequência exata de um `Tab` aceito

Para quem implementa do zero, a ordem importa:

1. `keydown` no campo, `key === "Tab"`, sem `Shift`.
2. Resolver `key` do item destacado pela marca na lista (4.3-B). Sem item: retornar sem
   tocar no evento.
3. Perguntar à lista: `accepted = onTab(key)`. A lista decide (tem escopo? pode entrar?)
   e, se sim, **já executa** a transição de estado (empurra o escopo, zera o texto).
4. Se `accepted`: `preventDefault()`, `stopPropagation()`, `search.restart()`.
5. Se não: nada. O evento segue e o foco se move como o diálogo mandar.

Por que a lista decide e não o campo: o campo não conhece os itens; a lista tem o mapa
`id → item` e é ela quem está por cima da camada 3.

## 4.5 `Backspace` que sobe

1. `keydown` no campo, `key === "Backspace"`, `value === ""`, `path.length > 0`.
2. `onPopPath()` → camada 3 faz `path.pop()`.
3. `search.restart()`.
4. Não chamar `preventDefault` — com o campo vazio, o padrão já não faz nada, e o
   `keyup` de alguns leitores de tela espera o evento intacto.

Com `value !== ""`, não tocar no evento: apagar texto é o padrão.

## 4.6 O atalho global

- Escutar `keydown` em `window`/`document`, `(metaKey || ctrlKey) && key === "k"`,
  `preventDefault` (o navegador usa `⌘K` para a barra de busca em alguns casos).
- **Alternar**, não só abrir: `⌘K` com a paleta aberta fecha.
- Atalhos de **uma tecla** (como `/` para focar uma busca de tela) precisam ignorar
  eventos vindos de campos de texto, menus e diálogos abertos. `⌘K` não precisa: é um
  acorde com modificador e vale em qualquer lugar.

## 4.7 Foco ao abrir e ao fechar

- Ao abrir: foco no campo (`autoFocus` no campo, não no diálogo).
- Ao fechar: o foco volta para o elemento que estava focado antes (todo diálogo acessível
  faz isso; confirme que o seu faz).
- Depois de `Enter` que navega: a paleta fecha **antes** da navegação acontecer de fato
  (ou no mesmo tick), para não deixar um diálogo órfão sobre a tela nova.
