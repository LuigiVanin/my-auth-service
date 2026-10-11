# Infraestrutura na AWS — avatares

Os passos abaixo são executados pelo console, na conta pessoal `961868442569`,
região `us-east-1`. Os JSONs estão aqui para que a configuração seja reproduzível
e revisável; nenhum deles é lido pelo código. A ordem importa: a política do bucket
cita o id da distribuição, e a distribuição cita o bucket.

Nada aqui é executado pelo backend nem por quem escreve código: é configuração
feita uma vez, com o perfil administrador (`htnek-AdminAccess`), porque criar
usuário e política é IAM.

```
navegador ──(1) POST /…/avatar/upload ──▶ backend ── assina ──▶ { url, fields, key }
navegador ──(2) POST multipart ─────────▶ S3  pending/avatar/…   (recusa > 2 MiB, tipo errado)
navegador ──(3) PUT /…/avatar { key } ──▶ backend ── Head, Copy, Delete ──▶ S3 avatar/…
qualquer um ─(4) GET ───────────────────▶ CloudFront ──(OAC)──▶ S3 avatar/*
```

## 1. Bucket `vanin-auth-service-storage`

Conferir, em *Properties* e *Permissions*:

| Configuração | Valor | Por quê |
| --- | --- | --- |
| Block all public access | **ligado**, as quatro caixas | A leitura pública é do CloudFront, não do bucket |
| Object Ownership | *Bucket owner enforced* (ACLs desligadas) | O formulário assinado não carrega ACL; nada depende dela |
| Default encryption | SSE-S3 (`AES256`) | Padrão; sem custo; o CloudFront lê sem configuração extra |
| Versioning | desligado | A versão vive no nome da chave (RC-4), não no S3 |

### CORS (*Permissions* → *Cross-origin resource sharing*)

O navegador envia o arquivo direto ao S3, então o S3 precisa aceitar a origem do
frontend. `POST` é o único método: o cliente nunca lê nem apaga direto.

```json
[
  {
    "AllowedOrigins": ["http://localhost:3000", "https://<origem-do-frontend>"],
    "AllowedMethods": ["POST"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3000
  }
]
```

Trocar `<origem-do-frontend>` pela origem real quando houver. A porta `3000` é a
de `frontend/vite.config.ts`.

### Ciclo de vida (*Management* → *Lifecycle rules* → *Create rule*)

| Campo | Valor |
| --- | --- |
| Nome | `expire-pending-uploads` |
| Escopo | *Limit the scope using filters*, prefixo `pending/` |
| Ação | *Expire current versions of objects*: **1** dia |
| Ação | *Delete expired object delete markers or incomplete multipart uploads*: **1** dia |

É o que cumpre o RNF-5: um upload autorizado e nunca confirmado some sozinho.

## 2. CloudFront

*CloudFront* → *Create distribution*:

| Campo | Valor | Por quê |
| --- | --- | --- |
| Origin domain | `vanin-auth-service-storage.s3.us-east-1.amazonaws.com` | O endpoint REST, **não** o de website estático: o OAC só funciona com este |
| Origin access | *Origin access control settings* → *Create new OAC*, assinatura *Sign requests* | Deixa o bucket privado e autoriza só esta distribuição |
| Viewer protocol policy | *Redirect HTTP to HTTPS* | |
| Allowed HTTP methods | `GET, HEAD` | Nada é escrito pela CDN |
| Cache policy | **Custom**, ver abaixo | O `CachingOptimized` padrão tem TTL máximo de um ano; o RC-12 quer 24 h na borda |
| Response headers policy | *SecurityHeadersPolicy* (gerenciada) | Entrega `X-Content-Type-Options: nosniff` (RC-11) |
| Price class | *Use all edge locations* | As classes mais baratas excluem a América do Sul |
| WAF | desligado | Custo sem retorno para imagens públicas |
| Default root object | vazio | |

### Cache policy `avatars-24h`

*Policies* → *Cache* → *Create cache policy*:

| Campo | Valor |
| --- | --- |
| Minimum TTL | `0` |
| Default TTL | `86400` |
| Maximum TTL | `86400` |
| Headers, cookies, query strings | nenhum incluído na chave de cache |
| Compression | Gzip e Brotli ligados (inócuo para imagem, útil se outro prefixo vier) |

O objeto chega com `Cache-Control: public, max-age=31536000, immutable`; o
**navegador** respeita o ano inteiro, e a **borda** reconfere no S3 a cada 24 h por
causa do *Maximum TTL*. É isso que faz um avatar apagado sumir da CDN em no máximo
um dia sem invalidação.

Ao salvar a distribuição, o console mostra a política de bucket a aplicar. Ela é
mais larga que a necessária (libera o bucket inteiro): usar a do passo 3.

## 3. Política do bucket

*Permissions* → *Bucket policy*. Trocar `<ID_DA_DISTRIBUICAO>` pelo id gerado no
passo 2 (formato `E1ABC2DEF3GHI4`). Só `avatar/*` é liberado: `pending/` nunca
sai pela CDN.

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

## 4. Usuário IAM do backend

O `aws login` da máquina de desenvolvimento não serve para um servidor: sessão de
no máximo 12 h, com navegador. O Raspberry Pi usa um usuário IAM dedicado com uma
chave de acesso de longa duração, limitada a este bucket e a estes dois prefixos.

*IAM* → *Users* → *Create user*:

| Campo | Valor |
| --- | --- |
| Nome | `auth-service-backend` |
| Acesso ao console | **desmarcado** — é um serviço, não uma pessoa |
| Permissões | *Attach policies directly* → *Create policy* → JSON abaixo, nome `auth-service-storage-avatars` |

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

Por que cada ação:

- `s3:PutObject` — o formulário assinado só funciona com as permissões de quem o
  assinou; é esta linha que autoriza o upload do navegador, e também a cópia para
  `avatar/`.
- `s3:GetObject` — a conferência (`HeadObject`) e a origem da cópia (`CopyObject`
  lê a origem com a permissão do chamador).
- `s3:DeleteObject` — apagar o `pending/` após a cópia e o avatar antigo.
- `s3:ListBucket` restrito por prefixo — é o que faz um objeto inexistente
  responder `404` em vez de `403`; sem ela o backend não distingue "não existe" de
  "sem permissão".

Depois de criar: *Security credentials* → *Create access key* → *Application
running outside AWS*. Guardar o par **uma vez**; ele vai para a configuração do
serviço, com o mesmo tratamento da chave do Resend. Nunca para o repositório.

Um novo tipo de arquivo no futuro (RNF-9) acrescenta um par de prefixos aqui e na
política do bucket — nada mais.

## 5. Teste manual antes de escrever código

1. Subir um `teste.png` pelo console em `avatar/teste.png`.
2. Abrir `https://<dominio>.cloudfront.net/avatar/teste.png` — tem que mostrar a
   imagem, com `x-content-type-options: nosniff` e `cache-control` vindos nas
   respostas.
3. Subir outro em `pending/teste.png` e abrir
   `https://<dominio>.cloudfront.net/pending/teste.png` — tem que responder
   `403`.
4. Abrir `https://vanin-auth-service-storage.s3.us-east-1.amazonaws.com/avatar/teste.png`
   direto — tem que responder `403`: o bucket continua privado.
5. Apagar os dois objetos.

## O que vai para a configuração do serviço

```yaml
storage:
  provider: s3
  bucket: vanin-auth-service-storage
  region: us-east-1
  access_key_id: <chave do usuário auth-service-backend>
  secret_access_key: <segredo>
  public_base_url: https://<dominio>.cloudfront.net

avatar:
  max_bytes: 2097152
  grant_ttl_seconds: 300
```

O nome das chaves é decisão do plano de implementação; os valores são os deste
documento.
