# Prompt para o agente

Cole a pasta `command-palette/` no repositório de destino (por exemplo em
`docs/playbooks/`) e envie a mensagem abaixo ao agente, ajustando os colchetes.

---

Quero implementar neste projeto a paleta de comandos descrita em
`docs/playbooks/command-palette/`. Leia o `README.md` e os arquivos `01` a `10` **antes**
de propor qualquer coisa; `referencia/` é exemplo de outra stack, use só como apoio.

Siga o protocolo de `09-caminhos-de-adocao.md` §9.1, nesta ordem, e **pare nos dois
pontos marcados com ⏸** para eu responder:

1. Inventário do projeto e da API (preencha a tabela de `02` §2.6 com os recursos
   reais daqui).
2. As sete lacunas de `01` — responda por escrito o que o código permite inferir, e me
   pergunte o resto.
3. Plano por nível (1, 2, 3), com os arquivos a criar e alterar e o mapeamento das peças
   de `03` para a nossa stack (tabela de `09` §9.4). Espere minha aprovação.
4. Implemente nível por nível, rodando os cenários de `08` ao fim de cada um, e me mostre
   o resultado medido (contagem de requests, foco, caminho), não só "passou".

Restrições que não se negociam: uma única função de casamento; nenhuma URL de API
montada dentro da paleta — reaproveite a camada de dados das listagens; `Tab` só
consumido quando a lista devolve `true`; o item do `Tab` lido da marca de destaque na
lista, não de `aria-activedescendant`; estado da paleta dentro do diálogo.

Contexto do projeto: [stack, biblioteca de componentes, cliente de dados, onde fica a
navegação, onde fica a permissão, idioma da UI, recursos que devem ser pesquisáveis
primeiro].
