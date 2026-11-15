# Handoff para o frontend: convite para uma organização

Destino: `auth-service-frontend`, checado em `./frontend`, no commit `111e824`.
Origem: `auth-service`, feature `docs/features/2026-10-13-organization-invites/`.

**Cinco rotas novas, um `method` novo no cadastro e um campo novo na organização.**
Tudo é aditivo: nenhuma rota existente muda de forma, e nada quebra se o frontend
subir depois do backend. O campo `invite_ttl_seconds` passa a aparecer em toda
resposta de organização, com `null` quando não foi definido.

O que **não** foi verificado: nada disso rodou contra o frontend. Foi verificado
contra a suíte do backend, contra o documento OpenAPI gerado e contra o servidor de
pé, de ponta a ponta — criar, descrever, cadastrar por convite, aceitar com conta,
substituir, revogar e listar, com o e-mail de verdade saindo pelo Resend. Os caminhos
de arquivo abaixo foram lidos no commit citado, mas nenhum patch foi aplicado lá.

---

## Por que isso importa agora

Três trechos do frontend dizem que o convite não existe, e deixam de ser verdade:

- `src/pages/console/ParticipantsPage.tsx:112` — "There is no invite or participant
  creation in /core: a user joins an organization by signing up or by creating it."
- `docs/core-api.md` §4, linha 261 — "Não há convite nem criação de participante."
- `docs/core-api.md` §10, linhas 641–643 — "nem rota de convite" e "Rota de convite
  ou criação de participante."

---

## O fluxo, de ponta a ponta

```
console (quem convida)                      e-mail                   página de aceite (quem recebe)
──────────────────────                      ──────                   ──────────────────────────────
POST /core/organizations/{id}/invites  ──►  link com ?token=…  ──►   GET /auth/invite?token=…
GET  /core/organizations/{id}/invites                                   │
DELETE …/invites/{invite_id}                                            ├─ account_exists: true
                                                                        │     POST /auth/invite/accept { token }
                                                                        │
                                                                        └─ account_exists: false
                                                                              POST /auth/register?method=invite
                                                                                   { token, name, password? }
```

O link aponta para a página configurada no backend em `INVITE_ACCEPT_URL`
(desenvolvimento: `http://localhost:3000/invite?token={token}`). **A rota `/invite`
não existe hoje em `src/App.tsx`**: precisa nascer, fora do layout autenticado,
como `/login` e `/signup`.

---

## 1. Console — criar, listar, revogar

As três exigem a organização **atual**: `{id}` diferente dela responde `403`, como
em `PUT /core/organizations/{id}`. As três passam pela cadeia completa de guards.

| Rota | Grant | Sucesso |
| --- | --- | --- |
| `POST /core/organizations/{id}/invites` | `as::organizations::invites::CREATE` | `201 InviteResponse` |
| `GET /core/organizations/{id}/invites` | `as::organizations::invites::READ` | `200` página de `InviteResponse` |
| `DELETE /core/organizations/{id}/invites/{invite_id}` | `as::organizations::invites::DELETE` | `200 InviteResponse` (revogado) |

**Quem tem os grants**: só organizações com teto `MANAGER_PROFILE` ou maior. Uma
organização nascida numa pool comum (teto `LOGIN_PROFILE`) não convida, não lista e
não revoga. Use o gate de sempre (`can`) com as três chaves; não infira pelo profile.

### Criar

```json
{ "email": "fulano@exemplo.com", "profile_id": "<uuid>", "ttl_seconds": 86400 }
```

- `profile_id` tem de ser visível à organização e **não pode dar mais do que quem
  convida tem** ali — senão `403`. O seletor deve oferecer os mesmos profiles que a
  troca de profile de participante oferece.
- `ttl_seconds` é opcional, de `3600` (1 hora) a `2592000` (30 dias). Ausente, vale o
  `invite_ttl_seconds` da organização, e depois o padrão do serviço, 7 dias.
- Convidar de novo o mesmo e-mail **substitui** o convite pendente: o link anterior
  para de funcionar. É o "reenviar convite" — não existe rota separada.
- O token **nunca** volta na resposta; ele só viaja no e-mail.

| Status | Quando |
| --- | --- |
| `400` | payload inválido, ou profile que a organização não enxerga |
| `403` | `{id}` não é a organização atual, ou profile acima de quem convida |
| `409` | o e-mail já participa da organização |
| `501` | o servidor não tem `INVITE_ACCEPT_URL` configurado |

### Listar

`?status=pending|accepted|revoked|expired` (opcional; ausente lista todos),
`?skip`, `?limit`. Ordem: mais novo primeiro. `status` desconhecido é `400`.

### Revogar

`404` para id que não é desta organização; `409` para convite que já não está
pendente (o `detail` diz o estado).

### `InviteResponse`

```ts
// src/types/invites.ts
export type InviteStatus = "pending" | "accepted" | "revoked" | "expired";

export interface Invite {
  id: string;
  organization_id: string;
  profile_id: string;
  invited_by_user_id: number;
  email: string;
  expires_at: string;
  accepted_at: string | null;
  revoked_at: string | null;
  profile?: Profile;          // o mesmo shape de Participant.profile
  created_at: string;
  updated_at: string;
  status: InviteStatus;
}

export interface InvitesPage {
  total: number;
  amount: number;
  skip: number;
  limit: number;
  data: Invite[];
}
```

---

## 2. Página de aceite — `/invite?token=…`

### ⚠️ Tire o token da URL assim que ler

O token fica no histórico do navegador e no `Referer` de qualquer requisição que a
página fizer para fora. Leia uma vez, guarde em memória e limpe a URL:

```ts
const token = new URLSearchParams(location.search).get("token");
history.replaceState(null, "", location.pathname);
```

### Descrever — `GET /auth/invite?token=…`

Só os cabeçalhos da aplicação (`X-Pool-Key`, `X-Public-Key`), sem sessão.

```ts
export interface InviteDescription {
  organization: { id: string; name: string; avatar_url: string | null };
  invited_by: { name: string };
  profile: { id: string; name: string; key: string };
  email: string;
  expires_at: string;
  account_exists: boolean;
}
```

`account_exists` decide a tela: `true` mostra "Aceitar convite"; `false` mostra o
formulário de cadastro com o e-mail **fixo**, vindo da descrição.

| Status | O que mostrar |
| --- | --- |
| `404` | "Convite inválido" — token desconhecido, segredo errado, ou de outra aplicação |
| `400` | "Convite já usado, revogado ou expirado" — o `detail` diz qual dos três |

Um convite só é lido pela **mesma aplicação que o emitiu**. Se o frontend do convite
usar outras chaves de aplicação, tudo responde `404`.

### Aceitar com conta — `POST /auth/invite/accept`

```json
{ "token": "<token>" }
```

**Não precisa de login.** O token prova que a pessoa recebeu o e-mail; a conta
daquele e-mail entra na organização com o profile do convite, e o e-mail passa a
verificado. Nenhuma sessão é aberta.

```ts
export interface AcceptInviteResponse {
  organization: OrganizationView;   // com avatar_url
  participant: ParticipantView;
}
```

| Status | Quando |
| --- | --- |
| `200` | aceito; também quando a pessoa já participava (o profile dela não muda) |
| `400` | já aceito, revogado ou expirado |
| `404` | token inválido, ou o e-mail não tem conta na pool — mande para o cadastro |

Depois do `200`, a organização atual da pessoa **não muda**. Se ela estiver logada,
ofereça "Ir para {organização}" chamando `PUT /core/organizations/switch`. Se não
estiver, mande para o login.

### Cadastrar por convite — `POST /auth/register?method=invite`

```json
{ "token": "<token>", "name": "Fulano", "password": "********", "phone": null, "metadata": {} }
```

- **Sem `email`**: ele vem do convite.
- `password` é obrigatório se a aplicação aceita `WITH_PASSWORD` (senão `422`); numa
  aplicação só OTP, omita.
- Resposta: o mesmo `RegisterResponse` do cadastro normal — sessão, tokens e
  `user`. Trate como um signup: guarde a sessão.
- O usuário nasce com e-mail verificado, dono da **própria** organização, e
  participando da organização do convite. A organização atual é a própria; para ir à
  do convite, `PUT /core/organizations/switch` com o `id` da descrição.

| Status | Quando |
| --- | --- |
| `201` | cadastrado e aceito |
| `400` | convite já usado, revogado ou expirado |
| `404` | token inválido |
| `409 USER_ALREADY_EXISTS` | o e-mail já tem conta — use o aceite |
| `422` | falta `password` numa aplicação com senha |

---

## 3. Organização — `invite_ttl_seconds`

`PUT /core/organizations/{id}` aceita `invite_ttl_seconds` (`3600` a `2592000`), e
toda resposta de organização passa a trazer o campo, `null` quando não definido.

```ts
// src/types/organizations.ts — no tipo da organização
invite_ttl_seconds: number | null;
```

Um valor definido **não volta a `null`** pelo payload — `null` é "não enviado". O
campo de edição pode trocar o valor, não limpar.

---

## Como verificar

1. Suba o backend da branch `feat/organization-invites` com `INVITE_ACCEPT_URL`
   apontando para a página do frontend.
2. Logado como o admin semeado, crie um convite para `delivered@resend.dev`
   (endereço de teste do Resend — o e-mail é aceito e descartado).
3. O token não volta na API; em desenvolvimento, `otps.code` guarda o segredo em
   claro e o token é `<otps.id>.<otps.code>` da linha apontada por `invites.otp_id`.
4. Abra `/invite?token=…`: deve mostrar o formulário de cadastro; cadastre.
5. Crie outra organização com teto Manager, troque para ela, convide o mesmo
   endereço e abra o novo link: agora deve mostrar "Aceitar".

---

## Fora de escopo, se quiser abrir issue

- Reenviar sem trocar o link — hoje reenviar é criar de novo, e o link anterior
  morre.
- Aceite na mesma tela do login, encadeando o switch.
- Limite de envio por quem convida (`pendencias.md` #37).
