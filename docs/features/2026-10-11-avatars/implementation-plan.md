---
status: planned
context: avatars
created: 2026-10-11
updated: 2026-10-11
implemented:
drivers: >
  Usuários, organizações e aplicações não têm imagem, e o frontend já reserva o
  lugar dela. O servidor é um Raspberry Pi sem folga de memória nem de banda, o que
  obriga o upload direto ao S3 e a entrega por CDN — e essa é a primeira integração
  do serviço com armazenamento de objetos, então a abstração que nasce aqui é a que
  todo arquivo futuro vai usar.
---

# Implementation Plan: avatars

## Objetivo

Entregar o fluxo de três passos — autorizar, enviar direto ao S3, confirmar — para
o avatar do usuário autenticado, da organização atual e das aplicações dela; a
coluna `avatar_key` nas três tabelas, escrita só por método dedicado; `avatar_url`
em toda resposta que carrega uma das três entidades; e a abstração de
armazenamento atrás da qual o SDK da AWS fica escondido. A infraestrutura é a de
[aws-console-setup.md](aws-console-setup.md) e é pré-requisito do teste de ponta a ponta, não do código.

## Decisões tomadas

### Arquitetura

- **O binário nunca passa pelo backend.** O cliente envia direto ao S3 com um
  *presigned POST*; o backend assina, confere e promove. É o que o Raspberry Pi
  comporta e é o que trava tamanho no próprio S3 (fonte: pedido original, RNF-1 e
  RNF-2).

- **Presigned POST com *policy conditions*, não presigned PUT.** Só o POST aceita
  `content-length-range`; uma URL de `PUT` assinada não tem como limitar tamanho,
  e a política do bucket também não. O formulário prende a chave exata, o tipo
  exato, o intervalo de tamanho, o `Cache-Control` e expira em 5 minutos (fonte:
  comportamento do S3, registrado como RC-2).

- **A chave vai para o banco; a URL é montada na resposta.** `avatar_key` é a
  coluna; `avatar_url = public_base_url + "/" + avatar_key` é derivada. Trocar
  domínio, CDN ou provedor não toca uma linha (fonte: RNF-6, decidido na sessão).

- **Chave versionada, cache imutável, nenhuma invalidação.** Cada autorização gera
  uma versão aleatória; um avatar trocado é uma URL nova. Por isso o objeto leva
  `Cache-Control: public, max-age=31536000, immutable` e a CDN nunca precisa ser
  invalidada. A borda reconfere em 24 h pela *cache policy*, para que um avatar
  apagado suma sem invalidação (fonte: RC-4 e RC-12).

- **Dois prefixos, `pending/` e `avatar/`, com papéis distintos.** O upload cai em
  `pending/`, que a CDN não enxerga e o ciclo de vida apaga em um dia; a
  confirmação copia para `avatar/`, o único prefixo que a política do bucket libera
  à distribuição. Um upload abandonado nunca fica público (fonte: RNF-4, RNF-5).

- **Infraestrutura pelo console, JSONs versionados em `aws-console-setup.md`.** CDK ficou
  fora; os documentos de política estão no repositório para a configuração ser
  reproduzível e revisável. O roteiro é um passo a passo executável, com caixas
  de marcar, valores a anotar e um teste manual de ponta a ponta antes de
  qualquer código (fonte: resposta à decisão 2 — "a resposta é por console mesmo",
  com o pedido de deixar o passo a passo registrado nesta pasta).

### Onde cada coisa vive

- **`shared/storage/` é a abstração, com o adaptador S3 em
  `shared/storage/adapters/`, montado em `infra/bootstrap/`.** É o mesmo lugar e o
  mesmo desenho de `shared/email`: um cliente de serviço externo não é um módulo
  de domínio, e o bootstrap é onde a configuração vira cliente. A interface é
  `IStorage` com cinco operações — `PresignUpload`, `Head`, `Copy`, `Delete`,
  `PublicUrl` — e nenhum módulo importa o SDK (fonte: leitura de
  `shared/email` e `steering/module-structure.md`, RNF-7).

- **`avatar` é um módulo `api` em `app/modules/avatar/`.** É um ponto de entrada
  HTTP que compõe `user`, `organization` e `app`; não possui tabela. Ele carrega o
  fluxo genérico uma vez e conhece os sujeitos por um adaptador interno — o que o
  RNF-8 pede: um sujeito novo é um adaptador novo, não uma cópia do fluxo (fonte:
  regra "an HTTP entry point composing other modules means an api module" de
  `steering/module-structure.md`).

- **O sujeito é uma interface interna do módulo `avatar`**, com três
  implementações (`users/me`, `organizations/{id}`, `apps/{id}`), cada uma com
  duas responsabilidades: **resolver** a entidade para o chamador, reutilizando a
  checagem de escopo da rota de edição correspondente, e **gravar** a chave. É o
  que mantém o RC-6 verdadeiro sem duplicar a regra de escopo de nenhum módulo
  (fonte: leitura de `UserService.UpdateSelf`,
  `OrganizationService.UpdateForCaller`, `AppService.FindById`).

- **A escrita da chave vai pelo serviço do módulo dono**, não pelo repositório.
  Não há transação — a confirmação é Head, Copy, `UPDATE` e Delete, e dois desses
  são no S3 — então é o caso "mutação única fora de unidade de trabalho" da tabela
  de três casos: `IUserService`, `IOrganizationService` e `IAppService` ganham
  `SetAvatarKey(id, key *string) error`, cada um um envelope do `WriteAvatarKey` do
  seu repositório (fonte: `steering/service-layer.md`, a tabela de três casos).

- **`WriteAvatarKey` é método dedicado de repositório, e `avatar_key` fica fora
  dos daos.** Nenhum payload alcança a coluna, seja qual for o corpo. `nil` limpa.
  Ganha um DryRun em `tests/shared/dedicated_queries_test.go` por tabela (fonte:
  `steering/models-layer.md`, "A column the service owns", RC-7).

- **A política de arquivo é dado, não código espalhado.** O módulo `avatar` declara
  uma `storage.Policy{Prefix, PendingPrefix, MaxBytes, ContentTypes, GrantTTL,
  CacheControl}` e a entrega ao `IStorage` em cada chamada. Um tipo de arquivo novo
  é outra `Policy` (fonte: RNF-9).

### Contrato

- **Nove rotas, três por sujeito**, todas sob `/core` com a cadeia completa de
  guards (`AuthGuard`, `EmailVerificationGuard`, `OrganizationGuard`,
  `PermissionsGuard`):

  | Rota | Resposta | Faz |
  | --- | --- | --- |
  | `POST /core/users/me/avatar/upload` | `201 UploadGrant` | assina |
  | `PUT /core/users/me/avatar` `{ key }` | `200 entity.User` | confirma |
  | `DELETE /core/users/me/avatar` | `200 entity.User` | remove |
  | `POST /core/organizations/{id}/avatar/upload` | `201 UploadGrant` | |
  | `PUT /core/organizations/{id}/avatar` | `200 entity.Organization` | |
  | `DELETE /core/organizations/{id}/avatar` | `200 entity.Organization` | |
  | `POST /core/apps/{id}/avatar/upload` | `201 UploadGrant` | |
  | `PUT /core/apps/{id}/avatar` | `200 entity.App` | |
  | `DELETE /core/apps/{id}/avatar` | `200 entity.App` | |

  `/core/users/me/avatar*` é registrado no controller do módulo `avatar`, que
  precisa entrar **antes** de qualquer `/core/users/:id` — a regra de ordem de
  `tests/modules/routes/route_order_test.go` passa a cobrir estas também (fonte:
  anti-padrão "A parameter route registered before the literal it shadows").

- **`UploadGrant`** é o corpo da autorização:

  ```json
  {
    "key": "pending/avatar/users/42/9f1c3a7e2b4d6081.webp",
    "url": "https://vanin-auth-service-storage.s3.us-east-1.amazonaws.com",
    "fields": { "key": "…", "policy": "…", "x-amz-signature": "…", "Content-Type": "image/webp", "Cache-Control": "…" },
    "expires_at": "2026-10-11T18:05:00Z",
    "max_bytes": 2097152
  }
  ```

  O cliente monta um `FormData` com cada entrada de `fields` **na ordem recebida**,
  acrescenta o arquivo por último no campo `file`, e faz `POST` em `url`. A resposta
  `204` do S3 é o sucesso. O corpo do `POST …/upload` é `{ "content_type": "image/webp" }`,
  validado com `oneof` (fonte: formato do presigned POST do S3; `steering/models-layer.md`,
  enumerações com `oneof`).

- **`avatar_url` é montada resposta por resposta, nos DTOs, e a entidade não a
  conhece.** As três entidades ganham só `AvatarKey *string
  gorm:"column:avatar_key;default:null" json:"-"`. Cada resposta que hoje devolve
  uma das três passa a devolver uma *view* em `{module}/models/{module}.dto.go`
  que embute a entidade e acrescenta `AvatarUrl *string json:"avatar_url"` — o
  mesmo desenho de `GetUserResponse` com `tracking`. `null` quando não há chave, e
  não omitido: o frontend lê `avatarUrl ?? undefined` sem adivinhar. A URL vem de
  `storage.IUrlResolver` (um método, `PublicUrl(key *string) *string`), injetado
  nos controllers que respondem essas entidades e passado ao construtor da view,
  como `dto.NewUserView(user, urls)` (fonte: resposta à decisão 1 — "passar um por
  um seja ideal, mesmo que dê mais trabalho e corra o risco de esquecimento";
  a alternativa recusada era um hook `AfterFind` com resolvedor global).

- **As entidades aninhadas também são sombreadas.** `entity.User` serializa
  `current_organization`, `entity.App` serializa `organization` e `owner_user`,
  `entity.Organization` serializa `owner_user`. Cada view sombreia o campo
  aninhado com a view correspondente, ou o caminho aninhado sai sem `avatar_url`.
  O inventário de respostas a tocar é o primeiro passo da implementação e entra
  aqui quando feito (fonte: leitura das três entidades).

- **O risco de esquecimento é coberto por teste, não por disciplina.** Um teste
  em `tests/modules/routes/` percorre o documento OpenAPI construído e falha se
  algum schema de resposta — inclusive aninhado — tiver os campos de uma das três
  entidades sem `avatar_url`. Uma resposta nova que devolva `entity.User` cru
  quebra a suíte em vez de chegar ao frontend sem avatar (fonte: decorrência da
  decisão 1, que aceitou o risco; o documento OpenAPI já é construído nos testes
  de rota).

- **Os grants são os que já existem.** `as::users::me::UPDATE`,
  `as::organizations::UPDATE` e `as::apps::UPDATE` ganham as rotas novas no
  catálogo. Nenhum grant novo, logo nenhum perfil semeado muda e `profiles.md` só
  registra a extensão. Quem edita, muda o avatar; separar depois é tirar as rotas
  para uma chave própria (fonte: resposta "isso" à pergunta de escopo, confirmada
  na decisão 3 — "se o user tem a capacidade de realizar UPDATE nas features ele
  pode atualizar o avatar da mesma").

- **Erros reutilizados, nenhum código novo.** `400` para chave fora do prefixo do
  sujeito ou `content_type` fora da lista; `404` para chave que não existe em
  `pending/` (expirou, nunca foi enviada, ou já foi confirmada); `422` para objeto
  que existe mas excede o limite ou tem tipo diferente do assinado — não deveria
  acontecer com a policy do S3, mas a conferência é a segunda barreira; `500` para
  falha do S3 em qualquer passo (fonte: `steering/error-handling.md`, "would a
  client write a different branch for this?").

### Configuração

- **Bloco `storage` e bloco `avatar` em `config.Config`**, com os três
  carregadores (YAML, `.env`, ambiente) preenchendo os dois. As credenciais
  recebem o mesmo tratamento de `resend_api_key`. `public_base_url` sem barra
  final; o resolvedor junta (fonte: `infra/config/env.go`, [aws-console-setup.md](aws-console-setup.md)).

- **Sem `storage` configurado, o serviço sobe e as rotas de avatar respondem
  `501 NOT_IMPLEMENTED`.** O adaptador é opcional no grafo FX; um ambiente de
  desenvolvimento sem bucket continua funcionando para todo o resto (fonte:
  decisão de implementação — a alternativa, recusar o boot, pune quem não mexe em
  avatar).

## Dependências

- `github.com/aws/aws-sdk-go-v2`, `…/config`, `…/credentials` e `…/service/s3`,
  este último em versão que exponha `PresignClient.PresignPostObject` — conferir
  a versão mínima no momento de adicionar.
- Migração: `make migrate up` cria `avatar_key` nas três tabelas (nullable,
  `AutoMigrate`). Sem seed afetado.
- Os controllers que respondem usuário, organização ou aplicação — inclusive
  `login`, `register` e `authorize` — ganham `storage.IUrlResolver` no
  construtor.
- Constructores novos no grafo: `go test ./cmd/` obrigatório. Mocks novos em
  `tests/modules/mock/`: `MockStorage`, e os três serviços ganham `SetAvatarKey`
  nos mocks existentes.
- Infraestrutura de [aws-console-setup.md](aws-console-setup.md) para o teste de ponta a ponta; a suíte
  não depende dela.

## Pendências deixadas (a abrir em `pendencias.md` na implementação)

- Sem limite de taxa para autorizações de upload — G-1.
- Troca não atômica entre S3 e banco; órfãos em `avatar/` sem reconciliação —
  G-3 e RC-10.
- Órfão em `avatar/` é público até ser apagado — G-4.
- Invalidação de CDN quando a expiração de 24 h não for aceitável — RC-12.
- Apagar o avatar junto com a entidade, quando existir rota de remoção.

## O que divergiu do plano

_(preenchido durante a implementação)_

## O que sobrou verificado

_(preenchido ao final)_
