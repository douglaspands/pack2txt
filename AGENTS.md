# Diretrizes do Antigravity CLI & Eficiência de Tokens 🚀

Este repositório é otimizado para agentes de IA e desenvolvedores que utilizam o **Antigravity CLI (`agy`)**, **Gemini**, **Claude** e outras LLMs.

---

## 🧠 Boas Práticas de Engenharia de Contexto para Agentes

Ao operar neste projeto, todos os agentes de IA devem seguir rigorosamente os princípios de **eficiência máxima de tokens**:

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

### 4. Pipelines Unix Limpos
- Ao criar scripts de automação, use a flag `--stdout` para que logs e decorações de terminal sejam isolados e apenas o envelope limpo seja passado pelo pipe:
  ```bash
  pack2txt pack ./src --stdout | pack2txt unpack - -d ./dest
  ```
