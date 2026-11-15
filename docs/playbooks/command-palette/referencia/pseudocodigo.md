# Referência — pseudocódigo sem framework

A lógica inteira da paleta, sem biblioteca. Serve para implementar em qualquer stack e
para conferir uma implementação existente contra o desenho. Notação: funções puras onde
possível; `state` é o que o framework mantém reativo; `effect` é o que roda quando as
dependências mudam.

## Tipos

```
Item   = { id, label, description?, icon?, keywords?: string[], section?, hint? }
Result = Item & { href?: string, action?: () => void, scope?: Scope }

Scope  = {
  id, label, noun, placeholder, grant,
  useResults: (name: string, opts: { enabled: bool, limit: int })
              -> { results: Result[], isLoading: bool, error: Error | null }
}

NavItem = { label, description, to, icon, grants: Grant[], keywords?, search?: Scope }
```

## Casamento (uma função, exportada)

```
collator = Collator(sensitivity = "base")      // ignora caixa e acento

contains(haystack, needle) =
  needle == "" or collator.indexOf(haystack, needle) >= 0

match(item, query) =
  contains(item.label, query) or any(contains(k, query) for k in item.keywords ?? [])
```

## Descritor de escopo (fábrica)

```
defineScope({ id, label, noun, placeholder, grant, query, toResult }) =
  Scope {
    id, label, noun, placeholder, grant,
    useResults(name, { enabled, limit }) =
      q = query({ name: name == "" ? undefined : name, limit })
      r = useCachedQuery(q, {
            enabled,
            keepPreviousData: true,          // dentro do MESMO escopo (a lista remonta ao trocar)
            select: page -> page.data.map(toResult),
          })
      return { results: r.data ?? [], isLoading: r.isLoading, error: r.error }
  }
```

Sem cliente de cache:

```
useResults(name, { enabled, limit }) =
  state results = [], isLoading = false, error = null
  effect on [name, enabled, limit]:
    if not enabled: return
    ctrl = AbortController()
    isLoading = true
    fetchList(query({ name, limit }), ctrl.signal)
      .then(page -> if not ctrl.aborted: results = page.data.map(toResult); error = null)
      .catch(e  -> if not ctrl.aborted: error = e)
      .finally(-> if not ctrl.aborted: isLoading = false)
    cleanup: ctrl.abort()
  return { results, isLoading, error }
```

## Estado da paleta (vive DENTRO do diálogo)

```
state path: Scope[] = []
state text: string  = ""

query     = trim(text)
debounced = useDebounced(query, 300ms)
name      = query == "" ? "" : debounced       // RC-03
scope     = last(path)                         // undefined na raiz

enterScope(scopes: Scope[]) =
  path = path ++ scopes
  text = ""
  search.restart()

popScope() =
  if path.isEmpty: return
  path = path[0 .. -1]
  search.restart()
```

## Lista raiz

```
RootList(query, name) =
  navItems = NAV.filter(i -> i.grants.isEmpty or canAny(i.grants))
  local    = navItems.map(i -> Item {
               id: i.to, label: i.label, description: i.description, icon: i.icon,
               keywords: i.keywords, section: "Pages",
               hint: (i.search and can(i.search.grant)) ? "Tab" : undefined })
             ++ SETTINGS_ITEMS

  visible       = query == "" ? local : local.filter(i -> match(i, query))
  missesLocally = name != "" and not any(match(i, name) for i in local)   // RC-01: mesma função

  // número FIXO de buscas; a permissão só liga/desliga
  searches = FALLBACK_SCOPES.map(s -> (s, s.useResults(name, {
               enabled: missesLocally and can(s.grant), limit: 5 })))
             .filter((s, _) -> can(s.grant))

  fallback = (visible.isEmpty and query != "")
             ? flatMap(searches, (s, r) -> r.results.map(res -> (res, origin: s)))
             : []

  emptyState =
    if searches.isEmpty:                                    "Nothing matches that. Try a page name."
    elif any(r.error for (_, r) in searches):               describeError(firstError)
    elif query != name or any(r.isLoading for (_, r) ...):  Searching
    else:                                                   "Nothing matches “{query}” in pages, A or B."

  items = visible.nonEmpty ? visible : fallback.map((res, origin) -> withTabHint(res))

  onAction(id) =
    if id is settings id: openSettings(id); return
    entry = fallback.find(e -> e.res.id == id)
    navigate(entry ? entry.res.href : id)

  onTab(id) -> bool =
    page = navItems.find(i -> i.to == id)
    if page?.search and can(page.search.grant): enterScope([page.search]); return true
    entry = fallback.find(e -> e.res.id == id)
    if entry?.res.scope and can(entry.res.scope.grant):
      enterScope([entry.origin, entry.res.scope]); return true    // caminho completo
    return false

withTabHint(res) = (res.scope and can(res.scope.grant)) ? res with hint = "Tab" : res
```

## Lista do escopo (componente com key = scope.id)

```
ScopeList(scope, query, name) =
  { results, isLoading, error } = scope.useResults(name, { enabled: true, limit: 10 })

  emptyState =
    if error:                       describeError(error)
    elif isLoading or query != name: Searching
    elif query != "":               "No {scope.noun} match “{query}”."
    else:                           "No {scope.noun} yet."

  items = results.map(withTabHint)

  onAction(id) = r = results.find(id); if r: r.action ? r.action() : navigate(r.href)
  onTab(id)    = r = results.find(id)
                 if r?.scope and can(r.scope.grant): enterScope([r.scope]); return true
                 return false
```

## Teclado no campo

```
onKeyDown(e) =
  if e.key == "Backspace" and input.value == "" and path.nonEmpty:
    popScope(); return                         // sem preventDefault

  if e.key != "Tab" or e.shiftKey: return

  key = highlightedItemKey()                   // da MARCA na lista, não do aria-activedescendant
  if key == undefined: return
  if list.onTab(key):                          // a lista decide e já transiciona
    e.preventDefault()
    e.stopPropagation()                        // o focus trap escuta em document
    search.restart()

highlightedItemKey() =
  listEl = document.getElementById(input.getAttribute("aria-controls"))
  return listEl?.querySelector("[data-focused]")?.dataset.key
```

## Reinício do autocomplete

```
state generation = 0
search.restart() = generation += 1
render: <Autocomplete key={generation} ...>      // remonta campo + lista; o estado (path, text) está acima
```

## Atalho global

```
window.on("keydown", e ->
  if (e.metaKey or e.ctrlKey) and lower(e.key) == "k":
    e.preventDefault()
    isOpen = not isOpen)
```

## Destino sem rota (na tela de destino)

```
PARAM = "selected-user"
openId = int(url.searchParams.get(PARAM)) or null
open(id)  = url.searchParams.set(PARAM, id);  replaceState
close()   = url.searchParams.delete(PARAM);   replaceState
render:   <UserModal userId={openId} onClose={close} />
```

## Invariantes para conferir numa implementação

```
assert match é importada do mesmo módulo em RootList e no primitivo
assert nenhuma string de URL de API dentro da pasta da paleta
assert name == "" sempre que text == ""   (sem esperar o debounce)
assert FALLBACK_SCOPES é constante
assert ScopeList tem key = scope.id
assert onTab retorna false para item sem scope ou sem permissão
assert preventDefault/stopPropagation só quando onTab retornou true
assert estado (path, text) está num componente filho do Root
```
