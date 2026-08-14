# OpenSpec SPEC-004: Codificadores de Texto de Alta Densidade

**Document ID:** `SPEC-004`  
**Status:** ✅ **CONCLUÍDO / IMPLEMENTADO**  
**Versão:** 1.0.0  
**Módulos de Código:** [`internal/encoder/`](file:///home/douglas/Workspace/gemini/pack2txt/internal/encoder/)  
**Módulos de Teste:** [`internal/encoder/encoder_test.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/encoder/encoder_test.go)  

---

## 1. Escopo e Propósito

Esta especificação define os contratos de codificação binário-para-texto suportados, com foco na densidade máxima de caracteres e tokens para modelos de linguagem (LLMs).

---

## 2. Especificação Técnica dos Codificadores

### 2.1 Base32768 (`b32768` - Padrão)
* **Densidade de Informação:** 15 bits por caractere Unicode BMP (7 bits para byte restante).
* **Faixas Unicode Seguras:**
  - CJK Unified Ideographs (`0x4E00..0x9DFF`)
  - Yi Syllables / Hangul (`0xA000..0xCFFF`)
  - Extensão A para remanescentes de 7 bits (`0x3400..0x347F`)
* **Eficiência Comprovada:** Reduz em **60% a contagem de caracteres** em relação ao Base64 convencional.

### 2.2 Base91 (`b91`)
* **Densidade de Informação:** ~13 a 14 bits por 2 caracteres ASCII.
* **Alfabeto:** 91 caracteres ASCII imprimíveis:
  `ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%&()*+,./:;<=>?@[]^_`{|}~"`

### 2.3 Base85 (`b85`)
* **Padrão:** Ascii85 (RFC 1924 / Adobe Standard).
* **Densidade:** 4 bytes para 5 caracteres ASCII.

### 2.4 Base64 (`b64`)
* **Padrão:** RFC 4648 com suporte a padding opcional e URLs.

---

## 3. Critérios de Aceitação (Gherkin Scenarios)

```gherkin
Feature: High-Density Binary-to-Text Encoders

  Scenario: Round-trip de todos os encoders com dados binários arbitrários
    Given um buffer de 64 KB de bytes aleatórios
    When cada encoder ("b32768", "b91", "b85", "b64") codifica e decodifica o buffer
    Then o resultado decodificado deve ser 100% idêntico ao buffer original

  Scenario: Validação de densidade do Base32768
    Given uma amostra binária de 10 KB
    When a amostra é codificada em Base64 e em Base32768
    Then a contagem de caracteres do Base32768 deve ser menor que 50% da contagem de caracteres do Base64
```
