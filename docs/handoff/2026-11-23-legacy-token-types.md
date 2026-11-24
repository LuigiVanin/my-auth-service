# Handoff para o frontend: `FAST_JWT` e `SESSION_UUID` viram legados

Destino: `auth-service-frontend`, checado em `./frontend`. Escrito em 2026-11-23, junto com
a feature do frontend `docs/features/23-11-2026-console-users-and-detail-fixes/`.

**Uma restrição de escrita, nenhum campo novo.** O frontend sobe na mesma janela.

## O que muda

- `POST /core/apps` e `PUT /core/apps/{id}` aceitam só `"token_type": "JWT"`. `FAST_JWT` e
  `SESSION_UUID` respondem `400 BAD_REQUEST`, com
  `data.fields[] = {"field": "TokenType", "tag": "oneof", "param": "JWT"}`.
- Leitura não muda: um app gravado antes continua voltando com o tipo que tem, e o
  `authorize` continua emitindo e validando `SESSION_UUID` para ele.
- O enum `token_type` do Postgres continua com os três valores — nenhum dado é migrado.

## O que o frontend faz

- Nenhum seletor, filtro ou rótulo oferece os dois tipos.
- Um app que ainda tenha um deles mostra o valor com o selo "Legacy"; a edição oferece só
  `JWT`, e trocar para `JWT` é a única escrita possível no campo.

## O que fica como está

`FAST_JWT` já não tinha `case` no `authorize` — um app com esse tipo não consegue gerar token
hoje. Não foi corrigido: o tipo é legado e não volta a ser escrito.
