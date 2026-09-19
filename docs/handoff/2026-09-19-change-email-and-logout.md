# Handoff para o frontend: troca de e-mail verificada e logout de verdade

Destino: `auth-service-frontend`, checado em `./frontend`, no commit `2fd0777`.
Origem: `auth-service`, feature `docs/features/2026-09-19-change-email/`.

**Duas rotas novas, uma resposta corrigida e um comportamento que o frontend
precisa antecipar.** As rotas são aditivas. A resposta corrigida é a de
`PUT /auth/verify_email`, que estava **errada**. O comportamento é o logout
implícito da troca de e-mail, e é a única parte que quebra se os dois lados não
subirem na mesma janela.

O que **não** foi verificado: nada disso rodou contra o frontend. Foi verificado
contra a suíte do backend e contra o documento OpenAPI gerado. Os caminhos de
arquivo abaixo foram lidos no commit citado, mas o patch não foi aplicado lá.

---

## Por que isso importa agora

`AccountSettingsPanel.tsx` já tem o lugar reservado, desabilitado, com `Badge`
de "Soon":

```tsx
<SettingsGroup
  title="Security"
  description="Changing your email and password will be available soon."
>
  <SecurityRow
    icon={<Mail className="h-4 w-4" aria-hidden="true" />}
    title="Email address"
    description={current.email}
    action="Change email"
  />
```

E o campo de e-mail do `AccountForm` carrega o aviso que deixa de ser verdade:

```tsx
<TextField
  label="Email"
  type="email"
  value={user.email}
  isDisabled
  hint="Managed by your pool administrator for now."
/>
```

A metade de e-mail dessas duas frases sai. A de senha continua.

---

## 1. `PUT /auth/verify_email` — a resposta mudou, e antes estava errada

Três defeitos, corrigidos de uma vez:

| Antes | Agora |
| --- | --- |
| A rota **nunca verificava ninguém**: a saída antecipada estava invertida e pulava a escrita justamente para quem ainda não era verificado | A escrita acontece; chamar de novo para quem já é verificado é no-op |
| O OTP continuava válido depois do uso, pelos 5 minutos restantes | O OTP é consumido |
| O corpo devolvia o usuário **antes** da escrita, com `verify_email: false` | O corpo devolve a linha recém gravada, com `verify_email: true` e `verified_email_date` preenchido |

O envelope não mudou: `201` com `{ message, user }`.

**O que fazer:** se existir algum remendo forçando `verify_email = true` no
estado local depois do `201`, ele pode sair. Nenhuma tela do commit lido consome
essa rota ainda, então na prática isto é informação para quando a tela nascer.

---

## 2. `POST /otp/generate_consumable?action=CHANGE_EMAIL`

Autenticada. O `OtpGuard` isenta só `REGISTER`, `LOGIN` e `FORGOT_PASSWORD`; todo
o resto cai no `AuthGuard`. **O comentário de cabeçalho de
`src/services/otp.ts` diz que "estas chamadas funcionam sem sessão" — deixa de
valer para a função nova, e vale a pena ajustar a frase.**

O corpo é o genérico de um campo, o mesmo de `FORGOT_PASSWORD`, e **o e-mail é o
novo**:

```jsonc
// Authorization: Bearer <access token>
{ "email": "novo@exemplo.com" }
```

```jsonc
// 200 OK — mesmo GenerateOtpResponse de sempre
{ "otp_id": "...", "app_id": "...", "action": "CHANGE_EMAIL",
  "expires_at": "...", "created_at": "..." }
```

O código de 6 dígitos vai **para o endereço novo**. É isso que prova a posse, e é
por isso que a interface tem que dizer "enviamos um código para
`novo@exemplo.com`" e não "para o seu e-mail". O `OtpVerificationModal` já recebe
`email` como prop e escreve a frase certa sozinho.

Valem os dois relógios de sempre: 5 minutos de validade, 20 segundos entre dois
pedidos para o mesmo contato (`429 TOO_MANY_REQUESTS`).

**Status novo nesta rota:** `409 USER_ALREADY_EXISTS` quando o endereço já
pertence a alguém da pool. `describeApiError` já traduz esse código para "An
account with this email already exists." — não precisa de entrada nova.

### `src/types/otp.ts`

```ts
/**
 * Body de `/otp/generate_consumable?action=CHANGE_EMAIL`.
 *
 * Mesmo shape do `FORGOT_PASSWORD`, polaridade oposta: ali o e-mail tem que
 * existir na pool, aqui tem que **não** existir — é para onde a conta está indo.
 * O backend responde 409 `USER_ALREADY_EXISTS` se já existir.
 */
export interface GenerateChangeEmailOtpPayload {
  email: string;
}
```

`AuthAction` e `AnonymousAuthAction` já estão certos — `CHANGE_EMAIL` está na
união e fora da isenção. Nada a mudar ali.

### `src/services/otp.ts`

```ts
export async function generateChangeEmailOtp(
  payload: GenerateChangeEmailOtpPayload,
): Promise<GenerateOtpResponse> {
  const { data } = await api.post<GenerateOtpResponse>(
    "/otp/generate_consumable",
    payload,
    { params: { action: "CHANGE_EMAIL" } },
  );

  return data;
}
```

---

## 3. `PUT /auth/change_email` — confirmar

```jsonc
// Authorization: Bearer <access token>
{
  "email": "novo@exemplo.com",
  "otp": { "id": "<otp_id do passo 2>", "code": "123456" }
}
```

```jsonc
// 200 OK
{
  "message": "email changed with success",
  "user": { "id": 1, "email": "novo@exemplo.com", "verify_email": true,
            "verified_email_date": "...", "...": "..." }
}
```

O e-mail do corpo tem que ser o mesmo para o qual o código foi enviado — o
backend confere contra o metadata do OTP e responde `401` se divergir. Mesma
mecânica que `ForgotPasswordPage` já trata ao congelar o `email` no estado
`PendingReset` em vez de reler do formulário. **Copiar essa decisão aqui.**

`verify_email` sai `true`: o código foi entregue no endereço novo e consumido,
então a posse está provada. Sem isso o usuário seria barrado em toda rota
`/core` pelo `EmailVerificationGuard` logo depois de trocar.

### ⚠️ A armadilha: o `200` é também um logout

**Todas** as sessões daquele usuário naquela aplicação são encerradas dentro da
mesma transação que move o e-mail. Os dois tokens morrem — o de acesso e o de
refresh.

O `api.ts` até se recupera sozinho: a próxima requisição toma `401`, o
interceptor tenta o refresh, o refresh falha, cai no `dropSession()` e o
`ProtectedRoute` redireciona. Mas o caminho passa por uma rodada de toasts de
erro e por um refetch inútil, e o usuário vê uma tela quebrada antes do login.

Então, ao receber o `200`:

1. **não** invalide query nenhuma — nada de `queryClient.invalidateQueries({ queryKey: userKeys.me() })`
   como o `AccountForm` faz no `onSuccess` do update de perfil. Esse refetch sai
   com um token já morto;
2. chame `dropSession()` de `@/lib/api`. Ele limpa o `localStorage`, notifica o
   `subscribeToSessionChange`, e o `AuthProvider` faz o resto: como o
   `nextUserId` vira `null` e difere do `cachedUserId`, ele **já chama
   `queryClient.clear()`** e move o status para `anonymous`. Não há cache a
   limpar à mão;
3. dê o toast antes, com o texto que explica o que aconteceu, e deixe o
   `ProtectedRoute` levar para `/login`.

**Não tente renovar.** `POST /auth/refresh` com o refresh token antigo responde
`TOKEN_EXPIRED`.

Vale para todos os aparelhos: quem estava logado em outro navegador cai também,
na próxima requisição. É deliberado — a identidade mudou.

### Status

| Status | Código | Quando |
| --- | --- | --- |
| 200 | — | trocado; **o usuário está deslogado** |
| 400 | `BAD_REQUEST` | corpo inválido, OTP errado/usado/expirado, ou o e-mail enviado é o que já está na conta |
| 401 | `UNAUTHORIZED` | credenciais de app ou token ausente/inválido, ou o OTP guarda outro e-mail |
| 404 | `NOT_FOUND` | nenhum app para o `X-Public-Key`, ou o OTP não existe |
| 409 | `USER_ALREADY_EXISTS` | outro usuário da pool já usa esse e-mail |

O `409` aparece **duas vezes** no fluxo, no passo 2 e no passo 3, e a segunda é
a que pega o endereço tomado no meio do caminho. A tela precisa tratá-lo na
confirmação, não só na geração.

### `src/lib/api-error.ts`

`describeOtpError` já cobre quase tudo. Falta uma linha em
`OTP_MESSAGE_BY_DETAIL` para o único `detail` novo que o backend escreve:

```ts
"The new email is the one already on the account":
  "This is already your email address.",
```

`"Email does not match OTP metadata"` já está lá e é o mesmo texto reaproveitado.
`USER_ALREADY_EXISTS` e `TOO_MANY_REQUESTS` já saem certos por `MESSAGE_BY_CODE`.

**Use `describeOtpError`, não `describeCoreError`.** A tela vive no console, mas
o erro é de ciclo de vida de OTP — `describeCoreError` traduziria o `404` do OTP
inexistente para "Record not found in this organization.", que não diz nada aqui.

### `src/types/auth.ts`

`OtpConsumption` já é importado de `./otp` neste arquivo, para os payloads de
login e cadastro por OTP — não precisa de import novo.

```ts
/** Body de `PUT /auth/change_email`. */
export interface ChangeEmailPayload {
  email: string;
  otp: OtpConsumption;
}

/**
 * Resposta 200 de `PUT /auth/change_email`.
 *
 * Não traz tokens, e isso é o ponto: o backend encerrou todas as sessões do
 * usuário neste app junto com a escrita. Quem chama tem que descartar a sessão.
 */
export interface ChangeEmailResponse {
  message: string;
  user: AuthUser;
}
```

### `src/services/auth.ts`

```ts
/**
 * Troca o e-mail consumindo o OTP. Lança `AxiosError` em falha.
 *
 * **Não** persiste sessão, e a ausência de `saveSession` aqui é ainda mais
 * significativa que em `resetPassword`: o backend não só responde sem tokens,
 * ele mata os que o chamador já tinha. Quem chama precisa de `dropSession()`.
 *
 * Por isso esta função entra no `AuthContext`, ao contrário do `resetPassword`:
 * ela mexe em sessão.
 */
export async function changeEmail(
  payload: ChangeEmailPayload,
): Promise<ChangeEmailResponse> {
  const { data } = await api.put<ChangeEmailResponse>(
    "/auth/change_email",
    payload,
  );

  return data;
}
```

---

## 4. `POST /auth/revoke` — o logout explícito

```
POST /auth/revoke
Authorization: Bearer <access token>
```

Sem corpo. Encerra a sessão daquele token — os dois tokens dela. Responde `200`
com `{ "message": "session revoked" }`, e é idempotente.

Encerra **só aquela sessão**; as de outros aparelhos continuam. Não existe rota
que liste sessões, então também não há como revogar a de outro aparelho.

### `src/services/auth.ts`

```ts
/**
 * Encerra a sessão atual no servidor. Lança `AxiosError` em falha.
 *
 * Idempotente: revogar de novo responde igual. Não descarta a sessão local —
 * quem faz isso é o `logout` do `AuthContext`, e ele faz mesmo quando esta
 * chamada falha.
 */
export async function revokeSession(): Promise<void> {
  await api.post("/auth/revoke");
}
```

### `src/contexts/AuthContext.tsx`

Hoje o logout é local e só:

```ts
const logout = useCallback(() => {
  dropSession();
}, []);
```

A sessão sobrevivia no banco até expirar. Passa a ser:

```ts
const logout = useCallback(async () => {
  // O servidor pode recusar - token já expirado, rede caída. O usuário pediu
  // para sair, então sai: o descarte local não fica refém da chamada.
  await revokeSession().catch(() => undefined);

  dropSession();
}, []);
```

`AuthContextType` em `src/contexts/auth-context.ts` muda de
`logout: () => void` para `logout: () => Promise<void>`.

**`UserMenu.tsx:61` passa `onAction={logout}` direto.** Com o logout assíncrono
isso continua compilando e funcionando — `onAction` ignora o retorno — mas o
menu fecha antes do servidor responder. Aceitável: o descarte local é o que o
usuário percebe. Se quiser estado de carregamento, é ali que ele entra.

### Um caso de borda que o interceptor cria

Se o access token já estiver expirado na hora do logout, o `401` do `/auth/revoke`
dispara o refresh do interceptor, a sessão **rotaciona**, e o retry revoga a
sessão **nova**. O usuário sai, que é o que importa.

O resíduo é do backend, não do frontend: a sessão antiga fica com
`invalidated = true` mas com a janela de refresh intacta, então o refresh token
dela ainda serviria. É o item 22 de `docs/features/pendencias.md`, e some quando
`revoked_at` existir.

**Não acrescente `/auth/revoke` a `NO_REFRESH_PATHS`.** Isso faria o logout
falhar em vez de revogar, e a sessão válida ficaria viva.

---

## 5. A tela

O fluxo tem a mesma forma de três fases do `ForgotPasswordPage`, e dá para
reaproveitar quase tudo: o `OtpVerificationModal` de `@/components/auth` já
recebe `email`, `expiresAt`, `onSubmit`, `onResend`, `title` e `submitLabel`, e
já mostra validade e cooldown pelo `useCountdown`.

A diferença é que aqui existe uma fase zero: **avisar que vai deslogar**. No
esqueci-a-senha o usuário já está fora; aqui ele está dentro do console, com o
modal de configurações aberto, e vai ser expulso. Isso tem que estar escrito
antes de o código ser enviado, não depois.

```
Fase 0  SecurityRow "Change email" → abre o formulário com o campo do e-mail
        novo e o aviso: "Você será desconectado de todos os aparelhos e vai
        entrar de novo com o endereço novo."

Fase 1  generateChangeEmailOtp({ email }) → 200 abre o OtpVerificationModal
        409 → "An account with this email already exists."

Fase 2  o modal coleta o código. Aqui há uma escolha - ver abaixo.

Fase 3  changeEmail({ email, otp }) → 200 → toast → dropSession()
```

### Verificar antes ou consumir direto?

`ForgotPasswordPage` chama `verifyOtp` na fase 2 e só consome na fase 3, porque
entre as duas existe uma tela inteira — escolher a senha nova — e descobrir o
código errado depois disso seria cruel. **Aqui não há tela no meio**: o código é
a última coisa que o usuário digita.

Então a recomendação é consumir direto, chamando `changeEmail` do próprio
`onSubmit` do modal, como o login por OTP faz. O custo está registrado e é o
mesmo do login: **o teto de 5 tentativas vive só no `VerifyConsumable`**, então
este caminho não o tem. Se preferir o teto, chame `verifyOtp` antes e
`changeEmail` logo em seguida — as duas chamadas em sequência, sem tela entre
elas.

### Onde mexer

| Arquivo | O quê |
| --- | --- |
| `src/components/console/settings/AccountSettingsPanel.tsx` | `SecurityRow` do e-mail perde `<Badge>Soon</Badge>` e `isDisabled`, e ganha o `onAction`. O `description` do `SettingsGroup` "Security" perde a metade sobre e-mail. O `hint` do `TextField` de e-mail no `AccountForm` deixa de dizer "Managed by your pool administrator for now." |
| `src/components/console/settings/` | o formulário e o estado das fases, no formato do `PendingReset` do `ForgotPasswordPage` — `{ email, otpId, expiresAt }` congelados na geração |
| `src/contexts/AuthContext.tsx` | `logout` assíncrono com `revokeSession`; `changeEmail` novo, porque mexe em sessão |
| `src/contexts/auth-context.ts` | as duas assinaturas no `AuthContextType` |

`SecurityRow` hoje tem `action: string` e nada de handler — vai precisar de
`onAction?: () => void` e de tornar `Badge`/`isDisabled` condicionais, já que a
linha de senha continua desabilitada.

### Não trave a tela num grant

`AccountSettingsPanel` usa `useCan` para `as::users::me::READ` e
`as::users::me::UPDATE`, e a tentação é fazer o mesmo com
`as::change_email::UPDATE`. **Não faça.** O `PermissionsGuard` não é encadeado
em nenhuma rota de `/auth` — os grants novos existem no catálogo e no
`LOGIN_PROFILE`, mas são expressivos, não efetivos. Travar o botão neles
esconderia um fluxo que funciona para quem tiver um profile customizado sem a
chave, e não protegeria nada para ninguém.

Trocar o próprio e-mail é ato sobre a própria conta, resolvido pelo token. Não
tem escopo de organização para consultar.

---

## Como verificar

Em `development` o código do OTP vai para o log do backend em claro, então dá
para rodar o fluxo inteiro sem caixa de entrada:

1. login, guarde os dois tokens;
2. `POST /otp/generate_consumable?action=CHANGE_EMAIL` com o e-mail novo →
   `200`, código no log;
3. repita com um e-mail que já existe → `409`;
4. `PUT /auth/change_email` → `200`, `user.email` novo e `verify_email: true`;
5. `POST /auth/refresh` com o refresh do passo 1 → `TOKEN_EXPIRED`. **Este é o
   passo que prova o logout**;
6. login com o e-mail novo → `200`;
7. `POST /auth/revoke`, depois `/auth/refresh` daquela sessão → `TOKEN_EXPIRED`.

Na tela, o que vale conferir é o passo 4: depois do `200`, o console tem que ir
para `/login` **sem** nenhum toast de erro no caminho. Um toast de erro ali
significa que sobrou um refetch com token morto.

---

## Fora de escopo, se quiser abrir issue

- **Não há como trocar o e-mail sem estar logado.** Quem perdeu acesso ao
  endereço antigo e não lembra a senha não tem saída pelo produto.
- **O `409` conta se um endereço existe na pool** para um chamador autenticado.
  É o mesmo oráculo de enumeração que o backend já tem no login e no cadastro, e
  está registrado como item 24 em `docs/features/pendencias.md`.
- **A sessão encerrada responde `TOKEN_EXPIRED`, e não algo como
  `SESSION_REVOKED`.** O backend fecha a janela de refresh em vez de marcar a
  linha como revogada; a coluna própria é o item 22. Se a tela precisar
  diferenciar "expirou" de "encerrada", é esse item que destrava.
- **O OTP de `CHANGE_EMAIL` não fica preso a quem o pediu** — item 23. Não muda
  nada no frontend hoje; está aqui para o caso de a tela vir a guardar o
  `otp_id` em algum lugar compartilhado.
- **O corpo do e-mail do OTP ainda é uma frase sem template**, e o assunto
  interpola a ação crua: `OTP Code - CHANGE_EMAIL`.
