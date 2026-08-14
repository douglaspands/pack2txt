# OpenSpec SPEC-007: CI/CD Multi-Plataforma & Distribuição

**Document ID:** `SPEC-007`  
**Status:** ✅ **CONCLUÍDO / IMPLEMENTADO**  
**Versão:** 1.0.0  
**Módulos de Configuração:** [`.github/workflows/release.yml`](file:///home/douglas/Workspace/gemini/pack2txt/.github/workflows/release.yml)  

---

## 1. Escopo e Propósito

Esta especificação define o pipeline de integração contínua (CI), testes automatizados com detecção de concorrência (*race detector*) e geração de binários estáticos multi-plataforma para distribuição no GitHub Releases.

---

## 2. Matriz de Compilação Multi-Plataforma

O workflow compila binários estáticos independentes (`CGO_ENABLED=0`) com stripping de símbolos de depuração (`-s -w`):

| Sistema Operacional | Arquitetura | Nome do Artefato |
|---|---|---|
| **Linux** | `amd64` (x86_64) | `pack2txt-linux-amd64` |
| **Linux** | `arm64` (aarch64) | `pack2txt-linux-arm64` |
| **macOS** | `amd64` (Intel) | `pack2txt-darwin-amd64` |
| **macOS** | `arm64` (Apple Silicon M1/M2/M3/M4) | `pack2txt-darwin-arm64` |
| **Windows** | `amd64` (x86_64) | `pack2txt-windows-amd64.exe` |

---

## 3. Gatilhos de Execução

1. **Pull Requests & Push em Branches Principais:** Executa a suíte completa de testes com detecção de concorrência (`go test -v -race ./...`) e validação de cobertura.
2. **Push de Tags de Release (`v*.*.*`):** Executa os testes, compila toda a matriz multi-plataforma e publica a release no GitHub Releases com notas geradas automaticamente.
