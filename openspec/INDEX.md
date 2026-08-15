# OpenSpec Manifest & Traceability Matrix 📜

**Projeto:** `pack2txt`  
**Metodologia:** Spec-Driven Development (SDD)  
**Versão do Protocolo:** OpenSpec v1.0.0  
**Status do Ciclo:** 🔒 **MILESTONE v1.0 CONCLUÍDA E SELADA** — 🟡 SPEC-008 (v1.1) em desenvolvimento  
**Guia para Novas Features:** Consulte [`SPEC_LIFECYCLE.md`](SPEC_LIFECYCLE.md)  

---

## 📑 Índice de Especificações (OpenSpec Suite v1.0)

| ID | Título da Especificação | Módulo Correspondente | Testes de Validação | Status |
|---|---|---|---|---|
| **[SPEC-001](file:///home/douglas/Workspace/gemini/pack2txt/openspec/SPEC-001_ENVELOPE_PROTOCOL.md)** | Protocolo de Envelope & Gramática ABNF | `internal/packer/envelope.go` | `internal/packer/packer_test.go` | 🔒 Concluído |
| **[SPEC-002](file:///home/douglas/Workspace/gemini/pack2txt/openspec/SPEC-002_SOLID_TAR_STREAMING.md)** | Solid TAR Streaming em Memória & Filtros | `internal/archive/tar.go`<br>`internal/archive/filter.go` | `internal/archive/tar_test.go` | 🔒 Concluído |
| **[SPEC-003](file:///home/douglas/Workspace/gemini/pack2txt/openspec/SPEC-003_COMPRESSION_ENGINE.md)** | Motor de Compressão Máxima & Auto-Optimizer | `internal/compressor/` | `internal/compressor/compressor_test.go` | 🔒 Concluído |
| **[SPEC-004](file:///home/douglas/Workspace/gemini/pack2txt/openspec/SPEC-004_TEXT_ENCODERS.md)** | Codificadores de Texto de Alta Densidade | `internal/encoder/` | `internal/encoder/encoder_test.go` | 🔒 Concluído |
| **[SPEC-005](file:///home/douglas/Workspace/gemini/pack2txt/openspec/SPEC-005_SECURITY_ZIP_SLIP.md)** | Segurança Rigorosa & Defesa Anti-Zip Slip | `internal/archive/security.go` | `internal/archive/tar_test.go` | 🔒 Concluído |
| **[SPEC-006](file:///home/douglas/Workspace/gemini/pack2txt/openspec/SPEC-006_CLI_PIPES_UX.md)** | Interface CLI, Pipes Unix & Experiência de Terminal | `cmd/pack2txt/main.go`<br>`internal/ui/` | `internal/ui/terminal_test.go` | 🔒 Concluído |
| **[SPEC-007](file:///home/douglas/Workspace/gemini/pack2txt/openspec/SPEC-007_CI_CD_DISTRIBUTION.md)** | CI/CD Multi-Plataforma & Distribuição | `.github/workflows/release.yml` | GitHub Actions CI Runner | 🔒 Concluído |
| **[SPEC-008](file:///home/douglas/Workspace/gemini/pack2txt/openspec/SPEC-008_IMAGE_CODEC.md)** | Transporte por Imagem & Codec de Módulos Tolerante a Perdas | `internal/imagecodec/`<br>`internal/fec/` | `internal/imagecodec/imagecodec_test.go`<br>`internal/fec/fec_test.go` | 🟡 Em Desenvolvimento |

---

## 🎯 Matriz de Rastreabilidade (Traceability Matrix)

```mermaid
graph TD
    subgraph "Especificações Formais (OpenSpec v1.0)"
        S1[SPEC-001: Envelope]
        S2[SPEC-002: Solid TAR]
        S3[SPEC-003: Compressão]
        S4[SPEC-004: Encoders]
        S5[SPEC-005: Zip Slip]
        S6[SPEC-006: CLI & Pipes]
        S7[SPEC-007: CI/CD]
        S8[SPEC-008: Transporte por Imagem]
    end

    subgraph "Implementação Go"
        P[internal/packer]
        A[internal/archive]
        C[internal/compressor]
        E[internal/encoder]
        U[internal/ui & cmd]
        CI[.github/workflows]
        IC[internal/imagecodec]
        FE[internal/fec]
    end

    subgraph "Suíte de Testes (TDD)"
        T1[packer_test.go]
        T2[tar_test.go]
        T3[compressor_test.go]
        T4[encoder_test.go]
        T5[terminal_test.go]
        T6[imagecodec_test.go]
        T7[fec_test.go]
    end

    S1 --> P --> T1
    S2 --> A --> T2
    S3 --> C --> T3
    S4 --> E --> T4
    S5 --> A --> T2
    S6 --> U --> T5
    S7 --> CI
    S8 --> IC --> T6
    S8 --> FE --> T7
```

---

## 🚀 Como Iniciar uma Nova Feature Futura

Para iniciar qualquer nova funcionalidade a partir desta baseline:
1. Abra um novo ciclo de planejamento com o comando `/plan`.
2. Crie a nova especificação correspondente (`openspec/SPEC-008_...`).
3. Obtenha a aprovação antes da codificação.
4. Desenvolva com TDD garantindo a aprovação de `go test -v -race ./...`.
