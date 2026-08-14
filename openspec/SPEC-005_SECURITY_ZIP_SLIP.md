# OpenSpec SPEC-005: Segurança Rigorosa & Defesa Anti-Zip Slip

**Document ID:** `SPEC-005`  
**Status:** ✅ **CONCLUÍDO / IMPLEMENTADO**  
**Versão:** 1.0.0  
**Módulos de Código:** [`internal/archive/security.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/archive/security.go)  
**Módulos de Teste:** [`internal/archive/tar_test.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/archive/tar_test.go)  

---

## 1. Escopo e Propósito

Esta especificação formaliza as defesas de segurança contra vulnerabilidades de *Path Traversal* / *Zip Slip*, sanitização de permissões de arquivo e isolamento do diretório de destino.

---

## 2. Regras de Sanitização de Caminhos

1. **Prevenção de Fuga do Diretório:**
   - O caminho de extração deve ser resolvido via `filepath.Clean(filepath.Join(destDir, entryPath))`.
   - Deve ser verificado se o caminho resultante é estritamente prefixado pelo diretório canônico de destino.
2. **Rejeição de Caminhos Maliciosos:**
   - Se `entryPath` contiver `../`, caminhos absolutos (`/etc/passwd`), ou letras de unidade do Windows (`C:\`), o descompactador deve abortar imediatamente com erro `ErrZipSlipViolation`.
3. **Máscara de Permissões Perigosas:**
   - Permissões especiais como `setuid` (`04000`), `setgid` (`02000`) e `sticky` (`01000`) são obrigatoriamente mascaradas (`mode & 0777`).

---

## 3. Critérios de Aceitação (Gherkin Scenarios)

```gherkin
Feature: Zip Slip Protection & Security Sanitization

  Scenario: Tentativa de extração com Zip Slip
    Given um arquivo TAR malicioso contendo uma entrada "../../evil.txt"
    When ExtractTar é invocado para um diretório de destino "/tmp/safe"
    Then a operação deve falhar imediatamente com ErrZipSlipViolation
    And nenhum arquivo deve ser gravado fora de "/tmp/safe"

  Scenario: Sanitização de permissões de arquivo
    Given uma entrada TAR com permissão setuid (04755)
    When o arquivo é extraído em disco
    Then a permissão final deve ser sanitizada para 0755
```
