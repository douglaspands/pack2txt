# OpenSpec SPEC-002: Solid TAR Streaming em Memória & Filtros

**Document ID:** `SPEC-002`  
**Status:** ✅ **CONCLUÍDO / IMPLEMENTADO**  
**Versão:** 1.0.0  
**Módulos de Código:** [`internal/archive/tar.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/archive/tar.go), [`internal/archive/filter.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/archive/filter.go)  
**Módulos de Teste:** [`internal/archive/tar_test.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/archive/tar_test.go)  

---

## 1. Escopo e Propósito

Esta especificação define o arquivamento Solid TAR (POSIX USTAR/PAX) 100% em memória RAM, regras de normalização de caminhos multiplataforma e filtros de exclusão automática para arquivos de desenvolvimento desnecessários.

---

## 2. Requisitos de Arquivamento em Memória

1. **Zero-Disk Streaming:** O fluxo TAR deve ser gerado diretamente em `bytes.Buffer` sem tocar em arquivos temporários em disco.
2. **Normalização de Caminhos:** Todos os caminhos nos cabeçalhos TAR devem:
   - Utilizar a barra normal (`/`) como separador canônico.
   - Ser relativos à raiz empacotada (sem `./` inicial ou barras absolutas).
3. **Preservação de Metadados:**
   - Permissões Unix (`0755` para diretórios e binários executáveis; `0644` para arquivos regulares).
   - Timestamps de modificação (`ModTime`).
   - Tamanho exato em bytes.

---

## 3. Filtros Padrão de Exclusão (Default Filter)

Por padrão, a CLI ignora automaticamente as seguintes pastas e padrões:

| Categoria | Pastas / Padrões Ignorados |
|---|---|
| **Controle de Versão** | `.git/` |
| **Ambientes Virtuais & Deps** | `node_modules/`, `.venv/`, `venv/`, `vendor/` |
| **Caches & Compilados** | `__pycache__/`, `.pytest_cache/`, `dist/`, `build/`, `out/`, `target/`, `.next/`, `.turbo/` |
| **Metadados do SO** | `.DS_Store`, `Thumbs.db` |
| **Binários Temporários** | `*.pyc`, `*.o`, `*.a`, `*.exe` |

*Flag de Sobrescrita:* A flag `--no-ignore` desativa todos os filtros caso o usuário deseje incluir arquivos ocultos e dependências.

---

## 4. Critérios de Aceitação (Gherkin Scenarios)

```gherkin
Feature: Memory Solid TAR Creation and Filtering

  Scenario: Ignorar node_modules e .git por padrão
    Given um diretório contendo arquivos de código, pasta "node_modules" e pasta ".git"
    When CreateTar é invocado com o filtro padrão ativo
    Then o arquivo TAR gerado não deve conter entradas de "node_modules" nem ".git"
    And deve conter todos os arquivos de código-fonte válidos

  Scenario: Inclusão total com --no-ignore
    Given um diretório com pastas ocultas
    When CreateTar é invocado com o filtro desativado (noIgnore = true)
    Then todas as pastas e arquivos devem ser incluídos no TAR
```
