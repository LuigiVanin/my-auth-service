## Backend

  - [] Testar via interface implementação de organização

  - [] Removendo a escolha de enviorionment como parâmetro de CLI e jogando para uma ENV
  
  - [] Adicionar testes no backend inteiro

  - [] Regra: Um usuário não pode editar o perfil da própria organização, não importa o profile do mesmo

  - [] Mudar UpdateDao para NullableDao para que se possa utilizar o mesmo para gerar queries de Where, isso pensando em casos de fields=false no banco - chamar de UpdateDao é injusto

  - [] Garantir Profiles intransigentes - Profiles não devem estar atrelados diretamente a nenhuma tabela e podem ser associados basicamente a qualquer entidade, dessa maneira podemos ter um profile para um usuário, para uma organização e para um users_pool, dessa maneira temos uma cascata de permissões em que o users_pool limita a organização que limita o usuário.

  - [] Deixar apenas o profile de Login como global, o restante será criado pelo dono da organização
  - [x] Separar rota de users/me e users/uuid , alguns users podem buscar e editar a si mesmo e outros podem modificar a todos os outros que tem visibilidade.
  - [] Rota de verificação de email
  - [] Fluxo de 2fa (Não prioritário)
  - [] Visibilidade em cascata - possibilidade de ver todos os itens associados abaixo da organização em forma de cascata

 - [] Possibilitar criar usuário via console de admin -> deve criar o usuário e enviar um email para definição de senha.

 - [] Legar opções de tokens diferentes do JWT
 - [] Legar opção de loing `WITH_LOGIN

## Frontend
 - [] Adicionar regra no Claude.md para escrever texto em inglês

 - [] Não apresentar opções de tokens diferentes do JWT
 - [] Não apresentar opção de loing `WITH_LOGIN`

 - [] Acertar design de variante principal do botão
  - [] Melhorar aspecto da sombra inset para botão 
 - [] Redesign e refactor no componente de Select e de DataPicker

