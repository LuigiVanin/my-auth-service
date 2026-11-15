# 07 — UI e acessibilidade

O que a paleta mostra em cada estado, e o que ela precisa anunciar. As classes e tokens
são do seu design system; aqui vão as regras.

## 7.1 Anatomia

```
┌─────────────────────────────────────────────────────────────┐
│ 🔍  [User Pools /] [Customers /]  Search users in Customers…│  ← campo, com o caminho
├─────────────────────────────────────────────────────────────┤
│ USERS                                                       │  ← cabeçalho de seção
│  ◻  Alice Doe                                   [Tab]       │  ← item destacado + dica
│     alice@example.com                                       │
│  ◻  Bob Roe                                                 │
│     bob@example.com                                         │
├─────────────────────────────────────────────────────────────┤
│ ↑↓ to navigate  ↵ to open  Tab to search inside  ⌫ to go back  esc to close │
└─────────────────────────────────────────────────────────────┘
```

- Largura fixa confortável (≈ 36 rem), ancorada no topo da tela (≈ 12 vh), não centrada
  verticalmente: a lista cresce para baixo sem mover o campo.
- Altura máxima (≈ 70 vh) com rolagem **só na lista**; campo e rodapé fixos.
- Scrim escurecido e, se o design permitir, desfocado.

## 7.2 O caminho

- Cada segmento é um chip antes do texto, com um separador `/` entre eles e depois do
  último.
- Chips **encolhem e truncam** antes do campo encolher; o campo tem largura mínima.
- Sem controle de remoção por mouse nos chips (seria um botão a mais dentro de um campo
  de texto). A saída é `Backspace`; se o produto exigir mouse, um "×" no último chip é a
  adição mínima.
- O chip não é focável.

## 7.3 Placeholder por nível

| Nível | Placeholder |
| --- | --- |
| raiz | "Search pages, App Clients and User Pools…" (nomeia o que a reserva busca) |
| escopo | o `placeholder` do escopo: "Search App Clients by name…", "Search users in Customers by name…" |

O placeholder é a única instrução sobre o escopo atual; por isso ele diz **o que** se
busca e **por qual campo**.

## 7.4 A dica `Tab`

- Um `<kbd>Tab</kbd>` à direita do item, **só enquanto o item está destacado**. Em todos os
  itens ao mesmo tempo vira ruído.
- Só em itens cujo escopo o usuário pode entrar (permissão checada).
- `aria-hidden`: é visual; o rodapé já diz "Tab to search inside".

## 7.5 Rodapé

Dicas em `<kbd>` + texto curto. O rodapé é **contextual**: "⌫ to go back" só aparece
dentro de um escopo. Ordem: navegar, abrir, entrar, voltar, fechar.

## 7.6 Textos de vazio

Uma única área de vazio na lista, com um destes textos (matriz completa em `05`):

| Estado | Texto | Observação |
| --- | --- | --- |
| procurando | "Searching…" com spinner | enquanto `query !== name` ou request no ar |
| nada local, nenhum recurso permitido | "Nothing matches that. Try a page name." | o texto da paleta de nível 1 |
| nada em lugar nenhum | "Nothing matches “x” in pages, App Clients or User Pools." | cita o texto e **onde** procurou |
| escopo, com texto | "No App Clients match “x”." | |
| escopo, sem texto | "No App Clients yet." | a primeira página veio vazia |
| erro | a mensagem da API traduzida | sem botão de retry |

Nunca mostrar "Nothing matches" enquanto uma request pode mudar a resposta.

## 7.7 Carregamento

- Enquanto carrega **com** resultados anteriores (mesmo escopo): a lista anterior fica;
  `aria-busy` na lista. Opcional: spinner discreto no campo.
- Enquanto carrega **sem** resultados: o vazio "Searching…".
- Sem skeleton: a lista tem 5–10 linhas e troca em menos de um segundo; o skeleton
  piscaria.

## 7.8 Acessibilidade

| Elemento | Requisito |
| --- | --- |
| diálogo | `role="dialog"`, `aria-label="Command palette"`, foco contido, `Esc` fecha, foco restaurado ao fechar |
| campo | `aria-label` reflete o escopo: "Search" na raiz, "Search in User Pools / Customers" dentro; `aria-controls` → lista; `aria-activedescendant` → item destacado (a biblioteca faz) |
| lista | `role="menu"` ou `listbox`; `aria-busy` enquanto carrega |
| item | `role="menuitem"`/`option`; `textValue` = rótulo (para o leitor anunciar o nome, não o ícone); marca de destaque no DOM (`data-focused`/`aria-selected`) — é dela que o `Tab` lê (`04`) |
| cabeçalho de seção | elemento de cabeçalho da coleção, anunciado ao entrar na seção |
| dica `Tab` | `aria-hidden` |
| chips do caminho | texto normal; o `aria-label` do campo já os anuncia |
| spinner | `aria-hidden`; o texto "Searching…" é o anúncio |

Teste com um leitor de tela real pelo menos uma vez: a sequência "digitar → setas →
Tab → setas → Enter" precisa anunciar o item destacado em cada passo e o novo escopo
depois do `Tab`.

## 7.9 Tema e marca

- Todas as cores vêm dos tokens semânticos do projeto (fundo de painel, texto, texto
  secundário, borda em alfa, destaque em alfa do texto). Nenhum `dark:` par.
- O destaque do item é um alfa do texto (`foreground/5`), não a cor da marca: funciona
  com marca branca, preta ou saturada.
- Ícones herdam `currentColor` dentro de um contêiner com cor de texto explícita.
- Animação de entrada/saída pelo mecanismo do diálogo (atributos `data-entering` /
  `data-exiting` ou equivalente), não por uma biblioteca de animação por cima.

## 7.10 Idioma

Todas as strings no idioma do produto, com o **vocabulário fixo** do produto: "App Clients",
"User Pools", nunca sinônimos. As palavras-chave (`keywords`) são o lugar dos sinônimos
("apps", "pools", "roles"): o usuário pode digitar qualquer um, a tela mostra um só.
