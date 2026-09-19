# Pendências — Registro Central de Dívida Técnica

Tudo que é sabido e ainda não foi feito mora aqui. Item descoberto fora do escopo
de uma tarefa **vem para este arquivo**, não para um `TODO` no código que ninguém
relê.

Cada item diz o que está errado, a consequência e o que precisaria acontecer. Os
que têm dono conhecido citam o arquivo e a linha.

> **Como usar**: ao pegar uma tarefa, varra os itens da área que você vai tocar —
> muitos são baratos de resolver junto e caros de resolver isolados. Ao fechar um
> item, apague-o daqui; o histórico fica no git.

---

## Bloqueantes

Nenhum hoje.

---

## Segurança e privacidade

### 1. O login revela se um e-mail tem conta

`app/modules/login/services/login.service.go:82` responde `404` quando o e-mail
não existe e `401` quando a senha está errada. A diferença entre os dois status é
um oráculo de enumeração de contas.

O frontend já contorna colapsando os dois em `describeApiError`, mas a proteção
mora no cliente, que é o lugar errado. O comentário no código reconhece o
problema.

**Correção**: um único `401` para os dois casos, com o mesmo corpo e idealmente o
mesmo tempo de resposta.

### 2. `PUT /core/users/{id}` expõe histórico de ip de outro usuário

Quem administra a pool lê `tracking.login` de qualquer usuário dela, com `ip` e
`user_agent` dos últimos dez logins. É deliberado — é a rota de administração — mas
não há nenhuma marcação de dado pessoal, nenhum registro de quem leu, e nenhuma
política de retenção além do teto de dez.

**Correção**: decidir se isso é aceitável no modelo de privacidade do produto. Se
não for, o caminho é reduzir o que o evento guarda ou exigir um grant separado.

### 3. `metadata` do OTP aceita campos arbitrários

`app/modules/core/otp/services/otp.service.go:55` registra que nada limita o que é
guardado dentro do metadata do OTP.

### 4. Não há rate limit por endereço de ip

`app/modules/core/otp/services/otp.service.go:54`. O rate limit existente é por
contato, então um ator distribuído passa por ele.

---

## Permissões

### 5. `LOGIN_PROFILE` edita a si mesmo mas não consegue se ler

O profile ganhou `as::users::me::UPDATE` na feature de rotas de update, mas nunca
teve `as::users::me::READ`. Consequência: um usuário recém-cadastrado recebe `403`
em `GET /core/users/me` e `200` em `PUT /core/users/me`.

**Correção**: uma linha em `cmd/database/seeds/profiles.go`. Ficou de fora porque
alargar leitura não era mudança de rota de update, e alargar profile semeado é
mudança de permissão que merece ser decidida à parte.

### 6. A validação de corpo roda antes dos guards

Registrado em [steering/controller-layer.md](../../steering/controller-layer.md).
Um chamador não autenticado distingue payload válido de inválido, e o serviço faz
trabalho de parse para requisições que serão recusadas. Vale revisitar como um
todo, não rota a rota.

### 7. Regra de método de login está no serviço, não num guard

`login.service.go:66` e `register.service.go:237` checam
`slices.Contains(app.LoginTypes, ...)` à mão. Os dois comentários dizem que isso
deveria ser um guard. Enquanto for serviço, cada entrypoint novo precisa lembrar
de repetir a checagem.

---

## Tracking e contadores

### 8. `LoginWithOtp` grava `login_type` como `WITH_PASSWORD`

`app/modules/login/services/login.service.go:170` passa `"WITH_PASSWORD"` para
`CreateNew` num login por OTP. O campo está errado na tabela `sessions` desde
antes, e agora **também** no evento de tracking do usuário.

**Correção**: uma linha. Não foi feita junto com a feature de tracking porque muda
o significado de dado de sessão já gravado, e merece decidir o que fazer com o
histórico existente.

### 9. Um caminho de login novo pode esquecer de chamar o `Track`

O disparo está nos quatro pontos de criação de sessão e **não** no funil
`SessionService.CreateNew`, porque `CreateNew` também é alcançado pelo refresh, que
foi deliberadamente excluído. É o custo da exclusão. Um quinto caminho de login
criado depois não é lembrado por nada.

### 10. `users_count` e `tracking` nunca são reconciliados

`users_count` é monotônico e diverge de `COUNT(*)` na primeira exclusão de usuário.
O tracking de signup não é podado por tempo — uma pool sem cadastro por onze meses
mantém os dez meses antigos até o próximo cadastro empurrá-los.

Nenhum dos dois é errado hoje, mas nada os corrige se ficarem errados.

### 11. `IUserRepository.Create` é público

Nada estrutural impede um quinto chamador de criar usuário fora de
`RegisterService.ProvisionUser` e furar tanto o contador quanto o tracking. Se o
número precisar ser exato, o caminho é fechar a criação atrás do serviço.

### 12. Não existe leitura agregada de tracking entre pools

"Quais pools mais cresceram no mês" não tem como ser respondido — o dado está num
`jsonb` por linha. É o único ponto que justificaria migrar o tracking para tabela
dedicada mais adiante.

---

## Contrato HTTP

### 13. `Session` e `Otp` não têm tags `json`

Todas as suas colunas serializariam em PascalCase (`IpAddress`, `ExpiresAt`) se
alguma rota as devolvesse. **Hoje nenhuma devolve**, então é latente, não um bug
vivo — mas é uma armadilha esperando a primeira rota que exponha sessão.

**Correção**: tags snake_case nas duas entidades, junto com `json:"-"` no que for
secreto, antes de qualquer rota que as exponha.

### 14. Não existe fluxo verificado de troca de e-mail

`constants.ActionChangeEmail` existe e não é usado em lugar nenhum. Consequência:
`PUT /core/users/{id}` é a única forma de trocar e-mail, é rota de administração, e
o próprio usuário não troca o seu.

### 15. `AppService.FindAllUserApps` responde 500

`app/modules/core/app/services/app.service.go:216` devolve
`Not Implemented Yet`. Não há rota apontando para ele hoje.

### 16. Métodos de serviço escritos e nunca chamados

`IOrganizationService.Create` e `SetOwner`
(`app/modules/core/organization/services/interface.go:13`) e
`IParticipantService.Create`
(`app/modules/core/participant/services/interface.go:17`) existem porque as
escritas equivalentes acontecem via repositório dentro de transação. Os próprios
comentários dizem que ficam para as rotas que farão essas mutações sozinhas.

São código morto até lá, e código morto que implementa interface não quebra em
compilação quando a regra em volta muda.

---

## Infraestrutura e testes

### 17. Não há teste de integração com banco

Tudo roda em `DryRun` com um pool de conexão falso, ou com mocks. Isso cobre a
forma do SQL e a regra de negócio, mas **nada** verifica que uma migração aplica,
que uma constraint dispara, ou que um lock se comporta como o esperado.

Os dois casos mais expostos: o `FOR UPDATE` do tracking e a ordem de lock do
`users_count` dentro da transação de provisionamento — ambos documentados em
prosa, nenhum verificado contra um Postgres real.

### 18. O `go` do accept loop do servidor não tem `recover`

`infra/bootstrap/webserver.go:94`. É o único `go` que sobrou fora do
`utils.Detach`, e é deliberado: roda pelo tempo de vida do processo, não é trabalho
de requisição, e um panic ali talvez deva mesmo ser fatal. Fica registrado para a
decisão ser consciente, não esquecida.

### 19. `error_handler.go` carrega um caminho legado

`app/middlewares/error_handler.go:42` diz que o tratamento ali deixou de ser
necessário quando o validador passou a lançar `AppError`.

---

## Documentação

### 20. `docs/features/` mistura duas gerações

Três features antigas são arquivos `.md` soltos, de antes da estrutura de pastas;
duas novas são pastas com `requirements.md` + `implementation-plan.md`, e as duas
foram **reconstruídas depois** da implementação — exatamente o antipadrão que a
regra existe para evitar, mantidas porque o histórico vale mais que a consistência.

Da próxima feature em diante, o par nasce antes do código.

### 21. `steering/guards-and-middlewares.md` descreve quatro guards, e existem cinco

A seção "The guards" abre com "All four implement `interfaces.IGuard`" e não tem
uma entrada para o `OrganizationGuard`, que é encadeado em quase toda rota de
`/core` e escreve `organization` e `participant` em `Locals`. Quem ler o documento
para descobrir o que cada guard exige e o que deixa no contexto não encontra o
único que resolve o escopo da requisição.

### 22. `sessions.invalidated` mistura "superada" com "encerrada"

A coluna é escrita por um único lugar, o `InvalidateAllExcept` do `CreateNew`, e
hoje ela só significa "nasceu uma sessão mais nova". Por isso o `/auth/refresh`
passou a aceitá-la: recusar derrubava quem não fez nada errado, porque outro
cliente logou.

Quando o logout for implementado — o `Revoke` ainda é `NotImplemented` — ele vai
precisar de uma coluna própria (`revoked_at`, ou um motivo na própria
`invalidated`). Sem isso, um refresh token de uma sessão encerrada de propósito
revive, que é exatamente o que o logout existe para impedir.
