## Backend

  - [] Testar via interface implementação de organização

  - [] Removendo a escolha de enviorionment como parâmetro de CLI e jogando para uma ENV
  
  - [] Adicionar testes no backend inteiro

  - [] Regra: Um usuário não pode editar o perfil da própria organização, não importa o profile do mesmo

  - [] Mudar UpdateDao para NullableDao para que se possa utilizar o mesmo para gerar queries de Where, isso pensando em casos de fields=false no banco - chamar de UpdateDao é injusto

  - [] Garantir Profiles intransigentes - Profiles não devem estar atrelados diretamente a nenhuma tabela e podem ser associados basicamente a qualquer entidade, dessa maneira podemos ter um profile para um usuário, para uma organização e para um users_pool, dessa maneira temos uma cascata de permissões em que o users_pool limita a organização que limita o usuário.

  - [] Deixar apenas o profile de Login como global, o restante será criado pelo dono da organização
  - [x] Separar rota de users/me e users/uuid , alguns users podem buscar e editar a si mesmo e outros podem modificar a todos os outros que tem visibilidade.
  - [x] Rota de verificação de email
  - [x] Troca de email verificada pelo próprio usuário (`PUT /auth/change_email`)
  - [x] Logout / revogação de sessão (`POST /auth/revoke`)
  - [] Fluxo de 2fa (Não prioritário)
  - [] Visibilidade em cascata - possibilidade de ver todos os itens associados abaixo da organização em forma de cascata

 - [] Possibilitar criar usuário via console de admin -> deve criar o usuário e enviar um email para definição de senha.

 - [] Legar opções de tokens diferentes do JWT
 - [] Legar opção de loing `WITH_LOGIN

 - [] `GET /core/users_pool`: parâmetro de ordenação (`updated_at desc`, `name asc`, `users_count desc`, `created_at asc`) exposto em `UserPoolListQuery` e no bloco OpenAPI — o `repo.Option.Sort` já existe. E um `ORDER BY` padrão estável: hoje a listagem não tem nenhum e a paginação pode repetir ou pular linhas. A tela usa no menu "Sort by" de `/console/pools`
 - [] `GET /core/users_pool`: filtro por teto (`closed` = `LOGIN_PROFILE`, `broad` = profile com metade `api` ou grant com curinga, `scoped` = o resto), com a classificação de `pool-ceiling.ts` do frontend definida no servidor. A tela usa nas pílulas "Permission ceiling"
 - [] `GET /core/users_pool`: filtro por escopo do profile padrão (`system` = `default_profile.organization_id IS NULL`, `organization` = da organização atual). A tela usa nas pílulas "Profile scope"
 - [] `GET /core/users_pool`: contagem por categoria de teto (facetas), na mesma resposta ou num endpoint próprio, respeitando o filtro de nome. A tela usa no número de cada pílula de teto

 - [] `GET /core/apps`: o mesmo parâmetro de ordenação e o mesmo `ORDER BY` padrão estável da entrada de `users_pool` acima, com `updated_at desc`, `name asc` e `created_at asc`, exposto em `GetAppsQuery` e no bloco OpenAPI. A tela usa no menu "Sort by" de `/console/apps`
 - [] `GET /core/apps`: filtro `login_type` (`WITH_PASSWORD`, `WITH_OTP`, `WITH_LOGIN`), casando o app que **contém** o método em `login_types`. A tela usa nas pílulas "Login method"
 - [] `GET /core/apps`: filtro `token_type` (`JWT`, `FAST_JWT`, `SESSION_UUID`). A tela usa nas pílulas "Token type"
 - [] `GET /core/apps`: filtro `private` (`true`/`false`). A tela usa nas pílulas "Visibility"
 - [] `GET /core/apps`: contagem por categoria (facetas) de `login_type`, `token_type` e `private`, respeitando `name` e `pool_id`. Só se as pílulas forem mostrar número

## Frontend
 - [] Adicionar regra no Claude.md para escrever texto em inglês

 - [] Não apresentar opções de tokens diferentes do JWT
 - [] Não apresentar opção de loing `WITH_LOGIN`

 - [] Acertar design de variante principal do botão
  - [] Melhorar aspecto da sombra inset para botão 
 - [] Redesign e refactor no componente de Select e de DataPicker

