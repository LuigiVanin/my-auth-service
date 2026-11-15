# 10 — Armadilhas

Defeitos que apareceram de verdade na implementação de origem, mais os que o desenho
evita de propósito. Para cada um: sintoma, causa, correção, e como detectar na sua
stack. Use como lista de revisão.

## Encontrados no navegador

### 1. `Tab` logo após digitar entra no item errado

- **Sintoma.** Digitar e apertar `Tab` sem pausa entra no item que estava destacado
  **antes** da digitação, ou em nenhum. Com meio segundo de pausa, funciona.
- **Causa.** O `aria-activedescendant` é atualizado com atraso (500 ms) ao digitar, por
  causa do leitor de tela. O destaque visual é imediato. O `Tab` lia o atributo.
- **Correção.** Ler a marca de destaque na lista (`[data-focused]`), alcançada via
  `aria-controls` do campo. `04` §4.3-B.
- **Detectar.** P3 em `08`, com 60 ms entre teclas e `Tab` sem espera.

### 2. A primeira seta depois do `Tab` não responde

- **Sintoma.** Entrou no escopo, `↓` uma vez: nada. `↓` de novo: primeiro item.
- **Causa.** O gerenciador de foco virtual guarda o id do item da lista anterior; a
  primeira seta só limpa esse estado.
- **Correção.** Remontar o autocomplete (chave de geração) a cada entrada e saída de
  escopo, dentro do primitivo. `04` §4.3-C.
- **Detectar.** P9.

### 3. `Tab` move o foco para fora do campo

- **Sintoma.** `Tab` entra no escopo (o caminho muda) **e** o foco vai para o rodapé ou
  para o botão de fechar.
- **Causa.** O focus trap do diálogo escuta `Tab` em `document`; `preventDefault` no
  campo não o impede.
- **Correção.** `stopPropagation()` junto com `preventDefault()`, só no `Tab` aceito.
  `04` §4.3-A.
- **Detectar.** `focusedIsInput` depois do `Tab`, em P3 e P8.

### 4. Palavras-chave nunca funcionaram

- **Sintoma.** Digitar uma palavra-chave ("keys") não acha o item ("App Clients"),
  embora o código filtre por `keywords`.
- **Causa.** A função de filtro recebia o nó da coleção com `value` indefinido: o item
  não passava `value={item}` ao componente de item, e o filtro só via o `textValue`.
- **Correção.** Passar o objeto do item como `value` no item da coleção.
- **Detectar.** P11. Este estava na paleta estática original e ninguém tinha notado — por
  isso o teste da palavra-chave entra no aceite.

## Evitados pelo desenho

### 5. Texto do escopo anterior vaza para o novo

- **Sintoma.** Entrar na pool e a primeira listagem de usuários vir filtrada por "custo".
- **Causa.** `text` zerado, mas `debounced(text)` ainda com o valor antigo por 300 ms.
- **Correção.** `name = query === "" ? "" : debounced`. `05` §5.2.
- **Detectar.** A4: a request não pode ter `name`.

### 6. Resultado do escopo anterior aparece no novo

- **Sintoma.** Entrar numa pool e ver, por um instante, a lista de pools como se fossem
  usuários.
- **Causa.** `keepPreviousData` sem remontar a lista ao trocar de escopo.
- **Correção.** `key={scope.id}` no componente da lista do escopo. `05` §5.4.
- **Detectar.** Ler `items` imediatamente após o `Tab` (A4): precisa ser vazio ou já o
  novo escopo, nunca o anterior.

### 7. Uma página e uma busca ao mesmo tempo

- **Sintoma.** Digitar "keys": o menu mostra "App Clients" e a reserva dispara.
- **Causa.** Duas funções de casamento: a do menu (rótulo + palavras-chave) e a da
  condição de reserva (só rótulo).
- **Correção.** Uma função exportada, usada nas duas pontas. `RC-01`.
- **Detectar.** P11 combinado com a contagem de requests: zero.

### 8. Uma request por tecla

- **Sintoma.** `requests` com uma entrada por letra.
- **Causa.** Condição de disparo lendo o texto imediato, ou debounce aplicado depois da
  query.
- **Correção.** A query só vê `name`. `05` §5.3.
- **Detectar.** A3.

### 9. Número de hooks variável

- **Sintoma.** Erro de "rendered more hooks than during the previous render" (React) ou
  reatividade quebrada (Vue) ao ganhar/perder permissão ou ao trocar de escopo.
- **Causa.** Chamar `useResults` dentro de um `map` sobre uma lista que muda de tamanho,
  ou trocar o escopo sem remontar a lista.
- **Correção.** Lista de recursos da reserva estática, cada um com sua chamada; permissão
  só liga/desliga. Lista do escopo com `key`. `05` §5.3, `03` §3.3.

### 10. Caminho sobrevive ao fechar

- **Sintoma.** Reabrir a paleta dentro do escopo anterior, com texto.
- **Causa.** Estado em store global ou no componente de fora do diálogo.
- **Correção.** Estado num componente **dentro** do Root, que desmonta ao fechar.
  `RC-05`.
- **Detectar.** A8.

### 11. Um caminho que termina em 403

- **Sintoma.** `Tab` em "User Pools", lista vazia com erro de permissão.
- **Causa.** Escopo oferecido sem checar a permissão de leitura.
- **Correção.** `can(scope.grant)` antes da dica, antes de habilitar a query e dentro
  do `onTab`. `05` §5.7.
- **Detectar.** A12, A13.

### 12. URL montada na paleta

- **Sintoma.** Funciona hoje; quebra quando a tela de listagem muda o parâmetro, ou o
  cliente HTTP ganha um header novo, ou o refresh de token muda.
- **Causa.** `fetch("/core/apps?name=" + q)` dentro do descritor.
- **Correção.** O descritor recebe a **query da tela**. `05` §5.5.
- **Detectar.** Revisão de código: nenhuma string de URL dentro da pasta da paleta.

### 13. `Esc` subindo nível em vez de fechar

- **Sintoma.** Usuário aperta `Esc` para sair e a paleta continua aberta um nível acima.
- **Causa.** Decisão de UX "esperta" demais.
- **Correção.** `Esc` sempre fecha; subir é `Backspace`. `04` §4.1.

### 14. `Enter` que às vezes entra e às vezes navega

- **Sintoma.** O usuário não sabe prever o que `Enter` faz.
- **Causa.** `Enter` entrando no escopo quando o item tem um.
- **Correção.** `Enter` age; `Tab` entra. `04` §4.1.

### 15. Filtrar no cliente um recurso sem filtro no servidor

- **Sintoma.** A busca "funciona" em desenvolvimento (8 registros) e não acha nada em
  produção (o registro está na página 3).
- **Correção.** O recurso fica fora até o backend filtrar. `09` §9.7.

### 16. Spinner piscando

- **Sintoma.** A lista pisca vazia e volta a cada tecla assentada dentro do escopo.
- **Causa.** Sem `keepPreviousData` no mesmo escopo, ou skeleton no lugar da lista.
- **Correção.** Manter a lista anterior e marcar `aria-busy`. `07` §7.7.

### 17. "Nothing matches" durante o debounce

- **Sintoma.** Digitar e ver "Nothing matches" por 300 ms antes dos resultados.
- **Causa.** Texto de vazio decidido só por "lista vazia", sem olhar `query !== name`.
- **Correção.** A matriz de `05` §5.3 e §5.4: "Searching…" enquanto os dois textos
  divergem.
