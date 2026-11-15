# Referência — a implementação de origem (React 19 + React Aria + TanStack Query)

**Isto é exemplo, não exigência.** O código abaixo é o que está em produção no projeto
de origem, anotado para mostrar onde cada regra do playbook se materializa. Se a sua
stack for parecida, dá para copiar e adaptar; se não for, use `pseudocodigo.md` e esta
página só para ver as decisões em contexto.

Stack de origem: React 19, React Aria Components 1.15 (`Autocomplete`, `Menu`,
`Dialog`), TanStack Query 5, React Router 7, Tailwind 4, Lucide (ícones). Backend REST em
Go com listagens `GET /core/<recurso>?name=&skip=&limit=` e envelope
`{ total, amount, skip, limit, data }`.

## Mapa de arquivos

| Arquivo | Camada | Conteúdo |
| --- | --- | --- |
| `src/components/ui/command-match.ts` | 1 | `useCommandMatch()`: a função de casamento, exportada |
| `src/components/ui/CommandPalette.tsx` | 1 | `CommandPaletteRoot`, `…Search`, `…Input`, `…List`, `…Footer`, `…Hint` e a montagem pronta `CommandPalette` |
| `src/components/console/layout/command-scopes.ts` | 2 | `defineScope`, `appsScope`, `poolsScope` (+ `poolUsersScope`), `profilesScope` |
| `src/components/console/layout/navigation.ts` | 2/3 | `CONSOLE_NAV` com o campo `search?: CommandScope` |
| `src/components/console/layout/ConsoleCommandPalette.tsx` | 3 | `ConsolePaletteBody` (estado), `RootList` (local + reserva), `ScopeList` |
| `src/components/console/layout/ConsoleLayout.tsx` | 3 | O `⌘K` e o `isOpen` |
| `src/components/console/pools/PoolUsersSection.tsx` | destino | Lê `?selected-user=` e abre o modal |
| `stories/ui/CommandPalette.stories.tsx` | teste | Montagem pronta, caminho em memória, carregando |

## Camada 1 — o casamento

```ts
// command-match.ts
export function useCommandMatch() {
  const { contains } = useFilter({ sensitivity: "base" }); // sem caixa, sem acento

  return useCallback(
    (item: CommandItem, query: string) =>
      contains(item.label, query) ||
      (item.keywords ?? []).some((keyword) => contains(keyword, query)),
    [contains],
  );
}
```

É importada pelo primitivo (filtro padrão) **e** pela camada 3 (`RootList`). `RC-01`.

## Camada 1 — Search: filtro desligável, Tab delegado, reinício

```tsx
const SearchContext = createContext<{
  registerTab: (handler: TabHandler | null) => void;  // a List registra
  tab: TabHandler;                                     // o Input pergunta
  restart: () => void;                                 // remonta o Autocomplete
} | null>(null);

export const CommandPaletteSearch = ({ filter, inputValue, onInputChange, children }) => {
  const match = useCommandMatch();
  const { contains } = useFilter({ sensitivity: "base" });
  const tabHandlerRef = useRef<TabHandler | null>(null);
  const [generation, setGeneration] = useState(0);
  const itemFilter = filter === undefined ? match : filter;   // undefined = padrão; null = desligado

  const autocompleteFilter = itemFilter
    ? (textValue, query, node) =>
        isCommandItem(node.value) ? itemFilter(node.value, query) : contains(textValue, query)
    : undefined;

  // Trocar de escopo troca a lista inteira, e o Autocomplete guarda o id do
  // item focado na lista velha: sem remontar, a primeira seta não responde.  (armadilha C)
  return (
    <Autocomplete key={generation} filter={autocompleteFilter}
                  inputValue={inputValue} onInputChange={onInputChange}>
      <SearchContext.Provider value={{
        registerTab: (handler) => { tabHandlerRef.current = handler; },
        tab: (id) => tabHandlerRef.current?.(id) ?? false,
        restart: () => setGeneration((value) => value + 1),
      }}>
        {children}
      </SearchContext.Provider>
    </Autocomplete>
  );
};
```

Nota de implementação: o handler de Tab vai num `ref` e é exposto por **funções** no
contexto (`registerTab`/`tab`), não como o próprio ref — o lint de React Hooks proíbe
escrever em `.current` de um valor vindo de `useContext`.

## Camada 1 — Input: o item destacado, Tab e Backspace

```tsx
/**
 * O item destacado na lista, não o `aria-activedescendant`: ao digitar, o
 * Autocomplete atrasa o atributo em 500 ms por causa do leitor de tela.   (armadilha B)
 */
function focusedItemKey(input: HTMLInputElement): string | undefined {
  const listId = input.getAttribute("aria-controls");
  const list = listId ? document.getElementById(listId) : null;
  return list?.querySelector<HTMLElement>("[data-focused]")?.dataset.key;
}

const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
  onKeyDown?.(event);

  if (event.key === "Backspace" && event.currentTarget.value === "" && path.length > 0) {
    onPopPath?.();
    search?.restart();
    return;
  }

  if (event.key !== "Tab" || event.shiftKey) return;
  const key = focusedItemKey(event.currentTarget);
  // O FocusScope do diálogo move o foco num listener de `document`; sem o
  // stopPropagation o Tab aceito ainda pula para o próximo focável.          (armadilha A)
  if (key !== undefined && search?.tab(key)) {
    event.preventDefault();
    event.stopPropagation();
    search.restart();
  }
};
```

O `SearchField` recebe `aria-label={path.length > 0 ? `Search in ${path.join(" / ")}` :
"Search"}` e `autoFocus` (é o que devolve o foco ao campo depois do reinício). Os chips
do caminho são `<span>` antes do `<Input>`, com `truncate` e `min-w-24` no input.

## Camada 1 — List: registra o Tab, fecha depois da ação

```tsx
export const CommandPaletteList = forwardRef(({ items, onAction, onTab, emptyState, defaultSection, isLoading }, ref) => {
  const root = useContext(RootContext);
  const registerTab = useContext(SearchContext)?.registerTab;

  useEffect(() => {
    registerTab?.(onTab ?? null);
    return () => registerTab?.(null);
  }, [registerTab, onTab]);

  return (
    <Menu ref={ref} items={groupBySection(items, defaultSection)}
          onAction={(key) => { onAction(String(key)); root?.close(); }}
          aria-busy={isLoading || undefined}
          renderEmptyState={() => <div …>{emptyState}</div>}>
      {(section) => (
        <MenuSection id={section.id} items={section.items}>
          <Header>{section.label}</Header>
          <Collection items={section.items}>{(item) => <CommandRow item={item} />}</Collection>
        </MenuSection>
      )}
    </Menu>
  );
});

const CommandRow = ({ item }) => (
  <MenuItem id={item.id} value={item} textValue={item.label} …>   {/* value={item}: sem isso o filtro não vê keywords (armadilha 4) */}
    {({ isFocused }) => (
      <>
        {/* ícone, label, description */}
        {item.hint && isFocused && <Kbd aria-hidden="true">{item.hint}</Kbd>}
      </>
    )}
  </MenuItem>
);
```

## Camada 1 — Root

```tsx
export const CommandPaletteRoot = forwardRef(({ isOpen, onOpenChange, children, className, ...rest }, ref) => (
  <ModalOverlay isOpen={isOpen} onOpenChange={onOpenChange} isDismissable className={OVERLAY_CLASSES}>
    <AriaModal className="w-full max-w-xl …">
      <Dialog ref={ref} aria-label="Command palette" className={twMerge(PANEL_CLASSES, className)} {...rest}>
        <RootContext.Provider value={{ close: () => onOpenChange(false) }}>
          {children}                                   {/* desmonta ao fechar → estado do filho zera (RC-05) */}
        </RootContext.Provider>
      </Dialog>
    </AriaModal>
  </ModalOverlay>
));
```

## Camada 2 — `defineScope` e os escopos

```ts
export interface CommandResult extends CommandItem { href: string; scope?: CommandScope; }

export interface CommandScope {
  id: string; label: string; noun: string; placeholder: string; grant: Grant;
  useResults: (name: string, options: { enabled: boolean; limit: number }) => ScopeResults;
}

/**
 * O tipo da listagem fica preso na closure: quem consome o escopo vê só
 * `CommandResult`, e a query é a mesma factory da tela — mesmo cache.
 */
function defineScope<T, TKey extends QueryKey>({ query, toResult, ...scope }: ScopeDefinition<T, TKey>): CommandScope {
  return {
    ...scope,
    useResults: (name, { enabled, limit }) => {
      const { data, isLoading, error } = useQuery({
        ...query({ name: name || undefined, limit }),
        enabled,
        placeholderData: keepPreviousData,      // só dentro do mesmo escopo (a lista remonta por key)
        select: (page) => page.data.map(toResult),
      });
      return { results: data ?? [], isLoading, error };
    },
  };
}

export const appsScope = defineScope({
  id: "apps", label: "App Clients", noun: "App Clients",
  placeholder: "Search App Clients by name…", grant: "as::apps::READ",
  query: (params) => appListQuery(params),                 // a query da tela de listagem
  toResult: (app) => ({ id: `app:${app.id}`, label: app.name, description: formatShortId(app.id),
                        icon: AppWindow, section: "App Clients", href: `/console/apps/${app.id}` }),
});

function poolUsersScope(pool: { id: string; name: string }): CommandScope {   // segundo nível
  return defineScope({
    id: `pool-users:${pool.id}`, label: pool.name, noun: "users",
    placeholder: `Search users in ${pool.name} by name…`, grant: "as::users::READ",
    query: (params) => userListQuery({ ...params, pool_id: pool.id }),
    toResult: (user) => ({ id: `user:${user.id}`, label: user.name, description: user.email,
                           icon: UserRound, section: "Users",
                           href: `/console/pools/${pool.id}?selected-user=${user.id}` }),   // destino sem rota
  });
}

export const poolsScope = defineScope({
  id: "pools", label: "User Pools", noun: "User Pools",
  placeholder: "Search User Pools by name…", grant: "as::users_pool::READ",
  query: (params) => userPoolListQuery(params),
  toResult: (pool) => ({ id: `pool:${pool.id}`, label: pool.name,
                         description: pool.description || `${formatCount(pool.users_count)} users`,
                         icon: Users, section: "User Pools", href: `/console/pools/${pool.id}`,
                         scope: poolUsersScope(pool) }),                                   // onde Tab entra
});
```

Na navegação:

```ts
export interface ConsoleNavItem {
  label: string; description: string; to: string; icon: LucideIcon; grants: Grant[]; keywords?: string[];
  /** Tab na paleta busca dentro deste item. Ausente: a listagem não filtra por nome. */
  search?: CommandScope;
}
// … { label: "App Clients", to: "/console/apps", grants: ["as::apps::READ"], keywords: ["apps", "applications", "clients", "keys"], search: appsScope }
// … { label: "Organizations", … }   ← sem `search`: o backend não filtra por nome
```

## Camada 3 — estado e as duas listas

```tsx
const ConsolePaletteBody = ({ onOpenSettings }) => {
  const [path, setPath] = useState<CommandScope[]>([]);
  const [text, setText] = useState("");

  const query = text.trim();
  const debounced = useDebouncedValue(query);           // 300 ms
  // Entrar num escopo zera o texto; sem este atalho o escopo novo receberia,
  // pelo tempo do debounce, o texto que era do anterior.                      (RC-03)
  const name = query === "" ? "" : debounced;
  const scope = path.at(-1);

  const enterScope: EnterScope = (scopes) => { setPath((current) => [...current, ...scopes]); setText(""); };

  return (
    <>
      <CommandPaletteSearch filter={null} inputValue={text} onInputChange={setText}>   {/* filtro desligado: a camada 3 filtra */}
        <CommandPaletteInput path={path.map((s) => s.label)}
                             onPopPath={() => setPath((current) => current.slice(0, -1))}
                             placeholder={scope?.placeholder ?? "Search pages, App Clients and User Pools…"} />
        {scope
          ? <ScopeList key={scope.id} scope={scope} query={query} name={name} onEnterScope={enterScope} />   /* key: RC-04 */
          : <RootList query={query} name={name} onEnterScope={enterScope} onOpenSettings={onOpenSettings} />}
      </CommandPaletteSearch>
      <CommandPaletteFooter>
        <CommandPaletteHint keys={["↑", "↓"]}>to navigate</CommandPaletteHint>
        <CommandPaletteHint keys={["↵"]}>to open</CommandPaletteHint>
        <CommandPaletteHint keys={["Tab"]}>to search inside</CommandPaletteHint>
        {path.length > 0 && <CommandPaletteHint keys={["⌫"]}>to go back</CommandPaletteHint>}
        <CommandPaletteHint keys={["esc"]}>to close</CommandPaletteHint>
      </CommandPaletteFooter>
    </>
  );
};

const ConsoleCommandPalette = ({ isOpen, onOpenChange, onOpenSettings }) => (
  <CommandPaletteRoot isOpen={isOpen} onOpenChange={onOpenChange}>
    <ConsolePaletteBody onOpenSettings={onOpenSettings} />     {/* estado dentro do Root */}
  </CommandPaletteRoot>
);
```

```tsx
const RootList = ({ query, name, onEnterScope, onOpenSettings }) => {
  const navigate = useNavigate();
  const { can, canAny } = useCan();
  const match = useCommandMatch();                               // a MESMA função do primitivo
  const canEnter = (scope: CommandScope) => can(scope.grant);

  const navItems = CONSOLE_NAV.filter((item) => item.grants.length === 0 || canAny(item.grants));
  const local: CommandItem[] = [
    ...navItems.map((item) => ({ id: item.to, label: item.label, description: item.description, icon: item.icon,
                                 keywords: item.keywords, section: "Pages",
                                 hint: item.search && canEnter(item.search) ? "Tab" : undefined })),
    ...SETTINGS_ITEMS,
  ];

  const visible = query ? local.filter((item) => match(item, query)) : local;               // texto imediato
  const missesLocally = name !== "" && !local.some((item) => match(item, name));            // texto assentado

  // Número fixo de hooks: a lista de escopos é estática, a grant só liga ou desliga.
  const appSearch  = appsScope.useResults(name,  { enabled: missesLocally && canEnter(appsScope),  limit: 5 });
  const poolSearch = poolsScope.useResults(name, { enabled: missesLocally && canEnter(poolsScope), limit: 5 });
  const searches = ([[appsScope, appSearch], [poolsScope, poolSearch]] as const).filter(([origin]) => canEnter(origin));

  const fallback = visible.length === 0 && query
    ? searches.flatMap(([origin, search]) => search.results.map((result) => ({ result, origin })))
    : [];
  const fallbackById = new Map(fallback.map((entry) => [entry.result.id, entry]));
  const fallbackError = searches.find(([, search]) => search.error)?.[1].error;

  const emptyState =
    searches.length === 0 ? "Nothing matches that. Try a page name."
    : fallbackError ? describeCoreError(fallbackError)
    : query !== name || searches.some(([, search]) => search.isLoading) ? <Searching />
    : `Nothing matches “${query}” in pages, App Clients or User Pools.`;

  return (
    <CommandPaletteList
      items={visible.length > 0 ? visible : fallback.map(({ result }) => withTabHint(result, canEnter))}
      emptyState={emptyState}
      isLoading={searches.some(([, search]) => search.isLoading)}
      onAction={(id) => {
        if (id.startsWith(SETTINGS_PREFIX)) { /* abre o modal de settings */ return; }
        navigate(fallbackById.get(id)?.result.href ?? id);
      }}
      onTab={(id) => {
        const page = navItems.find((item) => item.to === id);
        if (page?.search && canEnter(page.search)) { onEnterScope([page.search]); return true; }
        const entry = fallbackById.get(id);
        if (entry?.result.scope && canEnter(entry.result.scope)) { onEnterScope([entry.origin, entry.result.scope]); return true; }  // caminho completo
        return false;
      }}
    />
  );
};
```

```tsx
const ScopeList = ({ scope, query, name, onEnterScope }) => {
  const navigate = useNavigate();
  const { can } = useCan();
  const canEnter = (next: CommandScope) => can(next.grant);
  const { results, isLoading, error } = scope.useResults(name, { enabled: true, limit: 10 });
  const byId = new Map(results.map((result) => [result.id, result]));

  const emptyState = error ? describeCoreError(error)
    : isLoading || query !== name ? <Searching />
    : query ? `No ${scope.noun} match “${query}”.` : `No ${scope.noun} yet.`;

  return (
    <CommandPaletteList items={results.map((r) => withTabHint(r, canEnter))} isLoading={isLoading} emptyState={emptyState}
      onAction={(id) => { const r = byId.get(id); if (r) navigate(r.href); }}
      onTab={(id) => { const next = byId.get(id)?.scope; if (!next || !canEnter(next)) return false; onEnterScope([next]); return true; }} />
  );
};
```

## O atalho (`ConsoleLayout`)

```tsx
useEffect(() => {
  const onKeyDown = (event: KeyboardEvent) => {
    if (!(event.metaKey || event.ctrlKey)) return;
    const key = event.key.toLowerCase();
    if (key === "k") { event.preventDefault(); setIsSearchOpen((open) => !open); }
  };
  window.addEventListener("keydown", onKeyDown);
  return () => window.removeEventListener("keydown", onKeyDown);
}, []);
```

## O destino sem rota (`PoolUsersSection`)

```tsx
const SELECTED_USER_PARAM = "selected-user";
const [searchParams, setSearchParams] = useSearchParams();
const openUserId = Number(searchParams.get(SELECTED_USER_PARAM)) || null;      // a URL é a fonte

const setOpenUserId = (userId: number | null) =>
  setSearchParams((params) => {
    if (userId === null) params.delete(SELECTED_USER_PARAM); else params.set(SELECTED_USER_PARAM, String(userId));
    return params;
  }, { replace: true });

// … <button onClick={() => setOpenUserId(user.id)}>  …  <UserDetailModal userId={openUserId} onClose={() => setOpenUserId(null)} />
```

## O que foi medido para dar por pronto

Com Chrome headless (Playwright) sobre o Storybook e sobre o app com a API interceptada:

- "custo" digitado com 60 ms entre teclas → exatamente `[/core/apps?name=custo&limit=5,
  /core/users_pool?name=custo&limit=5]`.
- Tab na pool achada pela reserva → caminho `["User Pools", "Customers"]`, request
  `/core/users?limit=10&pool_id=…` **sem** `name`.
- "ali" → `/core/users?name=ali&limit=10&pool_id=…`; Enter → URL
  `/console/pools/pool-cust?selected-user=7` com o modal aberto; Esc limpa o parâmetro.
- Reabrir → raiz, campo vazio. Tab em App Clients → 10 itens, `/core/apps?limit=10`.
- Tab logo após digitar, primeira seta após Tab, Tab em item sem escopo mantendo o foco
  no campo, palavra-chave "keys": todos verdes.
- Não verificado: temas a olho; leitor de tela.
