# Passo a passo no console da AWS — avatares

Roteiro para criar, **pelo console**, toda a infraestrutura dos avatares. Feito
uma vez, à mão, seguindo a ordem abaixo: a política do bucket cita o id da
distribuição, e a distribuição cita o bucket. Os JSONs estão aqui para que a
configuração seja reproduzível e revisável; nenhum deles é lido pelo código.

Os nomes de botões e campos são os do console em outubro de 2026. A AWS muda o
console com frequência: se um rótulo estiver diferente, o objetivo de cada passo
está escrito ao lado para guiar a escolha.

```
navegador ──(1) POST /…/avatar/upload ──▶ backend ── assina ──▶ { url, fields, key }
navegador ──(2) POST multipart ─────────▶ S3  pending/avatar/…   (recusa > 2 MiB, tipo errado)
navegador ──(3) PUT /…/avatar { key } ──▶ backend ── Head, Copy, Delete ──▶ S3 avatar/…
qualquer um ─(4) GET ───────────────────▶ CloudFront ──(OAC)──▶ S3 avatar/*
```

## Estado da execução

> [!NOTE]
> **Executado em 2026-11-14 e 2026-11-15, e conferido por leitura na conta**
> (`aws ... --profile personal-PowerUser`, só comandos de leitura, e `curl` nas
> URLs públicas). Os passos 1 a 6.3 estão completos e se comportam como o
> esperado. Fica pendente só a limpeza dos arquivos de teste (6.4).
>
> Onde a execução divergiu do roteiro, um callout no próprio passo diz o quê.

## Recursos criados

| Recurso | Identificador |
| --- | --- |
| Conta AWS | `961868442569` |
| Bucket | `vanin-auth-service-storage` (`us-east-1`) — ARN `arn:aws:s3:::vanin-auth-service-storage` |
| Regra de ciclo de vida | `Expire Pending Uploads` |
| Cache policy | `avatars-24h` — id `9d4bc7ec-d605-4d30-b985-e8cee541cffd` |
| Distribuição CloudFront | `EAB1577MJ1814` — ARN `arn:aws:cloudfront::961868442569:distribution/EAB1577MJ1814` |
| Domínio da distribuição | `d1xtj308k1n7yp.cloudfront.net` |
| Origin Access Control | `E894TCW7KFKM8` (`oac-vanin-auth-service-storage.s3.us-east-1.amazonaw-mwh7rpgvdfc`) |
| Response headers policy | `Managed-SecurityHeadersPolicy` — id `67f7725c-6f97-4210-82d7-5512b31e9d03` (gerenciada pela AWS) |
| Política IAM do backend | `auth-service-storage-avatars` — ARN `arn:aws:iam::961868442569:policy/auth-service-storage-avatars` |
| Usuário IAM do backend | `auth-service-backend` — ARN `arn:aws:iam::961868442569:user/auth-service-backend` |
| Chave de acesso do backend | Ativa, criada em 2026-11-15, termina em `…3MI7`, tag `raspberry-pi`. O id completo e o segredo ficam só no `.env.development.yaml` |

---

## Passo 0 — Conferir onde você está

- [x] Entre no console como **`htnek-AdminAccess`**. Criar usuário e política é
      IAM, e o PowerUser não tem permissão para isso.
- [x] Confira o canto superior direito: a conta tem que ser **9618-6844-2569**.
      Se aparecer `0540-8352-4026`, é a conta da empresa: saia e não continue.
- [x] Região **US East (N. Virginia) `us-east-1`** no seletor do topo. O
      CloudFront e o IAM são globais e mostram "Global"; isso é normal.

> [!NOTE]
> Todos os recursos foram criados na conta pessoal `961868442569`. A conferência
> feita pelo CLI usou o perfil `personal-PowerUser`, que estava autenticado como
> **root**, e não como `htnek-PowerUser`. Isso não muda nenhum recurso, mas o
> perfil precisa ser refeito com o usuário IAM.

---

## Passo 1 — Conferir o bucket `vanin-auth-service-storage`

*S3* → *Buckets* → clique em `vanin-auth-service-storage`.

### 1.1 Acesso público bloqueado

- [x] Aba **Permissions** → seção **Block public access (bucket settings)**.
- [x] Tem que estar **On**, com as quatro opções marcadas. Se não estiver:
      *Edit* → marque **Block all public access** → *Save changes* → digite
      `confirm`.

*Objetivo:* o bucket nunca é lido diretamente pela internet. Quem lê é o
CloudFront, por uma permissão própria (passo 4).

### 1.2 ACLs desligadas

- [x] Ainda em **Permissions** → seção **Object Ownership**.
- [x] Tem que mostrar **Bucket owner enforced** ("ACLs disabled"). Se não:
      *Edit* → **ACLs disabled (recommended)** → *Save changes*.

### 1.3 Criptografia padrão

- [x] Aba **Properties** → seção **Default encryption**.
- [x] Tem que mostrar **Server-side encryption with Amazon S3 managed keys
      (SSE-S3)**. É o padrão de buckets novos; se for outra coisa, *Edit* →
      **SSE-S3** → *Save changes*.

*Objetivo:* SSE-KMS exigiria permissão extra para o CloudFront ler. SSE-S3 não.

> [!NOTE]
> Nada precisou ser feito: o bucket já estava com `AES256` (SSE-S3), o padrão de
> buckets novos, com SSE-C bloqueado.

### 1.4 Versionamento desligado

- [x] Ainda em **Properties** → seção **Bucket Versioning**: tem que estar
      **Disabled**. Deixe assim.

*Objetivo:* a versão de cada avatar já vai no nome do arquivo. Versionamento do
S3 só guardaria cópias apagadas, com custo.

### 1.5 CORS

O navegador envia o arquivo direto ao S3, então o S3 precisa aceitar requisições
vindas do endereço do frontend.

- [x] Aba **Permissions** → seção **Cross-origin resource sharing (CORS)** →
      *Edit*.
- [x] Cole o JSON abaixo → *Save changes*.

```json
[
  {
    "AllowedOrigins": ["*"],
    "AllowedMethods": ["POST"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3000
  }
]
```

`"*"` vale **só na fase de desenvolvimento**. O risco é baixo porque o S3 recusa
qualquer envio sem uma policy assinada pelo backend, mas o CORS deixa de ser uma
segunda barreira. Quando o frontend tiver um endereço de produção, troque `"*"`
pela lista explícita de origens, por exemplo
`["http://localhost:3000", "https://app.seudominio.com"]` (a porta `3000` é a de
`frontend/vite.config.ts`). `POST` é o único método: o navegador nunca lê nem
apaga direto no bucket.

> [!WARNING]
> Aplicado com `"AllowedOrigins": ["*"]`. Trocar pela lista explícita antes de ir
> para produção.

### 1.6 Regra de ciclo de vida para `pending/`

- [x] Aba **Management** → seção **Lifecycle rules** → *Create lifecycle rule*.
- [x] **Lifecycle rule name:** `expire-pending-uploads`.
- [x] **Choose a rule scope:** *Limit the scope of this rule using one or more
      filters*.
- [x] **Prefix:** `pending/` (com a barra no fim).
- [x] Em **Lifecycle rule actions**, marque:
  - [x] **Expire current versions of objects**
  - [x] **Delete expired object delete markers or incomplete multipart uploads**
- [x] Em **Expire current versions of objects** → **Days after object creation:**
      `1`.
- [x] Na seção de delete markers e multipart uploads, marque só **Delete
      incomplete multipart uploads** → **Number of days:** `1`. (A opção de
      *expired object delete markers* fica bloqueada quando a expiração está
      marcada, e não importa aqui porque o versionamento está desligado.)
- [x] *Create rule*.

*Objetivo:* um upload que o navegador fez mas nunca confirmou some sozinho em
um dia, sem nenhum processo no backend. A AWS executa essas regras uma vez por
dia, então "1 dia" na prática é entre 24 e 48 horas.

> [!NOTE]
> A regra foi criada com o nome **`Expire Pending Uploads`**, e não
> `expire-pending-uploads`. O comportamento é o do roteiro: prefixo `pending/`,
> expiração em 1 dia, uploads multipart incompletos cancelados em 1 dia, status
> `Enabled`.

---

## Passo 2 — Criar a cache policy `avatars-24h`

Criada antes da distribuição, para já poder escolhê-la no passo 3.

*CloudFront* → menu lateral **Policies** → aba **Cache** → *Create cache policy*.

- [x] **Name:** `avatars-24h`.
- [x] **Description:** `Avatares: borda reconfere em 24h, navegador respeita o Cache-Control do objeto`.
- [x] **TTL settings:**
  - **Minimum TTL:** `0`
  - **Maximum TTL:** `86400`
  - **Default TTL:** `86400`
- [x] **Cache key settings:** **Headers** = *None*, **Query strings** = *None*,
      **Cookies** = *None*.
- [x] **Compression support:** marque **Gzip** e **Brotli**.
- [x] *Create*.

*Objetivo:* cada avatar é enviado com `Cache-Control: public, max-age=31536000,
immutable`. O **navegador** guarda por um ano, o que é seguro porque a URL muda a
cada troca de avatar. A **borda do CloudFront** fica limitada a 24 h pelo
*Maximum TTL*, então um avatar apagado deixa de ser servido em no máximo um dia,
sem precisar de invalidação.

> [!NOTE]
> Cache policy **`avatars-24h`**, id **`9d4bc7ec-d605-4d30-b985-e8cee541cffd`**.
> Todos os campos conferem com o roteiro.

---

## Passo 3 — Criar a distribuição CloudFront

*CloudFront* → **Distributions** → *Create distribution*.

### 3.1 Plano de preço (se o console perguntar)

O console pode abrir com uma etapa de escolha de plano: planos de valor fixo
(incluindo um gratuito) ou *Pay as you go*.

- [x] Escolha o plano **gratuito**, se houver, ou **Pay as you go**. Os dois
      funcionam para este volume. Antes de escolher, leia os limites do plano
      gratuito na própria tela: se ele não permitir cache policy personalizada
      ou alguma opção dos passos abaixo, fique com *Pay as you go*.

> [!NOTE]
> O assistente não mostrou a etapa de plano: abriu direto em *Get started*. A
> tela de revisão mostrou **Billing: Pay-as-you-go ($0/month)**.

### 3.2 Nome e tipo

- [x] **Distribution name** (se pedir): `auth-service-avatars`.
- [x] **Distribution type** (se pedir): *Single website or app*.
- [x] **Domain** / **Custom domain**: deixe em branco. Vamos usar o domínio
      `*.cloudfront.net` que a AWS gera.

> [!NOTE]
> No console de 2026-11 a opção se chama *Single website configuration*. O nome
> virou a tag `Name = auth-service-avatars` da distribuição.

### 3.3 Origem

- [x] **Origin type:** *Amazon S3*.
- [x] **S3 origin** / **Origin domain:** clique em *Browse S3* (ou abra a lista)
      e escolha `vanin-auth-service-storage`. O valor tem que ficar
      `vanin-auth-service-storage.s3.us-east-1.amazonaws.com`. **Não** use um
      endereço com `s3-website` no nome: esse é o endpoint de site estático e
      não funciona com acesso privado.
- [x] **Origin path:** deixe em branco.
- [x] **Origin access:** escolha **Origin access control settings
      (recommended)**. Se o console mostrar uma caixa como **Allow private S3
      bucket access to CloudFront**, marque-a.
  - Se pedir para escolher um OAC: *Create new OAC* → nome
    `auth-service-avatars-oac` → **Signing behavior:** *Sign requests
    (recommended)* → *Create*.
- [x] Se o console oferecer **Use recommended origin settings**, pode aceitar.

> [!NOTE]
> O console não ofereceu escolher o OAC: a caixa **Allow private S3 bucket access
> to CloudFront** criou um sozinho, com nome gerado.
> OAC **`E894TCW7KFKM8`** (`oac-vanin-auth-service-storage.s3.us-east-1.amazonaw-mwh7rpgvdfc`),
> assinatura `always`, protocolo `sigv4`. A mesma caixa gravou no bucket uma
> política liberando o bucket inteiro, substituída no passo 4.

### 3.4 Comportamento padrão (Default cache behavior)

Se o assistente tiver uma opção de "configurações recomendadas" e não mostrar
estes campos, aceite o padrão e ajuste no passo 3.7, depois de criar.

- [x] **Viewer protocol policy:** *Redirect HTTP to HTTPS*.
- [x] **Allowed HTTP methods:** *GET, HEAD*.
- [x] **Cache policy:** `avatars-24h` (a do passo 2, na seção *Custom*).
- [x] **Origin request policy:** nenhuma.
- [x] **Response headers policy:** **SecurityHeadersPolicy** (na seção
      *Managed*).

*Objetivo da response headers policy:* ela acrescenta, entre outros,
`X-Content-Type-Options: nosniff`. Com isso, um arquivo disfarçado de imagem
nunca é executado pelo navegador como script ou página.

> [!NOTE]
> O assistente não mostrou estes campos: a revisão dizia que aplicaria o padrão
> para S3 e que dava para personalizar depois. Os cinco valores foram ajustados
> pelo passo 3.7, depois de criar.

### 3.5 Segurança e demais opções

- [x] **Web Application Firewall (WAF):** *Do not enable security protections*.
      É custo sem retorno para imagens públicas. (Se o plano gratuito incluir
      WAF sem custo e obrigatório, aceite.)
- [x] **Price class:** *Use all edge locations (best performance)*. As classes
      mais baratas excluem a América do Sul, e os usuários estão no Brasil.
- [x] **Default root object:** em branco.
- [x] **Standard logging:** desligado.
- [x] *Create distribution*.

> [!NOTE]
> Conferido na distribuição: sem WAF, `PriceClass_All`, sem default root object,
> logging desligado, HTTP/2 e IPv6 ligados (padrão).

### 3.6 Anotar os valores

- [x] Na página da distribuição criada, anote na tabela do topo deste documento:
  - **ID** (formato `E1ABC2DEF3GHI4`)
  - **Distribution domain name** (formato `d1234abcd.cloudfront.net`)
- [x] O **Status** fica como *Deploying* por alguns minutos. Pode seguir para o
      passo 4 enquanto isso.

> [!NOTE]
> Distribuição **`EAB1577MJ1814`**, domínio **`d1xtj308k1n7yp.cloudfront.net`**,
> ARN `arn:aws:cloudfront::961868442569:distribution/EAB1577MJ1814`. Status
> `Deployed`.

### 3.7 Conferir o comportamento (se o assistente pulou campos)

- [x] Aba **Behaviors** → selecione o comportamento `Default (*)` → *Edit*.
- [x] Confira os quatro valores do passo 3.4 e corrija o que estiver diferente
      → *Save changes*.

> [!NOTE]
> Conferido no comportamento `Default (*)`: `redirect-to-https`, `GET` e `HEAD`,
> cache policy `9d4bc7ec-d605-4d30-b985-e8cee541cffd` (`avatars-24h`), sem origin
> request policy, response headers policy `67f7725c-6f97-4210-82d7-5512b31e9d03`
> (`Managed-SecurityHeadersPolicy`). A response headers policy faltou na primeira
> conferência e foi corrigida em seguida.

---

## Passo 4 — Restringir a política do bucket

Se você marcou a caixa de acesso privado no passo 3.3, o console já gravou uma
política no bucket. Ela libera **o bucket inteiro** para a distribuição, inclusive
`pending/`. A política abaixo libera **só `avatar/`**.

*S3* → `vanin-auth-service-storage` → aba **Permissions** → seção **Bucket
policy** → *Edit*.

- [x] Apague o que estiver lá e cole o JSON abaixo.
- [x] Troque `<ID_DA_DISTRIBUICAO>` pelo ID anotado no passo 3.6.
- [x] *Save changes*.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "CloudFrontReadAvatars",
      "Effect": "Allow",
      "Principal": { "Service": "cloudfront.amazonaws.com" },
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::vanin-auth-service-storage/avatar/*",
      "Condition": {
        "StringEquals": {
          "AWS:SourceArn": "arn:aws:cloudfront::961868442569:distribution/<ID_DA_DISTRIBUICAO>"
        }
      }
    }
  ]
}
```

*Objetivo:* o CloudFront só lê dentro de `avatar/`, e só esta distribuição
(a condição `AWS:SourceArn`) tem esse acesso. Uploads ainda não confirmados, em
`pending/`, nunca saem pela CDN.

> [!NOTE]
> Aplicada com `<ID_DA_DISTRIBUICAO>` = `EAB1577MJ1814`; a política gravada no
> bucket é idêntica ao JSON acima, com a condição
> `arn:aws:cloudfront::961868442569:distribution/EAB1577MJ1814`. A política
> automática do passo 3.3 (`PolicyForCloudFrontPrivateContent`, liberando
> `vanin-auth-service-storage/*`) foi removida.

---

## Passo 5 — Criar o usuário IAM do backend

O serviço não roda na AWS (o deploy é no Raspberry Pi), então não existe *IAM
role* nem *instance profile* para entregar credenciais temporárias ao processo. E
o `aws login` / SSO não serve para um servidor: a sessão expira em no máximo 12
horas e a renovação exige navegador. A decisão é uma **chave de acesso estática**
de um usuário IAM dedicado ao backend, sem acesso ao console, cuja política só
alcança este bucket e os dois prefixos de avatar.

O que o backend faz no S3 e a ação IAM que cobre cada coisa:

| O backend precisa | Ação IAM |
| --- | --- |
| Assinar o formulário de upload (presign) | nenhuma própria (ver abaixo) |
| Criar objeto: o envio do navegador e o destino da cópia | `s3:PutObject` |
| "Atualizar" objeto | `s3:PutObject`: o S3 não edita, só sobrescreve |
| Conferir (`HeadObject`) e ler a origem da cópia (`CopyObject`) | `s3:GetObject` |
| Apagar | `s3:DeleteObject` |

**Assinar não é uma chamada à AWS.** O presign é um cálculo HMAC feito localmente
com o *secret access key*; nenhuma requisição sai do backend. Quem confere é o S3,
no momento do envio: ele valida a assinatura e checa se **a chave que assinou**
tem `s3:PutObject` naquela chave de objeto. Por isso uma chave revogada ou sem
permissão gera formulários que parecem válidos e só falham quando o navegador
envia.

### 5.1 Criar a política

*IAM* → **Policies** → *Create policy*.

- [x] **Policy editor:** clique em **JSON**, apague o conteúdo e cole o JSON
      abaixo → *Next*.
- [x] **Policy name:** `auth-service-storage-avatars`.
- [x] **Description:** `Backend do auth-service: assinar uploads, conferir, copiar e apagar avatares`.
- [x] *Create policy*.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ObjectsInAvatarPrefixes",
      "Effect": "Allow",
      "Action": ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"],
      "Resource": [
        "arn:aws:s3:::vanin-auth-service-storage/pending/avatar/*",
        "arn:aws:s3:::vanin-auth-service-storage/avatar/*"
      ]
    },
    {
      "Sid": "ListOnlyThosePrefixes",
      "Effect": "Allow",
      "Action": "s3:ListBucket",
      "Resource": "arn:aws:s3:::vanin-auth-service-storage",
      "Condition": {
        "StringLike": { "s3:prefix": ["pending/avatar/*", "avatar/*"] }
      }
    }
  ]
}
```

Por que cada permissão:

- **`s3:PutObject`**: o formulário de upload que o backend assina só funciona com
  as permissões de quem assinou. É esta linha que autoriza o envio feito pelo
  navegador, e também a cópia de `pending/` para `avatar/`.
- **`s3:GetObject`**: a conferência do arquivo enviado (`HeadObject`) e a leitura
  da origem na cópia (`CopyObject`).
- **`s3:DeleteObject`**: apagar o arquivo de `pending/` depois da cópia, e o
  avatar antigo quando ele é trocado ou removido.
- **`s3:ListBucket` restrito aos prefixos**: faz um arquivo inexistente responder
  `404`. Sem ela, o S3 responde `403`, e o backend não distingue "não existe" de
  "sem permissão".

> [!NOTE]
> Política **`auth-service-storage-avatars`**, ARN
> `arn:aws:iam::961868442569:policy/auth-service-storage-avatars`, versão `v1`.
> O documento é idêntico ao JSON acima. A descrição gravada foi
> `assinar uploads, conferir, copiar e apagar avatares`.

> [!IMPORTANT]
> Decisão em aberto: manter esta política restrita aos prefixos de avatar ou
> abrir as operações de objeto para o bucket inteiro
> (`arn:aws:s3:::vanin-auth-service-storage/*`, sem condição no `ListBucket`),
> para que um tipo de conteúdo novo não exija editar o IAM. A chave e o usuário
> são os mesmos nos dois casos.

### 5.2 Criar o usuário

*IAM* → **Users** → *Create user*.

- [x] **User name:** `auth-service-backend`.
- [x] **Provide user access to the AWS Management Console:** **desmarcado**. É um
      serviço, não uma pessoa → *Next*.
- [x] **Permissions options:** *Attach policies directly*.
- [x] No campo de busca, digite `auth-service-storage-avatars` e marque a política
      criada no 5.1. Nenhuma outra → *Next*.
- [x] Confira o resumo → *Create user*.

> [!NOTE]
> Usuário **`auth-service-backend`**, ARN
> `arn:aws:iam::961868442569:user/auth-service-backend`, criado em 2026-11-15. Sem
> perfil de login no console, sem grupos, sem políticas inline, só a
> `auth-service-storage-avatars` anexada.

### 5.3 Gerar a chave de acesso

- [x] Clique no usuário `auth-service-backend` → aba **Security credentials** →
      seção **Access keys** → *Create access key*.
- [x] **Use case:** *Application running outside AWS* → *Next*.
- [x] **Description tag value:** `raspberry-pi` → *Create access key*.
- [x] Clique em **Download .csv file** e anote o **Access key ID** e o **Secret
      access key** na tabela do topo. **O segredo só aparece nesta tela.** Se
      perder, apague a chave e gere outra.
- [x] *Done*.

Essa chave vai **só** para o arquivo de configuração do serviço, que está no
`.gitignore` (`.env.development.yaml`, `.env.prod.yaml`). Nunca para o
repositório, nunca para uma mensagem. Guarde o `.csv` num lugar seguro ou apague-o
depois de configurar.

> [!NOTE]
> Uma chave **ativa**, criada em 2026-11-15, terminando em **`…3MI7`**, com a tag
> `raspberry-pi`. A chave configurada no `.env.development.yaml` termina com o
> mesmo sufixo. O id completo e o segredo não são registrados neste documento.

---

## Passo 6 — Testar antes de escrever código

Espere o status da distribuição sair de *Deploying* (passo 3.6). Use uma imagem
qualquer do seu computador, chamada aqui de `teste.png`.

### 6.1 Leitura pública pela CDN

- [x] *S3* → `vanin-auth-service-storage` → *Create folder* → `avatar` →
      entre na pasta → *Upload* → `teste.png` → *Upload*.
- [x] No navegador, abra `https://<dominio-da-distribuicao>/avatar/teste.png`.
      **Esperado:** a imagem aparece.
- [x] Abra as ferramentas de desenvolvedor do navegador (F12) → aba **Network**
      → recarregue → clique na requisição. **Esperado:** nos cabeçalhos da
      resposta aparecem `x-content-type-options: nosniff` e `x-cache` (`Miss from
      cloudfront` na primeira vez, `Hit from cloudfront` ao recarregar).

> [!NOTE]
> `https://d1xtj308k1n7yp.cloudfront.net/avatar/teste.png` respondeu `200`, com
> `content-type: image/png`, `x-content-type-options: nosniff` e `x-cache` (`Miss`
> e depois `Hit`). `http://` respondeu `301` para `https://`. A imagem de teste não
> tem `cache-control` porque foi enviada pelo console; nos avatares reais quem o
> fixa é o formulário assinado pelo backend.

### 6.2 `pending/` não sai pela CDN

- [x] Crie a pasta `pending` na raiz do bucket e suba `teste.png` nela.
- [x] Abra `https://<dominio-da-distribuicao>/pending/teste.png`.
      **Esperado:** erro `403` / `AccessDenied`.

> [!NOTE]
> Com `pending/teste.png` existindo no bucket,
> `https://d1xtj308k1n7yp.cloudfront.net/pending/teste.png` respondeu `403`. A
> primeira tentativa deu `403` sem o arquivo existir, o que não provava nada; o
> teste foi refeito depois do passo 4.

### 6.3 O bucket continua privado

- [x] Abra
      `https://vanin-auth-service-storage.s3.us-east-1.amazonaws.com/avatar/teste.png`.
      **Esperado:** erro `403` / `AccessDenied`.

> [!NOTE]
> Acesso direto ao S3 respondeu `403` para `avatar/teste.png` e para
> `pending/teste.png`.

### 6.4 Limpar

- [ ] Apague os dois `teste.png` e as pastas `avatar/` e `pending/`. O backend
      recria os caminhos quando precisar; no S3 uma "pasta" é só um prefixo.

> [!WARNING]
> Pendente. Na última conferência o bucket tinha `avatar/`, `avatar/teste.png`,
> `pending/` e `pending/teste.png`. A regra do passo 1.6 apaga os de `pending/`
> sozinha; os de `avatar/` só saem à mão.

Se algum resultado for diferente do esperado, não siga para o código: o problema
está em um dos passos acima e é mais barato achar agora.

---

## Passo 7 — O que vai para a configuração do serviço

Os valores anotados entram no arquivo de configuração do ambiente
(`.env.development.yaml`, fora do git). O formato é o de
[`env/.env.example.yaml`](../../../env/.env.example.yaml) e
[`env/.env.example`](../../../env/.env.example):

```yaml
storage:
  provider: AWS-S3
  public_base_url: https://<dominio-da-distribuicao>

  aws:
    s3_bucket_name: vanin-auth-service-storage
    region: us-east-1
    access_key_id: <Access key ID do passo 5.3>
    secret_access_key: <Secret access key do passo 5.3>
```

`public_base_url` fica fora do bloco `aws` porque é o endereço da CDN, não do
provedor: trocar um não obriga a trocar o outro (RNF-6).

Sem `provider`, ou com `provider` e algum dos outros campos vazio, o serviço sobe
e as rotas de avatar respondem `501`. No segundo caso o log de boot traz um
`ERROR` com a lista das variáveis que faltam.

O limite e a validade da autorização têm padrão no código (2 MiB, 300 s). Para
mudar, acrescente o bloco opcional:

```yaml
avatar:
  max_bytes: 2097152
  grant_ttl_seconds: 300
```

ou as variáveis `AVATAR_MAX_BYTES` e `AVATAR_GRANT_TTL_SECONDS`.

> [!NOTE]
> O `.env.development.yaml` local já tem o bloco `storage` completo:
> `provider: AWS-S3`, `public_base_url: https://d1xtj308k1n7yp.cloudfront.net`,
> bucket, região e as duas credenciais preenchidas. O arquivo está no
> `.gitignore`. O bloco `avatar` não foi acrescentado, então valem os padrões.

---

## Para estender depois

- **Novo tipo de arquivo** (um anexo, um documento): um par de prefixos novo
  (`pending/<tipo>/` e `<tipo>/`) na política do passo 5.1 e, se ele for público,
  na política do bucket do passo 4. A regra de ciclo de vida do passo 1.6 já cobre
  todo `pending/`.
- **Domínio próprio na CDN:** certificado no *AWS Certificate Manager*, emitido
  obrigatoriamente em `us-east-1`; *Alternate domain name* na distribuição; um
  registro no DNS. No serviço, muda só `public_base_url`.
- **Frontend em produção:** acrescentar a origem ao CORS do passo 1.5.
