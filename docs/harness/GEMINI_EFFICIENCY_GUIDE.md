# Guia de Eficiência de Contexto e Otimizações Gemini para IAs

**Projeto:** `pack2txt`  
**Foco:** Engenharia de Eficiência de Tokens e Avaliação de Artefatos em Agentes de IA  
**Versão:** 1.0.0  

---

## 1. Visão Geral: O Problema do Contexto em LLMs

Ao transmitir repositórios ou múltiplos arquivos de código para modelos de linguagem (como Google Gemini 1.5/2.0/3.x Pro e Flash), o desenvolvedor e os agentes enfrentam dois grandes gargalos:
1. **Consumo de Janela de Contexto (Tokens):** Arquivos de texto bruto ou codificações ineficientes (Base64) desperdiçam dezenas de milhares de tokens desnecessariamente.
2. **Degradação de Raciocínio (Lost in the Middle):** Prompts excessivamente longos aumentam a latência de inferência e os custos de API.

O **`pack2txt`** resolve esses desafios ao combinar **Solid TAR + Compressão Máxima (Brotli Q11) + Codificação de Alta Densidade (Base32768)**.

---

## 2. Análise Comparativa de Densidade de Codificação

### 2.1 Comparação Teórica de Densidade

| Codificador | Bits por Caractere | Overhead Textual | Tamanho em Caracteres (relativo a Base64) |
|---|---|---|---|
| **Base64** | 6 bits | +33.3% | 100% (Linha de base) |
| **Base85** | 6.4 bits | +25.0% | 93.8% |
| **Base91** | ~6.7 bits | ~23.0% | 89.2% |
| **Base32768** | **15.0 bits** | **-46.7%** | **~48.5% (Economia de mais de 51% em caracteres!)** |

### 2.2 Impacto no Tokenizador do Gemini (SentencePiece / BPE)
Os tokenizadores modernos agrupam caracteres Unicode comuns (especialmente da faixa CJK BMP) com alta eficiência de compressão de vocabulário. Ao usar Base32768 combinada com Brotli Q11:
* Um repositório de **500 KB** de código-fonte em Go/TypeScript:
  - Tamanho TAR bruto: ~512.000 bytes
  - Comprimido com Brotli Q11: ~68.000 bytes
  - Codificado em Base64: ~90.667 caracteres (~23.000 tokens)
  - Codificado em Base32768: **~36.267 caracteres (~9.200 tokens)**
* **Resultado:** **Redução de ~60% no consumo de tokens do contexto!**

### 2.3 Transporte por Imagem (canais que só aceitam imagem, não texto)
Quando o destino é um app de chat que só aceita colagem de imagem (não uma caixa de texto), `pack2txt pack --image` (SPEC-008) codifica o payload comprimido como PNG em vez de texto — sem custo de tokens algum para o agente que só precisa *gerar* o arquivo, mas relevante para quem for *ler* a imagem depois (ex.: outro agente com visão). Dois perfis, com trade-off de tamanho de imagem vs. robustez:

| Perfil | Bits/módulo | Overhead RS | Robustez |
|---|---|---|---|
| `digital` (padrão) | 3 (8 níveis de cinza) | ~37,5% | Anexo de arquivo ou colagem com recompressão JPEG |
| `camera-safe` (`--camera-safe`) | 1 (binário) | ~58% | Também tolera rotação/escala/iluminação desigual/ruído — **correção de perspectiva real de câmera ainda não implementada** (ver `openspec/SPEC-008_IMAGE_CODEC.md` §3.4/§7) |

`camera-safe` produz imagens bem maiores (módulos maiores, menor densidade) — recomendado só para payloads pequenos (poucos KB).

---

## 3. Diretrizes de Otimização para Prompts e Agentes de IA

### 3.1 Formato de Entrega Limpo e Determinístico
Para que agentes de IA e humanos processem o payload sem erros de parsing:
1. O envelope `PACK2TXT:v1:<compressor>:<encoder>:<payload>` não utiliza quebras de linha no payload por padrão (linha única contínua).
2. Sem caracteres de escape como aspas duplas desnecessárias ou barras invertidas que possam ser corrompidas por interfaces de chat Markdown.
3. Envelope autocontido: o agente não precisa adivinhar o compressor ou encoder utilizado; a tag inicial já fornece os metadados necessários para descompressão direta.

### 3.2 Instrução Padronizada para Agentes de IA
Ao enviar um pacote para uma IA, a mensagem deve seguir o modelo conciso:

```markdown
Para restaurar a árvore completa de arquivos, utilize a CLI `pack2txt`:

pack2txt unpack << 'EOF' -d ./projeto
PACK2TXT:v1:brotli:b32768:一丁丂七...
EOF
```

---

## 4. Framework de Avaliação de Artefatos do Harness

Para avaliar a qualidade dos artefatos gerados pelo `pack2txt`, o harness utiliza métricas objetivas de eficiência:

### 4.1 Métricas de Avaliação
1. **Taxa de Compressão de Dados ($R_{comp}$):**
   $$\text{Compression Savings} = \left(1 - \frac{\text{Bytes Comprimidos}}{\text{Bytes Originais}}\right) \times 100\%$$
2. **Eficiência Textual ($R_{text}$):**
   $$\text{Text Density} = \frac{\text{Bytes Binários}}{\text{Contagem de Caracteres do Payload}}$$
3. **Eficiência de Tokens Estimada ($T_{est}$):**
   Contagem média de tokens por KB de código-fonte original.
4. **Fidelidade Bit-a-Bit:**
   $$\text{Hash MD5/SHA256}(\text{Origem}) == \text{Hash MD5/SHA256}(\text{Restaurado})$$

### 4.2 Matriz de Testes com Golden Fixtures
O harness de avaliação conterá fixtures de teste padronizadas:
- **`fixture_small_code`:** Projeto Go com 5 arquivos (~25 KB).
- **`fixture_mixed_assets`:** Projeto web com HTML, CSS, JS, JSON e SVG (~350 KB).
- **`fixture_deep_hierarchy`:** Árvore de diretórios com caminhos aninhados em 8 níveis.
- **`fixture_large_text`:** Repositório de documentação Markdown (~2 MB).
- **`fixture_edge_cases`:** Arquivos vazios, arquivos com permissão executável (`0755`), nomes com caracteres especiais Unicode.

---

## 5. Checklist de Avaliação de Artefatos para o Gemini Harness

Antes de aprovar e validar o binário compilado:
- [ ] O comando `pack` gera envelopes válidos com checksum e integridade garantida.
- [ ] O comando `inspect` exibe a lista completa de arquivos e taxa de compressão sem tocar no disco.
- [ ] O comando `unpack` restaura 100% da estrutura de diretórios, conteúdos e permissões.
- [ ] Tentativas de Zip Slip (`../../`) são bloqueadas com erro explícito de segurança.
- [ ] O streaming via `stdout` / `stdin` funciona transparentemente em pipelines Unix sem poluir o fluxo de dados com logs do terminal.
