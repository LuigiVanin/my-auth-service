# Handoff para o frontend: filtros novos em `GET /core/users`

Destino: `auth-service-frontend`, checado em `./frontend`. Escrito em 2026-11-23, junto com
a feature do frontend `docs/features/23-11-2026-console-users-and-detail-fixes/` (RF-10).

**Aditivo.** Nenhum parâmetro existente mudou; quem não manda os novos recebe o mesmo de antes.

## Parâmetros novos

| Parâmetro | Valor | Filtra |
| --- | --- | --- |
| `search` | texto | Nome **ou** e-mail, contém, sem diferenciar maiúsculas. Uma consulta numa tabela só, sem `JOIN`: cada usuário volta uma vez, mesmo casando nos dois campos. `name` e `email` continuam existindo |
| `profile_id` | uuid | Usuários cuja **organização atual** tem esse profile (`organizations.profile_id`). Um usuário não tem permissão própria; o profile de participante de quem se cadastrou é sempre o Admin da própria organização, então não serve de filtro |
| `email_verified` | `true` \| `false` | `users.verify_email` |
| `two_factor` | `true` \| `false` | `users.two_factor_enabled` |
| `signed_up_after` | instante RFC 3339 | Criados nesse instante ou depois (inclusivo) |
| `signed_up_before` | instante RFC 3339 | Criados antes desse instante (exclusivo) |

Instantes, e não datas: "cadastrou em 19/11" depende do fuso de quem pergunta. Um usuário criado
às 22h de 19/11 em Brasília está em 20/11 em UTC, e um filtro por data UTC não o acharia. O
frontend manda a meia-noite local do primeiro dia e a do dia seguinte ao último.

Todos combinam com `name`/`email`, com a paginação e com o `total` (a contagem usa o mesmo
predicado).

## Erros

Valor malformado (`profile_id` que não é uuid, `two_factor=yes`, data fora do formato) responde
`400 BAD_REQUEST` com `data.fields[]` (`field`, `tag`, `value`), antes de chegar ao banco.

## Onde ler o profile na resposta

Cada usuário da listagem já traz `current_organization.profile_id` — é o mesmo valor que o
filtro compara.
