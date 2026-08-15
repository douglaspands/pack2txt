# Diretrizes do Antigravity CLI & Eficiência de Tokens 🚀

Este repositório é otimizado para agentes de IA e desenvolvedores que utilizam o **Antigravity CLI (`agy`)**, **Gemini**, **Claude** e outras LLMs.

---

## 🧠 Boas Práticas de Engenharia de Contexto para Agentes

Ao operar neste projeto, todos os agentes de IA devem seguir rigorosamente os princípios de **eficiência máxima de tokens**:

### 0. Spec-Driven Development é obrigatório para novas funcionalidades
Sempre que uma nova funcionalidade, encoder, compressor ou comando for solicitado, o desenvolvimento DEVE seguir o ciclo de `openspec/SPEC_LIFECYCLE.md` **antes** de qualquer código de produção: `/plan` → rascunho `openspec/SPEC-XXX_NOME.md` → aprovação explícita do usuário → implementação TDD → atualização de `openspec/INDEX.md`. Consulte `openspec/INDEX.md` para a matriz de rastreabilidade completa.

### 1. Transferência de Contexto em Alta Densidade
- **Nunca faça dumps gigantes de código solto:** Ao compartilhar arquivos múltiplos ou árvores de código com o usuário ou entre agentes, utilize o comando `pack2txt` para gerar o envelope em **Base32768 + Brotli Q11**.
- **Economia de 60% em Caracteres:** A codificação padrão `b32768` comprime 15 bits por caractere Unicode BMP, poupando milhares de tokens no contexto do modelo em comparação ao Base64 tradicional.

### 2. Inspeção em Memória Pré-Extração
- Antes de extrair arquivos para o disco, utilize sempre:
  ```bash
  pack2txt inspect arquivo.txt
  ```
- Isso permite analisar a árvore de arquivos, permissões e tamanhos em memória RAM sem operações desnecessárias de I/O em disco.

### 3. Progressive Disclosure (Divulgação Progressiva)
- Não carregue documentações extensas desnecessariamente.
- Consulte a pasta `openspec/` para especificações formais de cada módulo (`SPEC-001` a `SPEC-007`).
- Consulte a pasta `docs/harness/` para diretrizes de performance e otimizações do Gemini.

### 4. Transporte por Imagem (para apps que só aceitam imagem)
- Quando o destino for um app de chat/canal que só aceita imagem colada (não texto), use `pack2txt pack <pasta> --image [--camera-safe] -o pacote.png` em vez do envelope de texto. Ver `openspec/SPEC-008_IMAGE_CODEC.md` para o formato, os dois perfis (`digital`/`camera-safe`) e as limitações conhecidas.
- `unpack`/`inspect` detectam automaticamente PNG/JPEG vs. envelope de texto — nenhuma flag adicional é necessária do lado da decodificação.

### 5. Pipelines Unix Limpos
- Ao criar scripts de automação, use a flag `--stdout` para que logs e decorações de terminal sejam isolados e apenas o envelope limpo seja passado pelo pipe:
  ```bash
  pack2txt pack ./src --stdout | pack2txt unpack - -d ./dest
  ```
