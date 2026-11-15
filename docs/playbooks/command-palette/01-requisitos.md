# 01 — Requisitos

Este arquivo é o que se escreve **antes** de codar. Copie-o para o registro de features
do seu projeto, preencha as lacunas da seção final e só então siga para `02`.

## Glossário

| Termo | Significado |
| --- | --- |
| **Paleta** | O diálogo aberto por `⌘K`/`Ctrl+K`: um campo de texto, uma lista e um rodapé de dicas |
| **Destino local** | Item que a paleta conhece sem consultar o servidor: página, ação, seção de configurações |
| **Recurso** | Entidade do domínio com listagem no servidor: App Client, User Pool, Profile, Organization… |
| **Resultado** | Uma linha de recurso vinda do servidor, já no formato de item da paleta |
| **Escopo** | Um recurso dentro do qual se busca. "App Clients" é um escopo; "a pool Customers" (buscando usuários dela) também |
| **Caminho** | A pilha de escopos ativos, mostrada antes do texto: `User Pools / Customers /` |
| **Busca de reserva** (fallback) | A busca no servidor que roda no nível raiz quando nenhum destino local casa |
| **Casamento** (match) | A função que decide se um item local corresponde ao texto |
| **Texto imediato** | O que está no campo agora |
| **Texto assentado** | O texto imediato depois do debounce — o único que vira request |
| **Foco virtual** | O item "selecionado" pelas setas enquanto o foco real do DOM continua no campo |

## Atores

- **Usuário com sessão aberta.** Vê os destinos locais que o seu perfil alcança e as
  buscas dos recursos que tem permissão de **ler**. A paleta nunca mostra um caminho
  que levaria a um 403.
- **Backend.** Autoriza cada listagem por conta própria; a filtragem da paleta por
  permissão é conveniência de UI, nunca segurança.

## Requisitos funcionais

### Nível 1 — destinos locais

- **RF-01** A paleta abre com `⌘K`/`Ctrl+K` de qualquer tela autenticada e por um botão
  visível (cabeçalho). O atalho alterna: aberta, fecha.
- **RF-02** Ao abrir, lista todos os destinos locais que o usuário alcança, agrupados por
  seção (ex.: "Pages", "Settings"), com o campo vazio e focado.
- **RF-03** O campo filtra a lista a cada tecla, pelo rótulo **e** por palavras-chave do
  item, sem distinguir caixa nem acento.
- **RF-04** `↑`/`↓` movem o destaque; `Enter` executa a ação do item destacado; `Esc`
  fecha. Clicar num item equivale a `Enter`.
- **RF-05** A ação padrão de um destino é navegar até ele. Um destino pode ter outra ação
  (abrir um modal de configurações, por exemplo); a paleta fecha depois de qualquer ação.
- **RF-06** Um destino com algo pesquisável dentro mostra uma dica `Tab` enquanto está
  destacado.

### Nível 2 — busca de reserva

- **RF-07** Quando o texto assentado não casa com **nenhum** destino local, a paleta busca
  os recursos principais pelo nome e mostra os resultados no lugar da lista, agrupados por
  recurso.
- **RF-08** A busca de reserva pede poucos resultados por recurso (5 é um bom número); a
  lista cabe numa tela sem rolar.
- **RF-09** `Enter` num resultado leva ao destino dele (`/apps/:id`). `Tab` num resultado
  que tem escopo (uma pool) entra nele, montando o caminho completo (`User Pools /
  Customers /`), não só o último segmento.
- **RF-10** Enquanto o texto imediato e o assentado divergem, ou enquanto a request está
  no ar, a lista vazia diz "Searching…", nunca "Nothing matches".

### Nível 3 — busca aninhada

- **RF-11** `Tab` sobre um destino ou resultado com escopo entra no escopo: o item vira um
  segmento do caminho, o campo esvazia, o placeholder passa a descrever a busca dentro
  daquele recurso, e a lista mostra a **primeira página** do recurso sem filtro.
- **RF-12** Dentro de um escopo, o texto assentado filtra a listagem no servidor; os
  destinos locais não aparecem.
- **RF-13** Um escopo pode ter outro dentro, derivado de um resultado: `Tab` numa pool
  entra em "usuários daquela pool". A profundidade é a do domínio, não um limite fixo.
- **RF-14** `Backspace` com o campo vazio remove o último segmento do caminho e volta à
  lista do nível acima. Com texto no campo, apaga texto normalmente.
- **RF-15** Itens cujo recurso não tem listagem filtrável por nome **não** entram no
  caminho: não mostram a dica, e `Tab` neles não faz nada.
- **RF-16** Um resultado cujo destino não tem rota própria ainda assim abre em um lugar
  útil (ex.: a tela da pool com o modal do usuário já aberto, via parâmetro na URL).
- **RF-17** Fechar a paleta descarta caminho e texto. Reabrir começa do nível raiz.

## Requisitos não funcionais

- **RNF-01** Debounce no texto que vira request (300 ms é o padrão); o filtro local
  continua a cada tecla.
- **RNF-02** A paleta reaproveita a camada de dados das telas de listagem (mesmo cliente
  HTTP, mesma chave de cache, mesmas políticas de retry e cancelamento). Nenhum request
  é montado "à mão" na paleta.
- **RNF-03** Requests de um texto que já mudou são cancelados ou ignorados; a lista nunca
  mostra o resultado de um texto antigo como se fosse do atual.
- **RNF-04** `Tab` só é consumido quando entra num escopo; em qualquer outro caso o
  comportamento padrão de foco do diálogo permanece (acessibilidade).
- **RNF-05** Leitor de tela: o campo anuncia o escopo atual; a lista anuncia o item
  destacado; o estado de carregamento é marcado (`aria-busy`).
- **RNF-06** Tema claro e escuro, e qualquer cor de marca configurável, sem par de classes
  por tema (use os tokens semânticos do projeto).
- **RNF-07** Strings de UI no idioma do produto, no vocabulário fixo do produto (ex.:
  "App Clients", nunca "apps").

## Regras e invariantes

- **RC-01 Uma única função de casamento.** A função que decide o que aparece na lista
  local é a mesma que decide se "deu zero" para disparar a busca de reserva. Duas
  implementações divergem exatamente onde importa (palavras-chave, acento) e produzem
  uma página e uma busca ao mesmo tempo.
- **RC-02 Permissão antes de qualquer request.** Um escopo ou busca de reserva só existe
  para quem tem a permissão de leitura do recurso. Sem ela, nem a dica `Tab` aparece.
- **RC-03 Texto não vaza entre escopos.** Ao entrar ou sair de um escopo, a primeira
  request do novo nível vai **sem** o texto do nível anterior, mesmo com debounce no
  meio.
- **RC-04 Resultado não vaza entre escopos.** A lista de um escopo nunca exibe o
  resultado do escopo anterior enquanto o novo carrega.
- **RC-05 Estado zerado por construção.** Caminho e texto moram dentro do diálogo, que é
  desmontado ao fechar; não há `reset()` manual a esquecer.
- **RC-06 O item que `Tab` usa é o item destacado na tela.** Não o que a biblioteca
  acha que está focado, se os dois divergirem (ver `04`).
- **RC-07 Identidade de item é única na lista inteira.** Ids de resultados são
  prefixados pelo recurso (`app:…`, `pool:…`, `user:…`) para nunca colidirem entre si nem
  com destinos locais.

## Escopo

**Dentro:** os três níveis acima; recursos cuja listagem filtra por nome; um nível
aninhado por relação pai → filho que a API expõe com filtro por id do pai; destino sem
rota via parâmetro na URL; documentação.

**Fora (registrar como dívida, não improvisar):** recursos sem filtro por nome no
backend; busca por campos além do nome (e-mail, id) quando a API os separa; mais de uma
página de resultados dentro da paleta; comandos com argumentos ("criar pool chamada X");
histórico de buscas; ranqueamento por frequência de uso.

## Lacunas a resolver antes de codar

Responda cada uma por escrito no registro da feature. As respostas mudam o desenho.

1. **Quais recursos têm listagem com filtro por nome hoje?** Liste endpoint por
   endpoint. O que não tem vai para a dívida do backend (ver `02`).
2. **Quais relações pai → filho a API expõe?** (`/users?pool_id=` sim; `/participants`
   sem filtro de nome, não.) Cada uma é um candidato a segundo nível.
3. **Todo recurso tem página de detalhe?** Para os que não têm, decida o destino (`RF-16`).
4. **Tab ou Enter entra no escopo?** Aqui: `Enter` age, `Tab` entra. Se o produto já usa
   `Tab` para outra coisa dentro do diálogo, escolha outra tecla (`→`), mas uma só.
5. **O que aparece com o campo vazio dentro de um escopo?** Aqui: a primeira página sem
   filtro. A alternativa (só a dica no placeholder) poupa uma request e perde a
   exploração.
6. **Como a UI sabe as permissões do usuário?** Precisa haver um `can(grant)` síncrono
   disponível onde a paleta renderiza.
7. **Qual é o vocabulário fixo do produto** para cada recurso, no singular e no plural?
   Entra no placeholder, nas seções e nos textos de vazio.
