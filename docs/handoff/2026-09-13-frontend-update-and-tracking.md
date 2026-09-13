# Handoff para o frontend: rotas de update e tracking

Duas features entraram na `main` pelo PR #9. Este documento é o contrato: o que
existe, o que enviar, o que volta, e as armadilhas que fazem uma integração parecer
funcionar e estar errada.

Specs as-built, com o porquê de cada decisão:
[2026-09-09-update-routes.md](../features/2026-09-09-update-routes/spec.md) e
[2026-09-09-tracking.md](../features/2026-09-09-tracking/spec.md).

---

## 1. As três semânticas que precisam estar certas

### 1.1 Omitir não é o mesmo que enviar vazio

Todo campo de corpo de update é opcional, e **o que não está no JSON não é
escrito**. Isso não é o mesmo que enviar o valor zero:

| O que você envia | O que acontece |
| --- | --- |
| campo ausente | a coluna não é tocada |
| `"name": ""` | escreve string vazia (e é recusado onde há `min=1`) |
| `"private": false` | escreve `false` |
| `"token_expiration_time": 0` | recusado (`gt=0`), mas em campos sem essa regra escreveria `0` |
| `"name": null` | **igual a ausente** — não limpa nada |

O último é o que mais pega. `null` num campo escalar não limpa a coluna: o backend
não consegue distinguir `null` de ausente, então trata como "não mexa". Se o
formulário serializa campos vazios como `null`, **remova-os do payload** antes de
enviar, ou eles viram no-op silencioso em vez do erro que você esperava.

A recomendação prática: monte o corpo só com os campos que o usuário realmente
mexeu (dirty fields). Enviar o formulário inteiro funciona, mas reescreve tudo com
o que estava na tela, inclusive o que outra aba mudou no meio.

### 1.2 `metadata` é merge, não substituição

Toda coluna `metadata` é mesclada com o que já está guardado
([RFC 7386](https://www.rfc-editor.org/rfc/rfc7386)), não sobrescrita:

```jsonc
// guardado
{ "plano": "pro", "tema": "escuro", "beta": true }

// enviado
{ "metadata": { "tema": "claro", "idioma": "pt", "beta": null } }

// resultado
{ "plano": "pro", "tema": "claro", "idioma": "pt" }
```

- as chaves que você **não** nomeia continuam lá;
- as que você nomeia vencem;
- **`null` numa chave apaga aquela chave** — é a única forma de apagar;
- objeto aninhado desce um nível e mescla recursivamente;
- **array é valor, não documento**: `{"lista": [3]}` substitui a lista inteira, não
  concatena.

Duas coisas que o endpoint não faz:

- **Não dá para limpar o objeto inteiro.** `"metadata": null` quer dizer "não
  mexa". Para esvaziar, apague chave por chave.
- **`metadata` tem que ser objeto.** Enviar `5`, `"texto"` ou `[1,2]` responde
  `400` com `data.field = "metadata"`.

### 1.3 Corpo vazio é `200`, não erro

`PUT` com `{}` responde `200` com a linha inalterada. Não é erro, e não é 404.

---

## 2. As rotas

Todas exigem os cabeçalhos de sempre — `X-Public-Key`, `X-Pool-Key`,
`X-Secret-Key` quando o app é privado, e `Authorization` — e passam pelo
`PermissionsGuard`, então cada uma precisa do seu grant no profile de quem chama.

| Método | Rota | Grant | Escopo |
| --- | --- | --- | --- |
| `PUT` | `/core/apps/{id}` | `as::apps::UPDATE` | app da organização atual |
| `PUT` | `/core/users_pool/{id}` | `as::users_pool::UPDATE` | pool da organização atual |
| `PUT` | `/core/users/{id}` | `as::users::UPDATE` | usuário de uma pool que a organização possui |
| `PUT` | `/core/users/me` | `as::users::me::UPDATE` | o próprio usuário autenticado |
| `PUT` | `/core/organizations/{id}` | `as::organizations::UPDATE` | tem que ser a organização atual |
| `PUT` | `/core/profiles/{id}` | `as::profiles::UPDATE` | profile visível para a organização atual |

**`/core/users/me` e `/core/users/{id}` são rotas diferentes de propósito**, com
grants diferentes. Um usuário comum tem `as::users::me::UPDATE` (está no
`LOGIN_PROFILE`) e edita a si mesmo. Quem administra a pool tem
`as::users::UPDATE` e edita qualquer usuário dela. **Chamar `/core/users/{id}` com
o próprio id não funciona como atalho** — continua exigindo o grant de
administração e a checagem de pool.

### 2.1 `PUT /core/apps/{id}`

```jsonc
{
  "name": "Portal",
  "login_types": ["WITH_PASSWORD", "WITH_OTP"],  // WITH_LOGIN | WITH_OTP | WITH_PASSWORD
  "token_type": "JWT",                            // JWT | FAST_JWT | SESSION_UUID
  "token_expiration_time": 3600,                  // > 0
  "refresh_token_expiration_time": 1296000,       // > 0
  "private": false,
  "verify_email": true,
  "enabled_2fa": false,
  "metadata": { }
}
```

Resposta `200`: a entidade `App`, com `secret_key` sempre vazio.

Não são editáveis: `users_pool_id` e `parent_app_id` (os usuários moram na pool do
app, mover deixaria todos para trás), `public_key` e `secret_key`.

**Dois campos mordem quem chama, avise na UI:**

- trocar `token_type` **invalida o formato de todo token já emitido** por aquele
  app — as sessões existentes param de ser aceitas;
- `private: true` passa a exigir `X-Secret-Key` naquele mesmo app, então o próprio
  cliente que fez a chamada pode se trancar para fora.

### 2.2 `PUT /core/users_pool/{id}`

```jsonc
{
  "name": "Clientes",
  "description": "",
  "default_profile_id": "uuid",
  "metadata": { }
}
```

Resposta `200`: a entidade `UsersPool`, incluindo `users_count` e `tracking`.

`default_profile_id` é o **teto de permissão de toda organização que nascer na
pool**. Ele passa pelo mesmo clamp da criação: se o profile pedido conceder mais do
que quem chama possui, responde `403 PERMISSION_DENIED` com a chave do profile no
`detail` — e **sem `data`**, ao contrário do `PUT /core/profiles/{id}`, que manda
`data.requested` e `data.granted`. Aqui você só tem a mensagem; para mostrar a
diferença, compare contra `GET /core/profiles` no cliente.

`users_count` e `tracking` não são escrevíveis por payload — se você mandar, são
ignorados.

### 2.3 `PUT /core/users/{id}` — administração

```jsonc
{
  "name": "Ana",
  "email": "ana@example.com",
  "phone": "+5511999999999",
  "verify_email": true,
  "two_factor_enabled": false,
  "metadata": { }
}
```

Resposta `200`: a entidade `User`, **sem** `tracking`.

`email` **não tem fluxo de verificação hoje** — esta rota é a única forma de trocar
email, e é rota de administração. Colisão dentro da mesma pool responde
`409 USER_ALREADY_EXISTS`. O email é normalizado para minúsculas antes de gravar,
então `ANA@Example.com` e `ana@example.com` são o mesmo email.

Senha nunca vem por este payload — continua no fluxo de esqueci-a-senha.

### 2.4 `PUT /core/users/me` — o próprio usuário

```jsonc
{
  "name": "Ana",
  "phone": "+5511999999999",
  "metadata": { }
}
```

Mais estreita de propósito: `email`, `verify_email` e `two_factor_enabled` decidem
como o usuário se autentica, então movê-los é ato de administração e não edição de
perfil.

### 2.5 `PUT /core/organizations/{id}`

```jsonc
{ "name": "Acme", "description": "", "metadata": { } }
```

**`id` tem que ser a organização atual do usuário.** Qualquer outra responde
`403 PERMISSION_DENIED`, porque as permissões que autorizaram a requisição são as
que ele tem *ali*. Para editar outra, troque de organização antes com
`PUT /core/organizations/switch`.

`profile_id` (o teto da organização) e o dono não são editáveis aqui.

**Cuidado com a ordem das rotas no cliente também:** `/core/organizations/switch` é
rota literal e `/core/organizations/{id}` é paramétrica. No backend a ordem de
registro já está travada por teste, mas se o seu cliente monta a URL
concatenando, garanta que `switch` nunca vire um `{id}`.

### 2.6 `PUT /core/profiles/{id}`

```jsonc
{
  "name": "Editor",
  "permissions": { "grants": ["as::users::READ"] },
  "metadata": { }
}
```

Só a metade `grants` do documento é tocada; um `api` escrito à mão na linha é
preservado. Recusado com `403` num profile global, no `Admin` da organização, no
profile com que quem chama participa, e quando os grants pedidos excedem o que ele
possui.

`GET /core/grants` lista todos os grants aceitos — é a fonte para montar um editor
de profile, e ela **mudou**: ganhou `as::users::UPDATE`, `as::users::me::UPDATE`,
`as::users_pool::UPDATE` e `as::organizations::UPDATE`.

---

## 3. Tracking

Duas colunas novas, mantidas pelo backend e **nunca escrevíveis por payload**.

### 3.1 Pool: cadastros por mês e por app

Em `users_pool.tracking`, no `GET /core/users_pool/{id}`:

```jsonc
{
  "signup": {
    "periods": {
      "2026-09": { "total": 30, "apps": { "<app-uuid>": 12, "<outro-app>": 18 } },
      "2026-08": { "total": 5,  "apps": { "<app-uuid>": 5 } }
    }
  }
}
```

- chaveado por `YYYY-MM` em **UTC**, no máximo **10 meses** — o mais antigo sai
  quando entra um novo;
- `total` é a soma dos apps daquele mês;
- são contadores: nenhum cadastro individual é guardado;
- um mês sem cadastro simplesmente **não aparece** — ao montar um gráfico,
  preencha os meses faltantes com zero você mesmo, não assuma dez chaves;
- `periods` pode vir `{}` numa pool sem cadastro, e `tracking` pode vir **ausente**.

**Não vem na listagem.** `GET /core/users_pool` omite o campo linha a linha, de
propósito. Para o gráfico de uma pool, chame o `GET` dela.

`users_pool.users_count` é o contador geral, monotônico — conta cadastros feitos,
não linhas existentes, então **não vai bater com um `COUNT(*)`** depois da primeira
exclusão de usuário. Ele vem tanto na listagem quanto no `GET` individual.

### 3.2 Usuário: os últimos logins

Em `users.tracking`, e este é o ponto que mais afeta o frontend:

```jsonc
{
  "login": {
    "events": [
      {
        "app_id": "<uuid>",
        "at": "2026-09-13T12:00:00Z",
        "ip": "203.0.113.7",
        "user_agent": "Mozilla/5.0 …",
        "login_type": "WITH_PASSWORD"
      }
    ]
  }
}
```

Mais recente primeiro, no máximo 10. `ip`, `user_agent` e `login_type` são
omitidos quando vazios.

**Ele só existe em duas respostas: `GET /core/users/me` e `GET /core/users/{id}`.**

Não vem em `POST /auth/login`, `POST /auth/register`, `POST /auth/refresh`, nem em
`GET /core/users` — a coluna carrega histórico de ip, e essas respostas embutem a
entidade do usuário em lugares onde isso não deve aparecer. Se a tela de "sessões
recentes" está no fluxo pós-login, ela precisa de uma chamada a `/core/users/me`;
o objeto de usuário que veio no login não traz o campo.

**Aviso de conteúdo:** `GET /core/users/{id}` devolve o histórico de ip de outro
usuário para quem administra a pool. Trate como dado pessoal na UI.

Um bug conhecido, anotado no spec: login com OTP grava `login_type` como
`"WITH_PASSWORD"`. Não confie nesse campo para distinguir o método até a correção.

### 3.3 O que dispara o tracking

| Evento | Efeito |
| --- | --- |
| `POST /auth/register` | `+1` no mês/app da pool **e** um evento de login no usuário |
| `POST /auth/login` (senha ou OTP) | um evento de login no usuário |
| `POST /auth/refresh` | **nada** — renovação não é login |

O tracking é escrito **fora da requisição**, numa goroutine, para não somar
latência. Na prática: pode haver um atraso de milissegundos entre o `200` do
cadastro e o contador refletir. **Não escreva um teste de ponta a ponta que cadastra
e lê o contador na linha seguinte sem um retry** — e, se a UI mostra o número logo
depois de uma ação, ela precisa reconsultar, não deduzir.

---

## 4. Armadilhas que dão trabalho se descobertas tarde

### BREAKING: nomes de campo padronizados para snake_case

**Isto exige mudança no frontend antes de subir junto.** A API tinha campos que se
chamavam de um jeito na ida e de outro na volta: você mandava `verify_email` no
`PUT` e lia `verifyEmail` na resposta. Pior, mandar o nome que você acabou de ler
era **ignorado em silêncio** — chegava como "não enviado", sem 400, sem aviso, com
a tela mostrando sucesso.

O backend foi padronizado: **agora tudo é snake_case, nas duas direções.**

| Entidade | Antes (na resposta) | Agora |
| --- | --- | --- |
| `User` | `verifyEmail` | `verify_email` |
| `User` | `twoFactorEnabled` | `two_factor_enabled` |
| `User` | `createdAt` / `updatedAt` | `created_at` / `updated_at` |
| `Profile` | `createdAt` / `updatedAt` | `created_at` / `updated_at` |
| `App` | `VerifiedEmailDate` | `verified_email_date` |

O último era um campo sem tag nenhuma, que saía em PascalCase. Nada consumia, mas
entrou na mesma passada.

Depois da mudança, o corpo do usuário é simétrico:

```jsonc
// o frontend manda
{ "name": "Ana", "verify_email": true, "two_factor_enabled": true }

// o backend devolve
{ "id": 1, "name": "Ana", "email": "…", "phone": "…",
  "verify_email": true, "two_factor_enabled": true,
  "metadata": {}, "created_at": "…", "updated_at": "…",
  "current_organization_id": "…" }
```

#### Etapas da correção no frontend

Onde os nomes legados estão hoje, levantado no repositório:

**Tipos — trocar a declaração:**

| Arquivo | Linhas | Campos |
| --- | --- | --- |
| `src/types/users.ts` | 19-20, 22-23 | `verifyEmail`, `twoFactorEnabled`, `createdAt`, `updatedAt` |
| `src/types/profiles.ts` | 38-39 | `createdAt`, `updatedAt` |
| `src/types/auth.ts` | 40-41 | `createdAt`, `updatedAt` |

**Leituras — trocar o acesso:**

| Arquivo | Linhas | O que lê |
| --- | --- | --- |
| `src/components/console/PoolUsersSection.tsx` | 104, 107, 109 | `user.verifyEmail`, `user.twoFactorEnabled`, `user.createdAt` |
| `src/components/console/UserDetailModal.tsx` | 45, 53, 63, 64 | `data.verifyEmail`, `data.twoFactorEnabled`, `data.createdAt`, `data.updatedAt` |
| `src/pages/console/ProfileDetailPage.tsx` | 156, 159 | `profile.createdAt`, `profile.updatedAt` |

**Documentação do frontend — atualizar as três:** `docs/core-api.md`,
`docs/handoff-core-console.md` e
`docs/features/06-09-2026-core-console/requirements.md`, que descrevem a convenção
antiga e viram fonte de erro para quem ler depois.

Ordem sugerida: comece pelos três arquivos de tipo. Se o projeto tem TypeScript com
`strict`, o compilador aponta cada leitura quebrada — aí as três telas são
correção guiada, não caça a `grep`. Depois a documentação.

**Verificação:** carregue a listagem de usuários de uma pool. Se o badge de "email
não verificado" e o de 2FA sumirem ou aparecerem errado, ou se a data vier
`Invalid Date`, sobrou alguma leitura no nome antigo.

**Não há compatibilidade retroativa:** os nomes antigos deixaram de existir na
resposta. Backend e frontend precisam subir na mesma janela.

**Para o resto da API nada muda** — `app`, `users_pool` e `organizations` já eram
snake_case nas duas direções.

**A regra que fica:** ainda assim, use tipos separados para o payload de update e
para a resposta. Não é mais por causa do nome, e sim porque o payload é todo
opcional e a resposta não é — reenviar o objeto lido continua sendo a forma mais
fácil de sobrescrever o que outra aba mudou.

**Erro de validação é `400`, não `422`**, e o nome do campo vem em PascalCase do Go:

```jsonc
{
  "type": "…", "title": "Bad Request", "status": 400,
  "code": "BAD_REQUEST",
  "detail": "…",
  "instance": "/core/apps/…",
  "data": {
    "fields": [
      { "field": "TokenExpirationTime", "tag": "gt", "param": "0", "value": 0 }
    ]
  }
}
```

`field` é `TokenExpirationTime`, **não** `token_expiration_time`. Para destacar o
input errado no formulário, você precisa de um mapa de nome Go → nome do campo. O
`422 UNPROCESSABLE_ENTITY` existe, mas só para corpo que nem chega a ser lido.

**"Não é sua" responde `404`, não `403`.** Um app, pool ou usuário de outra
organização é indistinguível de inexistente — é proposital, para o id não virar
oráculo. Não mostre "sem permissão" nesse caso; mostre "não encontrado".

**`403` tem dois códigos com significados diferentes:**

- `PERMISSION_DENIED` — o profile não alcança a rota, ou o que foi pedido excede o
  que quem chama possui. A saída é permissão.
- `NOT_A_PARTICIPANT` — o escopo é o problema. A saída é **trocar de organização**,
  não pedir permissão. Vale um tratamento próprio na UI.

---

## 5. Checklist de implementação

- [ ] **Renomear os campos legados** — três arquivos de tipo, três telas e três
      documentos, listados na seção 4. É breaking: sobe junto com o backend.
- [ ] Tipos separados para payload de update (tudo opcional) e para a resposta.
- [ ] Serializador que **remove campos ausentes** em vez de mandar `null`.
- [ ] Helper de `metadata`: enviar só as chaves mexidas, e `null` explícito para
      apagar.
- [ ] Tratamento de `400` com `data.fields`, com o mapa de nome Go → input.
- [ ] `404` como "não encontrado", nunca como "sem permissão".
- [ ] `NOT_A_PARTICIPANT` levando a trocar de organização.
- [ ] Tela de sessões recentes puxando de `/core/users/me`, não do objeto do login.
- [ ] Gráfico de cadastros preenchendo os meses ausentes com zero.
- [ ] Contadores reconsultados depois da ação, nunca deduzidos (o tracking é
      assíncrono).
- [ ] Editor de profile relendo `GET /core/grants` — quatro grants novos.
- [ ] Confirmação na UI antes de trocar `token_type` ou ligar `private` num app.

---

## 6. Ambiente

As duas features adicionam colunas (`users_pool.description`,
`users_pool.users_count`, `users_pool.tracking`, `users.tracking`) e **nenhuma
tabela nova**. `AutoMigrate` as cria com default, então `make migrate up` basta;
`make fresh` refaz tudo do zero.

A documentação OpenAPI de cada rota está em `/docs/` no serviço rodando, com
descrição, corpo e todos os status — é a fonte gerada do contrato, e este
documento é a leitura dela com as armadilhas anotadas.
