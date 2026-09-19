# Handoff para a `openapi-builder`: correções no gerador de schema

Destino: `github.com/LuigiVanin/openapi-builder`, checado em
`~/repos/swagger-builder`, no commit `e6e8946`. Origem: a documentação OpenAPI do
`auth-service` está errada em seis pontos, todos no mesmo lugar.

**Não toquei na biblioteca.** Tudo abaixo foi verificado rodando a versão atual
(`v0.2.1`) contra as entidades reais do `auth-service`, e a correção proposta foi
reimplementada e testada à parte antes de virar este documento. O que **não** foi
verificado é o patch compilando dentro do repositório da lib — ele não foi aplicado
lá.

---

## Por que isso importa agora

O frontend está montando o cliente a partir desse swagger. Hoje ele descreve nomes
de campo que não existem, tipos errados em data e em jsonb, e expõe no schema um
campo que o backend marcou como oculto.

---

## Os seis defeitos

Tudo em `openapi/utils.go`, nas funções `TypeToSchema` (linhas 65-83) e
`TypeToParam` (linhas 100-118).

### 1. `,omitempty` entra no nome da propriedade

`field.Tag.Get("json")` devolve a tag inteira, e ela vira a chave do schema sem
passar por nenhum corte.

```jsonc
// hoje
"organization_id,omitempty": { "type": "string" },
"tracking,omitempty":        { "type": "array", "items": { "type": "string" } }
```

Atinge `TypeToSchema` (linha 69) e `TypeToParam` (linha 103).

### 2. `json:"-"` vira uma propriedade chamada `-`

O campo não é pulado. Como todos os campos ocultos de um struct viram a mesma
chave `"-"`, eles colapsam num único item.

```jsonc
// entity.User, hoje
"-": { "type": "string" }
```

Em `entity.User` isso são cinco campos — `Uuid`, `UsersPoolId`, `PasswordHash`,
`UsersPool` e `Tracking`. O `Tracking` é o mais grave: ele é `json:"-"` justamente
para o histórico de ip não vazar nas respostas, e **aparece no schema**.

### 3. Campos não exportados entram no schema

Não há checagem de `field.IsExported()`. É o que produz o `created_at` estranho que
o consumidor vê:

```jsonc
// hoje — os campos internos de time.Time
"created_at": { "type": "object", "properties": {
  "ext": { "type": "integer" }, "loc": { "type": "string" }, "wall": { "type": "string" }
}}
```

### 4. Todo campo ponteiro é documentado como `string`

`compositeTypes` (linha 46) tem `Struct`, `Array` e `Slice`, mas não `Pointer`.
Então a linha 79 não recompõe, e sobra o `default` do `TypeToSwagger`, que é
`string`.

```jsonc
// hoje — Organization é *Organization
"organization,omitempty":     { "type": "string" },
"default_profile,omitempty":  { "type": "string" },
"owner_user,omitempty":       { "type": "string" }
```

Toda relação anulável do `auth-service` está documentada como string.

### 5. `time.Time` não tem tratamento próprio

Consequência do 3, mas merece caso explícito: o certo é
`{"type": "string", "format": "date-time"}`.

### 6. `json.RawMessage` e `[]byte` viram array de string

`RawMessage` é `[]byte`, então cai no ramo de slice.

```jsonc
// hoje
"metadata": { "type": "array", "items": { "type": "string" } }
```

O certo é `object` para `RawMessage`, e `{"type": "string", "format": "byte"}` para
`[]byte` cru, que o `encoding/json` serializa em base64.

---

## ⚠️ A armadilha: corrigir o defeito 4 sozinho trava o boot

**Assim que ponteiro passar a ser recomposto, `TypeToSchema` entra em recursão
infinita.** `entity.App` tem `ParentApp *App` — auto-referência — e hoje a lib só
escapa por acidente, porque não desce em ponteiro.

Verifiquei: com a correção do defeito 4 e sem proteção de ciclo, a recursão passa
de trinta níveis e não para.

Isso não seria um teste vermelho, seria **stack overflow no boot** do
`auth-service`: o documento é construído num `fx.Invoke`, antes do servidor subir.

Há ainda um ciclo mútuo — `UsersPool.OwnerUser *User` e `User.UsersPool *UsersPool`
— que hoje se resolve sozinho porque o segundo é `json:"-"`, mas que volta a
existir no dia em que alguém tirar essa tag.

**A proteção de ciclo é obrigatória e faz parte do mesmo patch.**

---

## A correção

Substitui as duas funções. Adiciona `encoding/json` e `time` aos imports.

```go
var (
	timeType       = reflect.TypeOf(time.Time{})
	rawMessageType = reflect.TypeOf(json.RawMessage{})
)

// jsonFieldName resolve o nome do jeito que o encoding/json resolve: a tag até a
// primeira vírgula, o nome do campo quando não há tag, e "pula" quando a tag é "-".
func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")

	// `json:"-"` omite o campo; `json:"-,"` nomeia o campo de "-".
	if tag == "-" {
		return "", false
	}

	name, _, _ := strings.Cut(tag, ",")

	if name == "" {
		return field.Name, true
	}

	return name, true
}

func TypeToSchema(t reflect.Type) Schema {
	return typeToSchema(t, map[reflect.Type]bool{})
}

// seen carrega os tipos abertos no caminho atual. Sem ele, um tipo que se
// referencia — App.ParentApp é *App — recursa para sempre agora que ponteiro é
// recomposto.
func typeToSchema(t reflect.Type, seen map[reflect.Type]bool) Schema {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t {
	case timeType:
		return Schema{Type: String, Format: "date-time"}
	case rawMessageType:
		return Schema{Type: Object}
	}

	// []byte sai em base64 no encoding/json, não como lista de números.
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return Schema{Type: String, Format: "byte"}
	}

	schema := Schema{Type: TypeToSwagger(t.Kind())}

	switch t.Kind() {
	case reflect.Array, reflect.Slice:
		schema.Items = typeToSchema(t.Elem(), seen).ToItems()

		return schema

	case reflect.Struct:
		if seen[t] {
			// Já está aberto acima no caminho: corta o ciclo e documenta como
			// objeto sem propriedades.
			return schema
		}

		seen[t] = true
		defer delete(seen, t)

		schema.Properties = map[string]Schema{}

		for index := range t.NumField() {
			field := t.Field(index)

			if !field.IsExported() {
				continue
			}

			name, ok := jsonFieldName(field)

			if !ok {
				continue
			}

			schema.Properties[name] = typeToSchema(field.Type, seen)
		}

		return schema

	default:
		return schema
	}
}

func TypeToParam(t reflect.Type) []Parameter {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return []Parameter{}
	}

	parameters := []Parameter{}

	for index := range t.NumField() {
		field := t.Field(index)

		if !field.IsExported() {
			continue
		}

		name, ok := jsonFieldName(field)

		if !ok {
			continue
		}

		parameters = append(parameters, Parameter{
			Name:     name,
			Required: true,
			Schema:   Schema{Type: TypeToSwagger(field.Type.Kind())},
		})
	}

	return parameters
}
```

Duas notas sobre o que mudou de forma:

- `Properties` passa a ser preenchido **só para struct**. Antes era sempre um mapa
  vazio; como a tag é `json:"properties,omitempty"`, um schema escalar deixa de
  emitir `"properties": {}`, o que é mais correto.
- `defer delete(seen, t)` faz a proteção ser **por caminho**, e não global: o mesmo
  tipo pode aparecer em dois ramos irmãos, o que é legítimo.

---

## Os testes existentes não quebram

Li os dez casos de `tests/utils._test.go`. Nenhum depende do que muda:
`TestTypeToSchemaPrimitive_Success` usa `assert.Empty`, que passa em mapa nil;
`TestTypeToParam_Success` afirma `names["Jump"]`, que é o fallback para o nome Go
de um campo sem tag, preservado. A suíte passa hoje, com 34 casos.

---

## Testes a acrescentar

Um por defeito, mais o ciclo. Sugestão de fixtures:

```go
type Tagged struct {
	Name     string          `json:"name"`
	Optional string          `json:"optional,omitempty"`
	Hidden   string          `json:"-"`
	Dash     string          `json:"-,"`
	NoTag    string
	internal string
	When     time.Time       `json:"when"`
	Raw      json.RawMessage `json:"raw"`
	Bytes    []byte          `json:"bytes"`
	Child    *Tagged         `json:"child"`
}
```

| Teste | Afirma |
| --- | --- |
| `TestTagWithOptionIsTrimmed` | existe `optional`, não existe `optional,omitempty` |
| `TestDashTagIsSkipped` | não existe a chave `-`, e `hidden` não aparece de nenhuma forma |
| `TestDashCommaIsAFieldNamedDash` | existe a chave `-` (o caso `json:"-,"`) |
| `TestUntaggedFieldKeepsGoName` | existe `NoTag` |
| `TestUnexportedFieldIsSkipped` | não existe `internal` |
| `TestTimeIsDateTimeString` | `when` é `{type: string, format: date-time}` e **não** tem `wall`/`ext`/`loc` |
| `TestRawMessageIsObject` | `raw` é `{type: object}` |
| `TestByteSliceIsBase64String` | `bytes` é `{type: string, format: byte}` |
| `TestPointerIsRecomposed` | `child` é `object`, não `string` |
| `TestSelfReferenceDoesNotRecurse` | o teste **termina** — sem a proteção ele estoura a pilha |
| `TestMutualReferenceDoesNotRecurse` | dois tipos apontando um para o outro, idem |

O de auto-referência é o que mais importa: sem a proteção ele não falha, ele mata o
processo de teste.

---

## Como verificar contra o `auth-service`

Depois de publicar a versão, no `auth-service`:

```bash
go get github.com/LuigiVanin/openapi-builder@<versão>
go build ./... && go test ./...
make dev
```

E então, em `/docs/`, conferir o schema de `UsersPool`. O esperado, que validei com
a lógica corrigida rodando contra a entidade real:

```jsonc
{
  "id": "string",
  "name": "string",
  "description": "string",
  "public_key": "string",          // sem o ,omitempty
  "default_profile_id": "string",
  "default_profile": { "type": "object", … },   // era "string"
  "users_count": 0,
  "metadata": { "type": "object" },             // era array de string
  "tracking": { "type": "object" },             // era "tracking,omitempty"
  "organization_id": "string",
  "owner_user": { "type": "object", … },
  "created_at": "string (date-time)",           // era {ext, loc, wall}
  "updated_at": "string (date-time)"
}
```

E em `User`, a chave `-` some — junto com ela, `tracking`, que volta a não ser
documentado, que é o correto: ele só sai por `GET /core/users/me` e
`GET /core/users/{id}`, por um DTO próprio.

`auth_service` tem um teste que constrói o documento inteiro
(`tests/modules/routes/route_order_test.go`, `TestTheOpenApiDocumentBuildsWithTheUpdatePayloads`).
Ele já cobre a construção não entrar em pânico, então a recursão infinita seria
pega ali — mas só depois do bump da dependência.

---

## Fora de escopo, se quiser abrir issue

- **`$ref` e `components/schemas`.** Hoje todo struct é inlinado, então
  `entity.App` aparece por extenso dentro de `Session`, dentro de `UsersPool`, e
  assim por diante. O documento fica enorme e repetido. É a correção estrutural de
  verdade; a proteção de ciclo acima é o remendo que a torna desnecessária para não
  quebrar.
- **`map[string]T`.** Vira `object` sem `additionalProperties`, e o `Schema` não
  tem esse campo hoje.
- **`required`.** `TypeToParam` marca tudo como `Required: true`. Um campo com
  `,omitempty` é, por definição, o contrário disso — dá para derivar.
- **Enum.** `login_types`, `token_type` e os códigos de erro do `auth-service` têm
  conjunto fechado e poderiam sair como `enum`.
