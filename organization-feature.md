Essa vai ser uma das refatorações mais complexas da plataforma e consiste, basicamente, na criação de Organizações de usuários.

### Coisas antes de começar

Antes de começar, preciso explicar alguns pontos que talvez não tenham ficado claros no projeto, mas vamos lá.

Um **app** é basicamente uma interface para interagir com um **users_pool**, que é de fato a entidade que controla e armazena os usuários. Um users_pool pode ter diversos apps, e cada app controla como um usuário pode se registrar, logar, consultar e etc. Por exemplo: se temos um app que permite apenas login e signup via OTP, ao usar esse cliente só é possível autenticar dessa maneira; já em um app que permite login via password — acessando o mesmo grupo de usuários — é possível criar uma conta e logar com senha.

No fluxo entre app, users_pool e usuário, podemos ter o seguinte exemplo:

Existe o app `admin`, associado ao users_pool `main_pool`. Dentro desse mesmo pool existe um usuário chamado `admin`, que é dono da organização `admin_organization`. Esse usuário/organização faz parte do pool e, ao mesmo tempo, é dono dele — algo que **NÃO PODE ACONTECER**, a não ser que você seja o ADMIN da plataforma. Isso faz com que o nosso usuário `admin` tenha acesso a todos os apps e pools dos quais é owner, inclusive ao próprio pool do qual faz parte. INCRÍVEL.

Esse usuário pode criar um novo usuário para o pool citado: será criado o user `primeiro usuário` e, junto com ele, uma organização chamada `primeiro organization's`. Esse usuário será adicionado como **MANAGER**, com permissão para criar outros apps e users_pools e assim replicar o comportamento que descrevi no início — MAS não vai conseguir ver o pool nem o app dos quais faz parte, pois não é owner deles.

### O que é uma organização

Basicamente, são grupos de usuários criados dentro de um `user_pool`, em que cada participante tem suas próprias permissões dentro daquela organização. As ações que um usuário pode desencadear dependem estritamente do que está no profile da sua entidade de participação (`Participant`). Ao realizar um signup em um app — e, consequentemente, em um user_pool — sempre é criada uma organização por baixo dos panos para aquele usuário, mesmo que ele nunca chegue a usufruir dela; por exemplo, em um app de usuário final cujas únicas funcionalidades permitidas são logar e registrar.

Todo usuário de user_pool tem obrigatoriamente uma organização da qual é dono e na qual possui permissões máximas. Essas permissões não são definidas pelo `owner_user_id` da organização, e sim pelo profile daquele usuário na organização, com permissões de `*`. Esse `*`, por sua vez, será restrito apenas pelas permissões da própria organização, que também tem seu profile.

OBS.: uma organização tem um profile destinado a ela, e esse profile deve limitar quais permissões o usuário pode ter dentro daquela organização.

OBS2.: nesse cenário, a entidade `Profile` se torna universal, podendo estar associada a qualquer entidade que tenha um `profile_id`. Vale salientar que, como já explicado acima, as camadas acima do profile limitam as camadas abaixo — por exemplo, `organization -> user`, em que as permissões da organização limitam as permissões do usuário. O mesmo se reflete quando se vai criar um usuário + organização dentro de um user_pool.

### Current Organization

Para que a parte de organização faça sentido, precisamos escopar todas as buscas e alterações ao contexto de uma organização. Por isso criamos o campo `current_organization_id` no usuário, para que ele já carregue consigo a qual organização sua busca de usuários compete, e etc. Para trocar essa informação, basta rodar um update no módulo de organização, via `UPDATE organization/switch`.

Aqui fiquei em dúvida sobre qual abordagem tomar: criar um query param em TODAS as requisições, adicionando `current_organization_id` e tendo que mudar todos os controllers e services; ou adicionar um novo campo no banco de dados, controlando isso no escopo do service.

OBS.: creio que no futuro tenhamos ambas as soluções, adicionando um bypass do current organization via query param, caso o id passado na query seja de uma organização da qual o usuário é participante.

### Esboço

Criei um esboço bem inicial da estrutura que quero para o banco, que pode ser visualizado em `infra/entities/organization.entity.go`, `infra/entities/participant.entity.go` e nas atualizações em `infra/entities/user.entity.go`.

### Implementação

Vamos debater bastante essa solução antes de partir para a implementação. Me tire todas as suas dúvidas e esclareça os caminhos possíveis de implementação. Lembre que é importante sempre atualizar os scripts de inicialização do projeto, como `cmd/database/init.go`, `cmd/database/reset.go` e `cmd/database/migration/main.go`.
