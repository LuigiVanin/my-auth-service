# Command palette com busca aninhada — playbook de implementação

Instruções para implementar, em **qualquer codebase**, uma paleta de comandos
(`⌘K` / `Ctrl+K`) que:

1. abre com uma lista local de destinos (páginas, ações, configurações) filtrada a cada
   tecla;
2. quando nada local casa com o texto, **busca no servidor** nos recursos principais,
   com debounce;
3. com `Tab` sobre um item pesquisável, **entra nele** e passa a buscar dentro
   (`App Clients /`, `User Pools / Customers /`), em quantos níveis o domínio tiver;
4. com `Enter`, executa a ação do item — por padrão, navegar até ele.

O playbook nasceu de uma implementação real (React 19 + React Aria + TanStack Query,
sobre uma API REST em Go) e foi escrito para ser **agnóstico**: a parte normativa não
cita biblioteca nenhuma; a referência mostra o código real como exemplo, não como
exigência. Serve tanto para um projeto que não tem paleta nenhuma quanto para um que tem
uma lista estática e quer crescer.

## Como usar

A pasta é autocontida: copie-a inteira para o repositório de destino (por exemplo em
`docs/playbooks/command-palette/`). Os links entre arquivos são relativos.

- **Pessoa implementando**: leia na ordem numérica. `01` e `02` antes de escrever
  qualquer código; `03` a `07` durante; `08` antes de dar por pronto.
- **Agente de IA**: envie o texto de [PROMPT.md](PROMPT.md). Ele manda o agente seguir o
  protocolo de `09` §9.1 (inventário → lacunas → plano aprovado → um nível por vez, cada
  um medido com `08`) e fixa as restrições que não se negociam. `10` é a lista de
  verificação dos defeitos conhecidos. Os arquivos em `referencia/` são exemplo de uma
  stack específica, não exigência.
- **Projeto com paleta parcial**: comece por `09`, que mapeia o que já existe para as
  peças daqui e diz o que trocar.

Os arquivos `01`–`10` não citam linguagem, framework nem biblioteca como requisito; onde
um nome aparece, é para ilustrar um mecanismo que existe em todas ("o equivalente de
`enabled` no seu cliente de cache").

## Mapa

| Arquivo | O que contém | Quando ler |
| --- | --- | --- |
| [01-requisitos.md](01-requisitos.md) | Glossário, atores, requisitos funcionais, não funcionais, regras e invariantes, escopo, lacunas a resolver antes de codar | Antes de tudo |
| [02-backend.md](02-backend.md) | O contrato que a API precisa oferecer: listagens com filtro por nome, paginação, ordenação estável, recursos aninhados, permissões, erros, e o que fica para depois | Antes do frontend — o que faltar aqui decide o que a paleta não consegue fazer |
| [03-arquitetura-frontend.md](03-arquitetura-frontend.md) | As três camadas (primitivo, escopos, integração), o modelo de estado, onde cada coisa mora e por quê | Antes de criar arquivos |
| [04-teclado-e-foco.md](04-teclado-e-foco.md) | O contrato de teclado, foco virtual, como interceptar `Tab` sem quebrar acessibilidade, e os três comportamentos de biblioteca que quebram tudo | Ao escrever o campo e a lista |
| [05-busca-e-dados.md](05-busca-e-dados.md) | Filtro local × busca no servidor, debounce, a busca de reserva, cache compartilhado, limites, estados de carregamento e erro | Ao ligar a paleta à API |
| [06-escopos.md](06-escopos.md) | O descritor de escopo, escopos aninhados, mapeamento de resultado, destino sem rota, como adicionar um recurso novo | Ao modelar o domínio |
| [07-ui-e-acessibilidade.md](07-ui-e-acessibilidade.md) | O caminho visível, dicas de tecla, placeholder por escopo, textos de vazio, rótulos ARIA, tema | Ao desenhar |
| [08-verificacao.md](08-verificacao.md) | O roteiro de testes: o que medir, scripts de navegador headless, lista de aceite | Antes de dar por pronto |
| [09-caminhos-de-adocao.md](09-caminhos-de-adocao.md) | Do zero, de uma paleta estática, de uma biblioteca pronta (`cmdk`, `kbar`, Headless UI…); tabela de equivalências | Projeto que já tem algo |
| [10-armadilhas.md](10-armadilhas.md) | Os defeitos que apareceram de verdade, com sintoma, causa, correção e como detectar na sua stack | Durante e ao revisar |
| [referencia/pseudocodigo.md](referencia/pseudocodigo.md) | A máquina de estados, o casamento, a busca de reserva e o Tab em pseudocódigo sem framework, mais os invariantes para conferir uma implementação | Qualquer stack |
| [referencia/implementacao-react-aria.md](referencia/implementacao-react-aria.md) | O código de origem (React + React Aria + TanStack Query), anotado com a regra que cada trecho materializa | Stack parecida |
| [PROMPT.md](PROMPT.md) | A mensagem pronta para entregar a pasta a um agente | Ao colar em outro projeto |

## Níveis de adoção

A feature tem três níveis, e cada um é entregável sozinho. Não pule para o 3 sem o 1 e
o 2 estarem redondos: os defeitos de foco e de debounce aparecem no 2 e ficam
escondidos pelo 3.

| Nível | O que entrega | Depende de |
| --- | --- | --- |
| **1 — destinos locais** | Lista estática de páginas e ações, filtro por rótulo e palavras-chave, `Enter` navega, `Esc` fecha, atalho global | Nada no backend |
| **2 — busca de reserva** | Sem casamento local, busca os recursos principais pelo nome, com debounce; resultado vira destino | Listagens com filtro por nome (`02`) |
| **3 — busca aninhada** | `Tab` entra num recurso e busca dentro; níveis encadeados (pool → usuários); `Backspace` sobe | Listagens filtráveis por recurso pai (`02`) |

## Definição de pronto

A feature está pronta quando **todas** as linhas abaixo são verdadeiras e foram
verificadas no navegador (não só lidas no código) — o roteiro está em `08`:

- [ ] Digitar não dispara request enquanto algo local casa; digitar algo que não casa
      dispara **uma** request por recurso depois do debounce, não uma por tecla.
- [ ] `Tab` logo depois de digitar entra no item **destacado na tela**, não no anterior.
- [ ] `Tab` sobre um item sem escopo não faz nada visível **e** não move o foco para fora
      do campo.
- [ ] Depois de entrar num escopo, a **primeira** seta para baixo já destaca o primeiro
      item.
- [ ] Entrar num escopo pede a primeira página **sem** o texto que estava no campo antes.
- [ ] `Backspace` com o campo vazio sobe um nível; com texto, apaga texto.
- [ ] `Enter` num resultado do servidor abre o destino certo, inclusive quando o destino
      não tem rota própria.
- [ ] Fechar e reabrir a paleta volta ao nível raiz, com o campo vazio.
- [ ] Cada busca só aparece para quem tem permissão de leitura do recurso.
- [ ] Palavras-chave casam (ex.: "keys" encontra "App Clients").
- [ ] Textos de vazio distinguem "procurando", "nada casou" e "erro".
- [ ] A documentação do projeto (catálogo de componentes, rotas, dívida) foi atualizada.
