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

## Valores a anotar durante o roteiro

Preencha conforme avança. Eles vão para a configuração do serviço no fim.

| O quê | Onde aparece | Valor |
| --- | --- | --- |
| ID da distribuição | Passo 3.6 | `E…` |
| Domínio da distribuição | Passo 3.6 | `d….cloudfront.net` |
| Access key ID do backend | Passo 5.3 | `AKIA…` |
| Secret access key do backend | Passo 5.3 | (só aparece uma vez) |

---

## Passo 0 — Conferir onde você está

- [ ] Entre no console como **`htnek-AdminAccess`**. Criar usuário e política é
      IAM, e o PowerUser não tem permissão para isso.
- [ ] Confira o canto superior direito: a conta tem que ser **9618-6844-2569**.
      Se aparecer `0540-8352-4026`, é a conta da empresa: saia e não continue.
- [ ] Região **US East (N. Virginia) `us-east-1`** no seletor do topo. O
      CloudFront e o IAM são globais e mostram "Global"; isso é normal.

---

## Passo 1 — Conferir o bucket `vanin-auth-service-storage`

*S3* → *Buckets* → clique em `vanin-auth-service-storage`.

### 1.1 Acesso público bloqueado

- [ ] Aba **Permissions** → seção **Block public access (bucket settings)**.
- [ ] Tem que estar **On**, com as quatro opções marcadas. Se não estiver:
      *Edit* → marque **Block all public access** → *Save changes* → digite
      `confirm`.

*Objetivo:* o bucket nunca é lido diretamente pela internet. Quem lê é o
CloudFront, por uma permissão própria (passo 4).

### 1.2 ACLs desligadas

- [ ] Ainda em **Permissions** → seção **Object Ownership**.
- [ ] Tem que mostrar **Bucket owner enforced** ("ACLs disabled"). Se não:
      *Edit* → **ACLs disabled (recommended)** → *Save changes*.

### 1.3 Criptografia padrão

- [ ] Aba **Properties** → seção **Default encryption**.
- [ ] Tem que mostrar **Server-side encryption with Amazon S3 managed keys
      (SSE-S3)**. É o padrão de buckets novos; se for outra coisa, *Edit* →
      **SSE-S3** → *Save changes*.

*Objetivo:* SSE-KMS exigiria permissão extra para o CloudFront ler. SSE-S3 não.

### 1.4 Versionamento desligado

- [ ] Ainda em **Properties** → seção **Bucket Versioning**: tem que estar
      **Disabled**. Deixe assim.

*Objetivo:* a versão de cada avatar já vai no nome do arquivo. Versionamento do
S3 só guardaria cópias apagadas, com custo.

### 1.5 CORS

O navegador envia o arquivo direto ao S3, então o S3 precisa aceitar requisições
vindas do endereço do frontend.

- [ ] Aba **Permissions** → seção **Cross-origin resource sharing (CORS)** →
      *Edit*.
- [ ] Cole o JSON abaixo → *Save changes*.

```json
[
  {
    "AllowedOrigins": ["http://localhost:3000"],
    "AllowedMethods": ["POST"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3000
  }
]
```

A porta `3000` é a de `frontend/vite.config.ts`. Quando o frontend tiver um
endereço de produção, volte aqui e acrescente-o à lista `AllowedOrigins`, por
exemplo `"https://app.seudominio.com"`. `POST` é o único método: o navegador
nunca lê nem apaga direto no bucket.

### 1.6 Regra de ciclo de vida para `pending/`

- [ ] Aba **Management** → seção **Lifecycle rules** → *Create lifecycle rule*.
- [ ] **Lifecycle rule name:** `expire-pending-uploads`.
- [ ] **Choose a rule scope:** *Limit the scope of this rule using one or more
      filters*.
- [ ] **Prefix:** `pending/` (com a barra no fim).
- [ ] Em **Lifecycle rule actions**, marque:
  - [ ] **Expire current versions of objects**
  - [ ] **Delete expired object delete markers or incomplete multipart uploads**
- [ ] Em **Expire current versions of objects** → **Days after object creation:**
      `1`.
- [ ] Na seção de delete markers e multipart uploads, marque só **Delete
      incomplete multipart uploads** → **Number of days:** `1`. (A opção de
      *expired object delete markers* fica bloqueada quando a expiração está
      marcada, e não importa aqui porque o versionamento está desligado.)
- [ ] *Create rule*.

*Objetivo:* um upload que o navegador fez mas nunca confirmou some sozinho em
um dia, sem nenhum processo no backend. A AWS executa essas regras uma vez por
dia, então "1 dia" na prática é entre 24 e 48 horas.

---

## Passo 2 — Criar a cache policy `avatars-24h`

Criada antes da distribuição, para já poder escolhê-la no passo 3.

*CloudFront* → menu lateral **Policies** → aba **Cache** → *Create cache policy*.

- [ ] **Name:** `avatars-24h`.
- [ ] **Description:** `Avatares: borda reconfere em 24h, navegador respeita o Cache-Control do objeto`.
- [ ] **TTL settings:**
  - **Minimum TTL:** `0`
  - **Maximum TTL:** `86400`
  - **Default TTL:** `86400`
- [ ] **Cache key settings:** **Headers** = *None*, **Query strings** = *None*,
      **Cookies** = *None*.
- [ ] **Compression support:** marque **Gzip** e **Brotli**.
- [ ] *Create*.

*Objetivo:* cada avatar é enviado com `Cache-Control: public, max-age=31536000,
immutable`. O **navegador** guarda por um ano, o que é seguro porque a URL muda a
cada troca de avatar. A **borda do CloudFront** fica limitada a 24 h pelo
*Maximum TTL*, então um avatar apagado deixa de ser servido em no máximo um dia,
sem precisar de invalidação.

---

## Passo 3 — Criar a distribuição CloudFront

*CloudFront* → **Distributions** → *Create distribution*.

### 3.1 Plano de preço (se o console perguntar)

O console pode abrir com uma etapa de escolha de plano: planos de valor fixo
(incluindo um gratuito) ou *Pay as you go*.

- [ ] Escolha o plano **gratuito**, se houver, ou **Pay as you go**. Os dois
      funcionam para este volume. Antes de escolher, leia os limites do plano
      gratuito na própria tela: se ele não permitir cache policy personalizada
      ou alguma opção dos passos abaixo, fique com *Pay as you go*.

### 3.2 Nome e tipo

- [ ] **Distribution name** (se pedir): `auth-service-avatars`.
- [ ] **Distribution type** (se pedir): *Single website or app*.
- [ ] **Domain** / **Custom domain**: deixe em branco. Vamos usar o domínio
      `*.cloudfront.net` que a AWS gera.

### 3.3 Origem

- [ ] **Origin type:** *Amazon S3*.
- [ ] **S3 origin** / **Origin domain:** clique em *Browse S3* (ou abra a lista)
      e escolha `vanin-auth-service-storage`. O valor tem que ficar
      `vanin-auth-service-storage.s3.us-east-1.amazonaws.com`. **Não** use um
      endereço com `s3-website` no nome: esse é o endpoint de site estático e
      não funciona com acesso privado.
- [ ] **Origin path:** deixe em branco.
- [ ] **Origin access:** escolha **Origin access control settings
      (recommended)**. Se o console mostrar uma caixa como **Allow private S3
      bucket access to CloudFront**, marque-a.
  - Se pedir para escolher um OAC: *Create new OAC* → nome
    `auth-service-avatars-oac` → **Signing behavior:** *Sign requests
    (recommended)* → *Create*.
- [ ] Se o console oferecer **Use recommended origin settings**, pode aceitar.

### 3.4 Comportamento padrão (Default cache behavior)

Se o assistente tiver uma opção de "configurações recomendadas" e não mostrar
estes campos, aceite o padrão e ajuste no passo 3.7, depois de criar.

- [ ] **Viewer protocol policy:** *Redirect HTTP to HTTPS*.
- [ ] **Allowed HTTP methods:** *GET, HEAD*.
- [ ] **Cache policy:** `avatars-24h` (a do passo 2, na seção *Custom*).
- [ ] **Origin request policy:** nenhuma.
- [ ] **Response headers policy:** **SecurityHeadersPolicy** (na seção
      *Managed*).

*Objetivo da response headers policy:* ela acrescenta, entre outros,
`X-Content-Type-Options: nosniff`. Com isso, um arquivo disfarçado de imagem
nunca é executado pelo navegador como script ou página.

### 3.5 Segurança e demais opções

- [ ] **Web Application Firewall (WAF):** *Do not enable security protections*.
      É custo sem retorno para imagens públicas. (Se o plano gratuito incluir
      WAF sem custo e obrigatório, aceite.)
- [ ] **Price class:** *Use all edge locations (best performance)*. As classes
      mais baratas excluem a América do Sul, e os usuários estão no Brasil.
- [ ] **Default root object:** em branco.
- [ ] **Standard logging:** desligado.
- [ ] *Create distribution*.

### 3.6 Anotar os valores

- [ ] Na página da distribuição criada, anote na tabela do topo deste documento:
  - **ID** (formato `E1ABC2DEF3GHI4`)
  - **Distribution domain name** (formato `d1234abcd.cloudfront.net`)
- [ ] O **Status** fica como *Deploying* por alguns minutos. Pode seguir para o
      passo 4 enquanto isso.

### 3.7 Conferir o comportamento (se o assistente pulou campos)

- [ ] Aba **Behaviors** → selecione o comportamento `Default (*)` → *Edit*.
- [ ] Confira os quatro valores do passo 3.4 e corrija o que estiver diferente
      → *Save changes*.

---

## Passo 4 — Restringir a política do bucket

Se você marcou a caixa de acesso privado no passo 3.3, o console já gravou uma
política no bucket. Ela libera **o bucket inteiro** para a distribuição, inclusive
`pending/`. A política abaixo libera **só `avatar/`**.

*S3* → `vanin-auth-service-storage` → aba **Permissions** → seção **Bucket
policy** → *Edit*.

- [ ] Apague o que estiver lá e cole o JSON abaixo.
- [ ] Troque `<ID_DA_DISTRIBUICAO>` pelo ID anotado no passo 3.6.
- [ ] *Save changes*.

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

---

## Passo 5 — Criar o usuário IAM do backend

O `aws login` da sua máquina não serve para um servidor: a sessão dura no máximo
12 horas e exige navegador. O Raspberry Pi usa um usuário IAM próprio, sem acesso
ao console, com uma chave de acesso limitada a este bucket e a estes dois
prefixos.

### 5.1 Criar a política

*IAM* → **Policies** → *Create policy*.

- [ ] **Policy editor:** clique em **JSON**, apague o conteúdo e cole o JSON
      abaixo → *Next*.
- [ ] **Policy name:** `auth-service-storage-avatars`.
- [ ] **Description:** `Backend do auth-service: assinar uploads, conferir, copiar e apagar avatares`.
- [ ] *Create policy*.

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

### 5.2 Criar o usuário

*IAM* → **Users** → *Create user*.

- [ ] **User name:** `auth-service-backend`.
- [ ] **Provide user access to the AWS Management Console:** **desmarcado**. É um
      serviço, não uma pessoa → *Next*.
- [ ] **Permissions options:** *Attach policies directly*.
- [ ] No campo de busca, digite `auth-service-storage-avatars` e marque a política
      criada no 5.1. Nenhuma outra → *Next*.
- [ ] Confira o resumo → *Create user*.

### 5.3 Gerar a chave de acesso

- [ ] Clique no usuário `auth-service-backend` → aba **Security credentials** →
      seção **Access keys** → *Create access key*.
- [ ] **Use case:** *Application running outside AWS* → *Next*.
- [ ] **Description tag value:** `raspberry-pi` → *Create access key*.
- [ ] Clique em **Download .csv file** e anote o **Access key ID** e o **Secret
      access key** na tabela do topo. **O segredo só aparece nesta tela.** Se
      perder, apague a chave e gere outra.
- [ ] *Done*.

Essa chave vai **só** para o arquivo de configuração do serviço, que está no
`.gitignore` (`.env.development.yaml`, `.env.prod.yaml`). Nunca para o
repositório, nunca para uma mensagem. Guarde o `.csv` num lugar seguro ou apague-o
depois de configurar.

---

## Passo 6 — Testar antes de escrever código

Espere o status da distribuição sair de *Deploying* (passo 3.6). Use uma imagem
qualquer do seu computador, chamada aqui de `teste.png`.

### 6.1 Leitura pública pela CDN

- [ ] *S3* → `vanin-auth-service-storage` → *Create folder* → `avatar` →
      entre na pasta → *Upload* → `teste.png` → *Upload*.
- [ ] No navegador, abra `https://<dominio-da-distribuicao>/avatar/teste.png`.
      **Esperado:** a imagem aparece.
- [ ] Abra as ferramentas de desenvolvedor do navegador (F12) → aba **Network**
      → recarregue → clique na requisição. **Esperado:** nos cabeçalhos da
      resposta aparecem `x-content-type-options: nosniff` e `x-cache` (`Miss from
      cloudfront` na primeira vez, `Hit from cloudfront` ao recarregar).

### 6.2 `pending/` não sai pela CDN

- [ ] Crie a pasta `pending` na raiz do bucket e suba `teste.png` nela.
- [ ] Abra `https://<dominio-da-distribuicao>/pending/teste.png`.
      **Esperado:** erro `403` / `AccessDenied`.

### 6.3 O bucket continua privado

- [ ] Abra
      `https://vanin-auth-service-storage.s3.us-east-1.amazonaws.com/avatar/teste.png`.
      **Esperado:** erro `403` / `AccessDenied`.

### 6.4 Limpar

- [ ] Apague os dois `teste.png` e as pastas `avatar/` e `pending/`. O backend
      recria os caminhos quando precisar; no S3 uma "pasta" é só um prefixo.

Se algum resultado for diferente do esperado, não siga para o código: o problema
está em um dos passos acima e é mais barato achar agora.

---

## Passo 7 — O que vai para a configuração do serviço

Os valores anotados entram no arquivo de configuração do ambiente
(`.env.development.yaml`, fora do git). O formato exato das chaves é decisão do
[plano de implementação](implementation-plan.md); o esboço é:

```yaml
storage:
  provider: s3
  bucket: vanin-auth-service-storage
  region: us-east-1
  access_key_id: <Access key ID do passo 5.3>
  secret_access_key: <Secret access key do passo 5.3>
  public_base_url: https://<dominio-da-distribuicao>

avatar:
  max_bytes: 2097152
  grant_ttl_seconds: 300
```

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
