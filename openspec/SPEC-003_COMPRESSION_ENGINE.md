# OpenSpec SPEC-003: Motor de Compressão Máxima & Auto-Optimizer

**Document ID:** `SPEC-003`  
**Status:** ✅ **CONCLUÍDO / IMPLEMENTADO**  
**Versão:** 1.0.0  
**Módulos de Código:** [`internal/compressor/`](file:///home/douglas/Workspace/gemini/pack2txt/internal/compressor/)  
**Módulos de Teste:** [`internal/compressor/compressor_test.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/compressor/compressor_test.go)  

---

## 1. Escopo e Propósito

Esta especificação formaliza a interface uniforme de compressores, os parâmetros de compressão máxima para Brotli, Zstandard e Gzip, e o algoritmo do **Auto-Optimizer Concorrente**.

---

## 2. Especificação dos Algoritmos

### 2.1 Brotli (`brotli` - Padrão)
* **Qualidade:** `11` (`brotli.BestCompression`).
* **Sliding Window:** `LGWin: 22` (janela de dicionário deslizante de 4 MB).
* **Meta de Eficiência:** Proporcionar economia de **> 75% a 99%** em código-fonte repetitivo.

### 2.2 Zstandard (`zstd`)
* **Nível:** `zstd.SpeedBestCompression`.
* **Concorrência:** Gerenciamento via `sync.Pool` para zero alocações redundantes de encoders/decoders.

### 2.3 Gzip (`gzip`)
* **Nível:** `gzip.BestCompression` (Level 9).

### 2.4 None (`none`)
* **Operação:** Pass-through puro (identidade `src == dst`), ideal para arquivos pré-comprimidos.

### 2.5 Auto-Optimizer (`auto`)
* **Comportamento:** Executa `brotli`, `zstd` e `gzip` simultaneamente em goroutines concorrentes sobre o buffer TAR.
* **Seleção:** Avalia `min(len(compressedBytes))` e registra no envelope o compressor vencedor.

---

## 3. Critérios de Aceitação (Gherkin Scenarios)

```gherkin
Feature: Compression Engine and Auto Optimizer

  Scenario: Round-trip com Brotli Quality 11
    Given um buffer de código-fonte de 40 KB
    When o compressor "brotli" é executado
    Then o tamanho comprimido deve ser menor que 1 KB (> 95% de economia)
    And o descompressor deve restaurar exatamente os 40 KB originais

  Scenario: Avaliação automática com o modo auto
    Given um buffer de dados genéricos
    When o modo "auto" é executado
    Then o compressor que produzir o menor payload em bytes deve ser escolhido
    And a descompressão deve ocorrer de forma transparente
```
