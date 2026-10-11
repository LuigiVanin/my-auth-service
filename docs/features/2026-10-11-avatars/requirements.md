---
status: approved
slug: avatars
extracted_at: 2026-10-11
gate: new-capability
modules: [avatar (novo, api), storage (novo, shared), user, organization, app]
sources:
  - session:2026-10-11 (pedido original com três requisitos, quatro perguntas respondidas)
  - code:infra/entities/{user,organization,app}.entity.go
  - code:app/modules/core/user/services/user.service.go (UpdateSelf, FindById)
  - code:app/modules/core/organization/services/organization.service.go (UpdateForCaller)
  - code:app/modules/core/app/services/app.service.go (FindById, Update)
  - code:shared/permissions/grants.go
  - code:cmd/database/seeds/profiles.go
  - code:shared/email (padrão de adapter)
  - code:infra/config/env.go
  - steering:module-structure.md
  - steering:models-layer.md ("A column the service owns")
  - steering:service-layer.md
  - steering:modules/profiles.md
  - frontend:steering/component-catalog.md (`Avatar`, prop `src`)
---

# Requirements — Avatares de usuários, organizações e aplicações

## Contexto

Nenhuma das três entidades tem imagem. O frontend já está pronto para receber uma:
o componente `Avatar` aceita `src={user.avatarUrl}` e cai nas iniciais quando não
há URL (`frontend/steering/component-catalog.md`, seção `Avatar`). O backend não
tem coluna, rota, SDK da AWS nem nenhuma abstração de armazenamento de objetos.

O servidor que roda a API é um Raspberry Pi 3 com 4 GB de RAM, mais da metade já
comprometida por outros serviços, atrás de uma conexão doméstica. Isso fecha a
porta ao desenho mais óbvio — receber o arquivo no backend e repassar ao S3 — e
define a arquitetura: **o binário da imagem nunca atravessa o backend**. O
navegador envia direto ao S3 com uma autorização assinada pelo backend, e a imagem
é servida por CDN a partir de um bucket que continua privado.

O bucket `vanin-auth-service-storage` (`us-east-1`, conta pessoal `961868442569`)
já existe e está vazio. Nada da conta da empresa é tocado.

## Atores

| Ator | Papel |
| --- | --- |
| Usuário autenticado | Troca ou remove o próprio avatar |
| Participante com permissão de edição | Troca ou remove o avatar da organização em que está, e o das aplicações dessa organização |
| Navegador (frontend) | Pede a autorização, envia o arquivo direto ao S3, confirma; exibe a URL |
| Backend (Raspberry Pi) | Assina autorizações, confere o objeto enviado, promove-o para a área pública, grava a chave |
| S3 | Guarda os objetos e **recusa** uploads fora das regras assinadas |
| CloudFront | Serve a área pública com cache; é a única porta de leitura externa |
| Qualquer cliente HTTP | Lê um avatar pela URL pública, sem autenticação |

## Requisitos funcionais

- **RF-1** — Um usuário autenticado consegue definir, trocar e remover o próprio
  avatar.
- **RF-2** — Quem pode editar uma organização consegue definir, trocar e remover o
  avatar dela; quem pode editar uma aplicação consegue o mesmo com a aplicação.
  "Pode editar" é a permissão de edição que já existe (`as::organizations::UPDATE`,
  `as::apps::UPDATE`), não uma nova.
- **RF-3** — Definir um avatar é um fluxo de três passos: o cliente **pede uma
  autorização de upload**, **envia o arquivo direto ao S3** com ela, e **confirma**
  ao backend informando a chave que recebeu. Só na confirmação a imagem passa a
  ser o avatar.
- **RF-4** — Toda resposta que hoje devolve um usuário, uma organização ou uma
  aplicação passa a carregar `avatar_url`: a URL pública pronta para o `src` de
  uma `<img>`, ou `null` quando não há avatar.
- **RF-5** — A URL pública de um avatar é estável enquanto ele não muda e **muda
  sempre que ele muda**, para que cache de CDN e de navegador nunca mostre uma
  imagem antiga.
- **RF-6** — Remover o avatar devolve `avatar_url: null` e apaga o objeto do
  armazenamento.

## Requisitos não funcionais

- **RNF-1** — O backend nunca recebe nem reenvia os bytes da imagem. As únicas
  chamadas que faz ao S3 por avatar são pequenas: assinar, consultar metadados,
  copiar dentro do S3 e apagar.
- **RNF-2** — O tamanho máximo de um avatar é **2 MiB**, e a trava vale **no
  próprio S3**: um envio acima disso é recusado antes de ocupar espaço, sem o
  backend participar. O limite é configuração, não constante.
- **RNF-3** — Só `image/png`, `image/jpeg` e `image/webp` são aceitos, também
  travados no S3. SVG fica de fora de propósito: pode carregar script.
- **RNF-4** — O bucket permanece com *Block all public access* ligado. A leitura
  pública passa só pelo CloudFront, só do prefixo `avatar/`, via *Origin Access
  Control*.
- **RNF-5** — Um upload autorizado e nunca confirmado **some sozinho** em até um
  dia, sem nenhum processo no backend, e nunca fica alcançável publicamente.
- **RNF-6** — O banco guarda a **chave** do objeto, nunca a URL. A URL é montada
  na resposta a partir de uma configuração (`public_base_url`). Trocar de domínio,
  de CDN ou de provedor não altera uma linha do banco.
- **RNF-7** — O acesso ao S3 fica atrás de uma interface do serviço. Nenhum módulo
  de domínio importa o SDK da AWS; os testes de serviço rodam com um mock e sem
  credenciais.
- **RNF-8** — Acrescentar um novo dono de avatar no futuro (uma pool, por exemplo)
  custa um adaptador de sujeito, um grupo de rotas e uma entrada no catálogo — o
  fluxo de três passos não muda.
- **RNF-9** — Acrescentar um novo **tipo de arquivo** no futuro (um anexo, um
  documento) custa uma política nova (prefixo, limite, tipos aceitos, tempo de
  vida) — a assinatura, a conferência e a promoção são as mesmas.
- **RNF-10** — Toda rota nova tem entrada no catálogo de grants. Nenhuma fica
  alcançável só pelo administrador da plataforma por esquecimento.

## Regras e invariantes

- **RC-1** — **Layout de chaves.** Área pública: `avatar/{sujeito}/{id}/{versão}.{ext}`.
  Área de espera: `pending/avatar/{sujeito}/{id}/{versão}.{ext}`. `sujeito` é
  `users`, `organizations` ou `apps`; `id` é o identificador público da entidade
  (o `id` numérico do usuário, o uuid da organização e da aplicação — nunca
  `users.uuid`, que é `json:"-"`); `versão` é aleatória e gerada pelo backend a
  cada autorização; `ext` deriva do tipo aceito. A confirmação move o objeto de
  `pending/` para `avatar/` mantendo o resto da chave.
- **RC-2** — **A autorização de upload prende tudo.** Ela vale para **uma chave
  exata**, um tipo exato, o intervalo de tamanho `[1, max]`, o `Cache-Control`
  fixado pelo backend, e expira em **5 minutos**. É um *presigned POST* com
  *policy conditions*, não um *presigned PUT*: só o POST consegue travar tamanho.
- **RC-3** — **Confirmar não confia no cliente.** Ao receber a chave, o backend
  exige que ela esteja dentro do prefixo `pending/avatar/{sujeito}/{id}/` do
  sujeito que a rota resolveu, consulta o objeto no S3 e recusa se ele não
  existe, excede o limite ou tem tipo fora da lista. Só então copia para
  `avatar/`, grava a chave e apaga o de `pending/`.
- **RC-4** — **A chave nunca se repete.** Cada autorização gera uma versão nova,
  então um avatar trocado é uma URL nova. Isso é o que permite
  `Cache-Control: public, max-age=31536000, immutable` no objeto e dispensa
  invalidação de CDN para sempre.
- **RC-5** — **Trocar apaga o anterior.** Confirmado o novo, o objeto antigo em
  `avatar/` é apagado. Falha nessa remoção é registrada em log e **não** desfaz a
  troca: o banco já aponta para o novo, e um objeto órfão custa centavos, uma troca
  desfeita custa um usuário confuso.
- **RC-6** — **Escopo é o das rotas de edição que já existem.** `users/me` não
  checa escopo porque o alvo é o chamador. `organizations/{id}` exige que `{id}`
  seja a organização atual, com o mesmo `403` de `UpdateForCaller`. `apps/{id}`
  exige que a aplicação pertença à organização atual, com o mesmo `404` de
  `AppService.FindById`. "Não é sua" continua sendo `404`, nunca `403`.
- **RC-7** — **`avatar_key` é coluna que o serviço possui**, nas três tabelas.
  Fica fora dos daos de atualização, é `json:"-"` na entidade e só o método
  dedicado de repositório `WriteAvatarKey` a escreve. Nenhum `PUT` consegue
  apontar um avatar para uma chave arbitrária. É a regra de
  `steering/models-layer.md`, "A column the service owns".
- **RC-8** — **`avatar_url` é derivada, nunca armazenada.** Ela é
  `public_base_url + "/" + avatar_key`, e `null` quando a chave é vazia. Sai nas
  mesmas respostas em que a entidade sai hoje — login, register, refresh,
  listagens, detalhes, e onde ela aparece aninhada em outra — sem exceção. A
  entidade não carrega o campo: cada resposta o monta explicitamente, e um teste
  sobre o documento OpenAPI é o que impede uma resposta de esquecê-lo.
- **RC-9** — **Remover é idempotente.** `DELETE` sem avatar responde `200` com o
  mesmo corpo de sempre.
- **RC-10** — **Confirmar o mesmo upload duas vezes não é erro na primeira e é
  `404` na segunda**, porque o objeto já saiu de `pending/`. Dois clientes do mesmo
  sujeito confirmando chaves diferentes ao mesmo tempo: o último grava, o primeiro
  vira órfão em `avatar/` até a remoção do RC-5 do segundo encontrá-lo — ou não.
  Aceito como limitação; ver G-3.
- **RC-11** — **O CloudFront entrega `X-Content-Type-Options: nosniff`** e força
  HTTPS. Um arquivo disfarçado de imagem não executa no navegador de ninguém.
- **RC-12** — **A remoção na CDN é por expiração, não por invalidação.** O objeto
  sai do S3 na hora; as bordas da CDN reconferem em no máximo 24 h (TTL máximo da
  *cache policy*), e o navegador que tinha a URL antiga nunca mais a recebe de
  nenhuma resposta da API. Invalidação explícita fica fora do escopo.

## Escopo

### Dentro

A infraestrutura na AWS (políticas, CORS, ciclo de vida, distribuição, usuário IAM
do backend), descrita passo a passo em [aws-console-setup.md](aws-console-setup.md); a abstração de
armazenamento com o adaptador S3; a coluna `avatar_key` nas três tabelas e o
método dedicado que a escreve; o módulo `avatar` com as nove rotas (três sujeitos
× autorizar, confirmar, remover); `avatar_url` em toda resposta que carrega uma das
três entidades; as entradas de catálogo; o handoff para o frontend.

### Fora (exclusão explícita)

- **Receber o arquivo no backend**, em qualquer variante — nem como fallback.
  Decisão da sessão, pelo RNF-1.
- **Redimensionar, recortar ou gerar variantes no servidor.** O frontend recorta e
  reduz para 512 px em webp antes de enviar, o que deixa cada avatar abaixo de
  200 KB; o limite de 2 MiB é a rede de baixo. Variantes por tamanho são
  compatíveis com o layout de chaves do RC-1 e entram quando houver demanda.
- **Infraestrutura como código (CDK, Terraform).** Criada pelo console seguindo
  [aws-console-setup.md](aws-console-setup.md), com os JSONs de política versionados ali para serem
  reproduzíveis. Decisão da sessão.
- **Domínio próprio na CDN.** Fica o `*.cloudfront.net`; `public_base_url` é
  configuração, então trocar depois é uma linha. Decisão da sessão.
- **Avatar por administração** (`PUT /core/users/{id}/avatar`, o dono da pool
  definindo o avatar de um usuário). Nenhum caso de uso pediu.
- **Limpeza de objetos quando a entidade é apagada.** Não existe rota de remoção de
  usuário, organização ou aplicação hoje; quando existir, apagar o avatar junto é
  dela.
- **Um adaptador local (disco) para desenvolvimento.** A interface permite, mas o
  bucket é barato e o DryRun não precisa dele.
- **Invalidação de CDN** (RC-12) e **reconciliação de órfãos** (RC-5, RC-10).

## Lacunas detectadas

- **G-1** — **Sem limite de autorizações por sujeito.** Um cliente autenticado pode
  pedir autorizações em série e subir 2 MiB em cada uma; nada no serviço limita
  taxa (`pendencias.md` não tem rate limiting em lugar nenhum). O dano é custo de
  armazenamento por no máximo um dia, pelo RNF-5. Fica registrado, não resolvido
  aqui.
- **G-2** — **A autorização não guarda estado.** O backend não registra que emitiu
  uma autorização; confirma qualquer chave que esteja no prefixo do sujeito e
  exista no S3. É suficiente porque só quem tem a permissão daquele sujeito
  consegue escrever naquele prefixo, mas significa que duas autorizações
  simultâneas são ambas válidas até a primeira confirmar.
- **G-3** — **A troca não é atômica entre S3 e banco.** Cópia feita e `UPDATE`
  falho deixa um objeto em `avatar/` sem dono; `UPDATE` feito e remoção do antigo
  falha deixa o antigo. Os dois são órfãos baratos e logados; não há transação que
  abranja os dois sistemas.
- **G-4** — **Um órfão em `avatar/` é público** até ser apagado, porque a política
  do bucket libera o prefixo inteiro à CDN. A chave aleatória do RC-1 o torna
  impossível de adivinhar, não impossível de alcançar por quem já tinha a URL.
- **G-5** — **`users.uuid` é `json:"-"`** e por isso não pode entrar na chave;
  o `id` numérico é público e sequencial, então a chave de um usuário revela a
  ordem de criação. A versão aleatória impede enumerar avatares; o id em si já é
  público em toda resposta.

## Evidências da sessão

Os três requisitos vieram numerados no pedido. O primeiro é a origem do RNF-2 e
do RC-2:

> "Temos que tomar cuidado com o tamanho de arquivos, não queremos nosso sistema
> sendo abusado em que usuário subam imagens muito pesadas de avatares. Portanto,
> travas para pastas específicas avatar/ seriam bem vindas se possível"

O segundo é a origem do RNF-1 e do RNF-6, e já trazia a alternativa certa:

> "Temos um raspberry pi 3 como servidor de 4gb de ram, mas que mais da metade já
> está comprometido com outros serviços rodando, é uma boa ideia transitar os
> arquivos de imagens para o servidor que roda nossa api backend ou isso poderia
> estourar nossa RAM? caso não seja, seria melhor uma abordagem de pre-signed key
> com o campo de avatar sendo uma url?"

O terceiro é a origem do RNF-4 e do RC-11:

> "Como servir essas imagens do s3 de maneira pública? usar um CDN via cloudfront?
> qual seria a melhor opção"

Quatro decisões foram levadas como pergunta e respondidas: limite de 2 MiB com
redução no frontend ("ok"), domínio genérico do CloudFront ("pode ser esse
genérico mesmo"), infraestrutura pelo console ("ok"), e escopo de permissão — o
próprio usuário para si, a permissão de edição correspondente para organizações e
aplicações ("isso"). O pedido de fechamento fixou o RNF-8 e o RNF-9:

> "Crie ela de maneira a ser expansível e flexível para não sofrermos com problemas
> comuns dessa abordagem no futuro."

## Fontes consultadas

Steering de workflow, module-structure, models-layer (a seção da coluna que o
serviço possui é a base do RC-7), service-layer e profiles; o catálogo de grants
e os perfis semeados; as três entidades; `UserService.UpdateSelf`,
`OrganizationService.UpdateForCaller` e `AppService.FindById`, que são as
checagens de escopo que o RC-6 herda; o padrão de adapter de `shared/email`, que o
armazenamento copia; o `Avatar` do catálogo de componentes do frontend.
