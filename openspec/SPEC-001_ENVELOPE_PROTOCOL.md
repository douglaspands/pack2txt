# OpenSpec SPEC-001: Protocolo de Envelope & Gramática

**Document ID:** `SPEC-001`  
**Status:** ✅ **CONCLUÍDO / IMPLEMENTADO**  
**Versão:** 1.0.0  
**Módulos de Código:** [`internal/packer/envelope.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/packer/envelope.go)  
**Módulos de Teste:** [`internal/packer/packer_test.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/packer/packer_test.go)  

---

## 1. Escopo e Propósito

Esta especificação formaliza a estrutura sintática, os identificadores mágicos, o versionamento e a máquina de parsing do envelope textual utilizado pela ferramenta `pack2txt`.

---

## 2. Gramática Canônica ABNF

```abnf
envelope       = magic ":" version ":" compressor ":" encoder ":" payload
magic          = "PACK2TXT"
version        = "v1"
compressor     = "brotli" / "zstd" / "gzip" / "none" / "auto"
encoder        = "b32768" / "b91" / "b85" / "b64"
payload        = 1*(safe-char)
safe-char      = %x20-7E / %x3400-4DBF / %x4E00-9FFF / %xA000-D7AF
```

---

## 3. Comportamento do Parser e Heurística de Fallback

### 3.1 Entrada Padronizada
Quando a entrada inicia com o prefixo `PACK2TXT:v1:`, o parser divide a string exatamente no 5º separador de dois-pontos (`:`), extraindo:
1. `magic` (deve ser idêntico a `PACK2TXT`)
2. `version` (deve ser compatível com `v1`)
3. `compressor` (normalizado para lowercase)
4. `encoder` (normalizado para lowercase)
5. `payload` (dados codificados)

### 3.2 Heurística de Fallback (Entrada Sem Cabeçalho)
Se a entrada for uma string crua (ex: colada diretamente de um chat sem cabeçalho):
1. **Verificação Unicode:** Se a string contiver caracteres da faixa BMP (> 127), assume-se `b32768`.
2. **Verificação Base64:** Se a string for estritamente ASCII e contiver apenas `[A-Za-z0-9+/=]`, assume-se `b64`.
3. **Verificação Base91 / Base85:** Se contiver pontuações ASCII adicionais (`"`, `#`, `$`), assume-se `b91` ou `b85`.
4. **Descompressão Automática:** O payload decodificado é passado para o motor de autodetecção de magic bytes (`zstd`, `gzip`, `brotli`, `tar`).

---

## 4. Critérios de Aceitação (Gherkin Scenarios)

```gherkin
Feature: OpenSpec Envelope Serialization & Parsing

  Scenario: Empacotamento padrão com envelope
    Given um conjunto de arquivos em disco
    When o comando pack é executado com compressor "brotli" e encoder "b32768"
    Then o texto gerado deve iniciar com "PACK2TXT:v1:brotli:b32768:"
    And o payload deve conter apenas caracteres Unicode válidos da especificação

  Scenario: Parsing de envelope com sucesso
    Given a string "PACK2TXT:v1:zstd:b64:SGVsbG8="
    When ParseEnvelope é invocado
    Then a versão deve ser "v1"
    And o compressor deve ser "zstd"
    And o encoder deve ser "b64"

  Scenario: Fallback para payload sem envelope
    Given uma string codificada em Base32768 sem cabeçalho
    When ParseEnvelope é invocado
    Then o parser deve autodetectar o encoder como "b32768"
    And deve prosseguir com a descompressão automática com sucesso
```
