# 08 — Verificação

Os defeitos desta feature duram **de um a três quadros** ou **300 ms**. Leitura de código
não os prova e um teste manual costuma ser lento demais para reproduzi-los. O roteiro
abaixo é o que foi usado para dar a implementação de origem por pronta, e é o critério
de aceite para qualquer outra.

Ferramenta: um navegador headless dirigido por script (Playwright, Puppeteer, Cypress,
WebDriver — qualquer um que digite com atraso controlado e leia o DOM). Os trechos abaixo
são Playwright porque é o mais curto; a lógica vale para todos.

## 8.1 Dois ambientes de teste

| Ambiente | O que verifica | Como |
| --- | --- | --- |
| **Primitivo isolado** (Storybook, página de demonstração, fixture) | Teclado, foco, caminho, filtro padrão — sem servidor | Uma story com dados estáticos e um subescopo em memória |
| **Aplicação com API simulada** | Debounce, número de requests, parâmetros, destino, permissão, estado entre aberturas | Interceptar a rede no navegador (`page.route`) e devolver JSON fixo; semear a sessão no `localStorage` |

Não teste contra o backend real: o teste precisa contar requests e controlar o conteúdo.

## 8.2 Leitura de estado

Um helper que fotografa a paleta a cada passo. Tudo que os cenários checam sai daqui:

```js
const snap = () => page.evaluate(() => {
  const input = document.querySelector("[role=dialog] input");
  return {
    value: input?.value,
    placeholder: input?.placeholder,
    focusedIsInput: document.activeElement === input,
    chips: [...document.querySelectorAll("[role=dialog] [data-path-segment]")].map(e => e.textContent),
    sections: [...document.querySelectorAll("[role=dialog] [role=menu] header")].map(e => e.textContent),
    items: [...document.querySelectorAll("[role=menuitem]")].map(e => e.dataset.key),
    highlighted: document.querySelector("[role=menuitem][data-focused]")?.dataset.key ?? null,
    empty: document.querySelector("[role=menu]")?.textContent.slice(0, 80),
  };
});
```

Adapte os seletores à sua marcação. `data-path-segment` é um atributo que vale
acrescentar aos chips só para isto.

**Mova o mouse para fora do diálogo** antes de ler o destaque (`page.mouse.move(0, 0)`):
um ponteiro parado sobre a lista destaca o item sob ele e invalida a leitura.

## 8.3 Cenários do primitivo isolado

| # | Passos | Esperado |
| --- | --- | --- |
| P1 | abrir | `value: ""`, `focusedIsInput: true`, todos os itens, nenhum destacado (ou o primeiro, conforme a biblioteca) |
| P2 | digitar `pool` (60 ms entre teclas) | só "User Pools" visível, destacado |
| P3 | `Tab` imediatamente após P2 | `chips: ["User Pools"]`, `value: ""`, placeholder do escopo, itens do subescopo, `focusedIsInput: true` |
| P4 | digitar `sta` | item "Staff" destacado |
| P5 | `Backspace` ×3 | `value: ""`, chips intactos, lista completa do escopo |
| P6 | `Backspace` com vazio | `chips: []`, lista raiz |
| P7 | digitar `over`, `Tab` | nada muda: mesmo item, mesmos chips, `focusedIsInput: true` (Overview não tem escopo) |
| P8 | `Esc`, reabrir, `↓` `↓`, `Tab` | entra em "App Clients" (o item alcançado só por setas) |
| P9 | após P8, `↓` **uma** vez | primeiro item do escopo destacado (armadilha C) |
| P10 | após P9, `Enter` | diálogo fechou (`[role=dialog]` ausente) |
| P11 | paleta montada com filtro padrão: digitar `keys` | "App Clients" visível (palavra-chave casou) |
| P12 | digitar `zzz` a mais | vazio com a mensagem de "nothing matches" |

P3 é o teste da armadilha B (`04`): o `Tab` sem pausa depois de digitar. Se ele passar
só com `waitForTimeout(600)` antes do `Tab`, a implementação está lendo
`aria-activedescendant`.

## 8.4 Cenários da aplicação com API simulada

Interceptação: toda rota da API devolve JSON fixo; cada request de listagem é registrada
em um array `requests` com caminho e query string. A sessão é semeada no
`localStorage` com um usuário cujo documento de permissões inclui todas as grants de
leitura.

| # | Passos | Esperado |
| --- | --- | --- |
| A1 | `Ctrl+K` | paleta aberta, placeholder da raiz, seções "Pages" e "Settings", `requests` vazio |
| A2 | digitar `custo` (60 ms entre teclas) | **imediatamente**: lista vazia com "Searching…"; `requests` ainda vazio |
| A3 | esperar 700 ms | seções "App Clients" e "User Pools" com os resultados; `requests` = exatamente `[/apps?name=custo&limit=5, /users_pool?name=custo&limit=5]` — **uma por recurso**, não uma por tecla |
| A4 | `↓` até a pool, `Tab` | `chips: ["User Pools", "Customers"]` (o caminho completo, vindo da reserva), `value: ""`, `requests` = `[/users?pool_id=<id>&limit=10]` **sem `name`** (RC-03) |
| A5 | digitar `ali`, esperar 700 ms | um item; `requests` = `[/users?name=ali&limit=10&pool_id=<id>]` |
| A6 | `Enter` | `page.url()` termina em `/pools/<id>?selected-user=<userId>`; um `[role=dialog]` aberto com os dados do usuário |
| A7 | `Esc` (fecha o modal) | URL sem `?selected-user` |
| A8 | `Ctrl+K` | `chips: []`, `value: ""`, lista raiz (RF-17) |
| A9 | digitar `app`, `Tab` | `chips: ["App Clients"]`, lista com 10 itens, `requests` contém `/apps?limit=10` sem `name` |
| A10 | `↓`, `Enter` | URL = `/apps/<id>` |
| A11 | `Ctrl+K`, digitar `qqq`, esperar 800 ms | vazio: "Nothing matches “qqq” in pages, App Clients or User Pools." |
| A12 | repetir A1–A3 com um usuário **sem** `users_pool:read` | só a seção "App Clients"; nenhuma request para `/users_pool` |
| A13 | repetir A9 com um usuário sem `apps:read` | o item "App Clients" não mostra a dica e `Tab` não entra |

## 8.5 O que medir além do DOM

- **Contagem de requests por tecla.** A3 é o teste do debounce. Se `requests` tiver
  cinco entradas para "custo", o `name` está sendo lido do texto imediato.
- **Ausência de `name` na primeira request do escopo.** A4 e A9. Se tiver `name`, a
  regra do texto vazio (`05`) não está aplicada.
- **Ausência de request para recurso sem permissão.** A12.
- **`focusedIsInput` depois de cada `Tab`.** Se virar `false`, o `stopPropagation` falta.

## 8.6 Verificação visual

Não é automatizável com proveito; faça à mão, uma vez por tema:

- [ ] Tema claro e escuro: chips, destaque, dica `Tab`, spinner, rodapé.
- [ ] Marca em cor extrema (branca, preta), se o projeto tem marca configurável: o destaque
      do item não pode depender da marca.
- [ ] Nome longo num chip: trunca, o campo mantém largura mínima.
- [ ] 10 resultados: a lista rola, campo e rodapé ficam fixos.
- [ ] Leitor de tela (um): digitar → setas → `Tab` → setas → `Enter`, com o item e o
      escopo anunciados.

## 8.7 Verificação estática

O mínimo do projeto (lint, tipos, build) mais:

- a story/demonstração do primitivo cobre: montagem pronta, caminho com dois níveis,
  estado carregando, estado vazio;
- o catálogo de componentes (ou equivalente) descreve as peças, o contrato de `Tab` e as
  três armadilhas de `04`;
- o registro de dívida lista os recursos fora da paleta e o porquê (`02`).
