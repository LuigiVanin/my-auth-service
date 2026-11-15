# 06 — Escopos

Um escopo é a unidade de extensão da paleta. Adicionar um recurso pesquisável é escrever
um descritor e apontar um item da navegação para ele; nada na camada 1 ou 3 muda.

## 6.1 O descritor

```
Scope = {
  id:          string            // único; para o subescopo, inclui o id do pai ("pool-users:<id>")
  label:       string            // o segmento desenhado no caminho ("App Clients", "Customers")
  noun:        string            // plural para os textos de vazio ("App Clients", "users")
  placeholder: string            // "Search App Clients by name…"
  grant:       Permission        // a permissão de leitura que habilita o escopo
  useResults:  (name, { enabled, limit }) => { results: Result[], isLoading, error }
}

Result = Item & {
  href:    string                // aonde Enter leva
  action?: () => void            // alternativa ao href, para o que não é navegação
  scope?:  Scope                 // onde Tab entra, se entrar
}

Item = {                          // o formato que a camada 1 entende
  id, label, description?, icon?, keywords?, section?, hint?
}
```

`useResults` é um hook (ou composable, ou store derivada) porque precisa participar do
ciclo de vida do componente que o chama; mas o descritor em si é um **objeto**, para poder
ser referenciado por dados estáticos (a navegação) e devolvido por funções puras (o
mapeamento de um resultado).

## 6.2 A fábrica `defineScope`

O descritor é construído por uma fábrica que recebe o que **varia por recurso** e esconde
o que é igual:

```
defineScope({
  id, label, noun, placeholder, grant,
  query:    (params: { name?, limit }) => QueryDefinition<Page<Row>>,   // a query da tela
  toResult: (row: Row) => Result,                                       // a conversão
})
→ Scope
```

Dentro, `useResults` faz:

1. `query({ name: name || undefined, limit })` — nome vazio vira ausente, não string
   vazia;
2. habilita pela flag recebida;
3. mantém o resultado anterior enquanto carrega (`keepPreviousData`);
4. transforma a página em `rows.map(toResult)` no nível da query (`select`), não no
   render;
5. devolve `{ results: data ?? [], isLoading, error }`.

**O tipo `Row` fica preso na closure.** Quem consome vê só `Scope` e `Result`. É isso que
permite uma lista heterogênea de escopos sem genéricos vazando.

## 6.3 Escopo de primeiro nível

Ligado a um item da navegação por um campo `search`:

```
{ label: "App Clients", to: "/console/apps", icon, grants: ["apps:read"],
  keywords: ["apps", "clients", "keys"],
  search: appsScope }
```

Regras:

- Item **sem** `search` não entra no caminho, não mostra a dica `Tab` e `onTab` devolve
  `false` para ele (`RF-15`). Overview, Organizations (sem filtro no backend) e as seções
  de Settings são exemplos.
- `search.grant` pode ser diferente de `grants` do item (a página pode exigir uma
  permissão e a listagem outra). Checa-se `search.grant` para a dica e para entrar.

## 6.4 Escopo de segundo nível

Nasce de um **resultado** do primeiro nível, numa função que recebe a linha do pai:

```
poolUsersScope(pool) = defineScope({
  id: `pool-users:${pool.id}`,
  label: pool.name,
  noun: "users",
  placeholder: `Search users in ${pool.name} by name…`,
  grant: "users:read",
  query: (params) => userListQuery({ ...params, pool_id: pool.id }),
  toResult: (user) => ({ id: `user:${user.id}`, label: user.name, description: user.email,
                         icon, section: "Users",
                         href: `/console/pools/${pool.id}?selected-user=${user.id}` }),
})

poolsScope.toResult = (pool) => ({ ..., scope: poolUsersScope(pool) })
```

- O `id` do subescopo inclui o id do pai: dois subescopos de pools diferentes são escopos
  diferentes (chave de identidade da lista, `RC-04`).
- O `label` é o **nome do pai**, não o nome do recurso filho: o caminho lê `User Pools /
  Customers /`, e o placeholder diz o que se busca ali.
- Criar o objeto a cada `toResult` é barato e correto; não memoize por id a menos que o
  profiler mande.
- Profundidade maior é o mesmo padrão repetido: o `toResult` do subescopo devolve outro
  `scope`.

## 6.5 Mapeamento linha → resultado

| Campo | Regra |
| --- | --- |
| `id` | Prefixado pelo recurso: `app:<id>`, `pool:<id>`, `user:<id>`. Nunca colide com um destino local (que usa a rota como id) nem com outro recurso (`RC-07`) |
| `label` | O nome que o usuário digitou para achar. Sem truncar aqui; a UI trunca |
| `description` | Um segundo identificador útil: id curto, e-mail, contagem, chave. Serve para desambiguar dois itens com o mesmo nome |
| `icon` | O mesmo ícone do recurso no menu |
| `section` | O nome do recurso no plural; agrupa os resultados da reserva |
| `href` | A rota de detalhe. Para destino sem rota, ver 6.6 |
| `scope` | Só se a API tiver listagem filha filtrável por este pai (`02`) |
| `hint` | **Não** preencher aqui; a camada 3 acrescenta `"Tab"` depois de checar a permissão do `scope` |

## 6.6 Destino sem rota própria

Um recurso que só aparece dentro de outro (usuários dentro da pool, abertos num modal)
não tem `href` natural. A ponte é um **parâmetro de URL que a tela de destino lê para
abrir o modal**:

```
href: `/console/pools/${pool.id}?selected-user=${user.id}`
```

Na tela de destino:

- a URL **é a fonte** do estado "qual modal está aberto" (não um espelho de `useState`);
- abrir pelo clique escreve o parâmetro; fechar apaga; ambos com `replace` no histórico,
  para não acumular entradas;
- o parâmetro some quando o recurso ganhar página própria: troca-se o `href` do escopo e
  remove-se a leitura do parâmetro — duas linhas.

Registre a ponte como dívida com data: "até existir a página de usuário".

## 6.7 Checklist para adicionar um recurso

1. Confirmar no inventário (`02`) que a listagem filtra por nome e qual permissão exige.
2. Confirmar que existe a query da tela de listagem com parâmetros `name` e `limit`.
3. Escrever o `defineScope` com `toResult` (id prefixado, href, description).
4. Se o recurso tem filhos filtráveis pelo id dele: escrever a função do subescopo e
   devolvê-la em `toResult`.
5. Apontar `search:` no item da navegação (primeiro nível) ou `scope:` no `toResult` do
   pai (segundo nível).
6. Se o recurso deve entrar na **busca de reserva**: acrescentar à lista fixa da camada
   3 (lembrando do número fixo de hooks, `05`).
7. Textos: `noun`, `placeholder`, e a frase de vazio da reserva, se mudou.
8. Documentar no catálogo de componentes (ou onde o projeto registra) e rodar o roteiro
   de `08` para o recurso novo.
