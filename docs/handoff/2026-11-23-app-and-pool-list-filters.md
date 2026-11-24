# Handoff para o frontend: ordenação e filtros em `GET /core/apps` e `GET /core/users_pool`

Destino: `auth-service-frontend`, checado em `./frontend`. Escrito em 2026-11-23, na branch
`feat/list-filters`, junto com a feature do frontend
`docs/features/23-11-2026-app-and-pool-list-filters/`.

**Aditivo, com uma mudança de comportamento:** as duas listagens passam a ter ordem. Sem
`sort`, vêm da atualização mais recente para a mais antiga; antes não tinham `ORDER BY`
nenhum, e a paginação podia repetir ou pular linhas.

## `GET /core/users_pool`

| Parâmetro | Valores | Efeito |
| --- | --- | --- |
| `sort` | `updated` (padrão), `name`, `signups`, `created` | Mais recente primeiro, A–Z, mais cadastros primeiro, mais antiga primeiro |
| `ceiling` | `closed`, `scoped`, `broad` | Teto do default profile — a mesma classificação de `pool-ceiling.ts` (`steering/modules/profiles.md`, "Ceiling tones") |
| `scope` | `system`, `organization` | Default profile global, ou da organização atual |

## `GET /core/apps`

| Parâmetro | Valores | Efeito |
| --- | --- | --- |
| `sort` | `updated` (padrão), `name`, `created` | Como acima; não há `signups` aqui |
| `login_type` | `WITH_PASSWORD`, `WITH_OTP`, `WITH_LOGIN` | Apps cujo `login_types` **contém** o método |
| `private` | `true`, `false` | Visibilidade |

Toda ordem desempata pelo id, então duas páginas nunca dividem uma linha. `total` respeita
os filtros.

## Erros

Valor fora do conjunto responde `400 BAD_REQUEST` com `data.fields[]` — antes do banco.
