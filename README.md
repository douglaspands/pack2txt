# pack2txt 📦✨

> **Text-Based Solid Archive Tool com Compressão Máxima e Codificação de Alta Densidade (Base32768) para LLMs, Agentes de IA e Pipelines Unix.**

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![OpenSpec](https://img.shields.io/badge/OpenSpec-v1-green.svg)](openspec/INDEX.md)
[![Antigravity Ready](https://img.shields.io/badge/Antigravity-Ready-purple.svg)](AGENTS.md)

---

## 📑 Sumário

- [🎯 O que é o pack2txt?](#-o-que-é-o-pack2txt)
- [🧠 Otimização Extrema de Contexto para LLMs & Tokens](#-otimização-extrema-de-contexto-para-llms--tokens)
- [🚀 Instalação](#-instalação)
- [📖 Guia Completo de Comandos](#-guia-completo-de-comandos)
  - [1. pack (Empacotar)](#1-pack2txt-pack-origem)
  - [2. unpack (Descompactar)](#2-pack2txt-unpack-arquivotxt-)
  - [3. inspect (Inspecionar em Memória)](#3-pack2txt-inspect-arquivotxt-)
  - [4. version (Versão)](#4-pack2txt-version)
  - [5. Atalho Inteligente](#5-atalho-inteligente)
- [⚙️ Detalhamento Completo de Flags & Parâmetros](#️-detalhamento-completo-de-flags--parâmetros)
- [🗜️ Motores de Compressão (Brotli, Zstd, Gzip, Auto)](#️-motores-de-compressão)
- [🔡 Codificadores Textuais (Base32768, Base91, Base85, Base64)](#-codificadores-textuais)
- [🔄 Workflows Práticos & Pipes Unix](#-workflows-práticos--pipes-unix)
- [🤖 Integração com Antigravity CLI & Agentes de IA](#-integração-com-antigravity-cli--agentes-de-ia)
- [🛡️ Segurança & Proteção Anti-Zip Slip](#️-segurança--proteção-anti-zip-slip)
- [📜 Especificação OpenSpec (SDD)](#-especificação-openspec-sdd)
- [🧪 Testes & Benchmarks](#-testes--benchmarks)

---

## 🎯 O que é o pack2txt?

Ao transferir repositórios de código, árvores de pastas ou conjuntos de arquivos para janelas de contexto de **Inteligência Artificial (Google Gemini, Claude, ChatGPT, Antigravity CLI)**, o desenvolvedor costuma esbarrar em dois problemas:
1. **Limites rígidos de caracteres** em interfaces de chat e APIs.
2. **Desperdício massivo de tokens** ao colar código bruto ou usar formatos ineficientes como Base64 tradicional (+33% de overhead).

O **`pack2txt`** resolve esse gargalo ao unir:
- **Solid TAR 100% em Memória:** Todos os arquivos são serializados em um único fluxo contínuo na memória RAM (`bytes.Buffer`), permitindo que compressores aproveitem redundâncias e padrões repetidos entre múltiplos arquivos.
- **Compressão Máxima (Brotli Q11 & Zstandard):** Redução de **> 75% a 99%** no tamanho de bases de código.
- **Codificação Base32768 (Default):** Mapeia 15 bits por caractere Unicode BMP, gerando **~60% menos caracteres** do que Base64.
- **Zero Arquivos Temporários:** Nenhum arquivo temporário é criado no disco (`/tmp`), garantindo velocidade instantânea (0ms de latência de disco) e segurança total.

---

## 🧠 Otimização Extrema de Contexto para LLMs & Tokens

### Análise Matemática de Densidade Textual

| Codificador | Bits / Caractere | Contagem de Caracteres (Amostra 10 KB) | Economia vs Base64 |
|---|---|---|---|
| **Base64** | 6.0 bits | ~13.336 caracteres | 0% (Linha de base) |
| **Base85** | 6.4 bits | ~12.500 caracteres | ~6.3% |
| **Base91** | ~6.7 bits | ~12.299 caracteres | ~7.8% |
| **Base32768** ⭐ | **15.0 bits** | **~5.334 caracteres** | **~60.0% de economia!** |

### Exemplo Real em Código-Fonte (Go + Docs ~ 60 KB)
- **Tamanho Original Bruto:** 57.650 bytes
- **Comprimido (Brotli Q11):** 13.190 bytes (**77,13% de economia**)
- **Codificado em Base64:** ~17.586 caracteres (~4.400 tokens)
- **Codificado em Base32768:** **7.202 caracteres (~1.800 tokens)**

> **Resultado:** O `pack2txt` permite enviar uma base de código inteira consumindo uma fração minúscula da janela de contexto da IA.

---

## 🚀 Instalação

### Instalação via Go (Requer Go 1.22+)
```bash
go install github.com/douglas/pack2txt/cmd/pack2txt@latest
```

### Compilação a partir do Código-Fonte
```bash
git clone https://github.com/douglas/pack2txt.git
cd pack2txt
go build -ldflags="-s -w" -o pack2txt ./cmd/pack2txt
```

---

## 📖 Guia Completo de Comandos

### 1. `pack2txt pack <origem>`
Empacota uma pasta inteira ou arquivo individual em um envelope textual padronizado.

```bash
# Empacotamento padrão (Brotli Q11 + Base32768) salvando em arquivo
pack2txt pack ./meu_projeto -o projeto.txt

# Empacotamento direto na tela
pack2txt pack ./meu_projeto

# Empacotamento com Zstandard e Base91
pack2txt pack ./meu_projeto -c zstd -e b91 -o projeto.txt

# Empacotamento com modo Auto-Optimizer (escolhe o menor binário)
pack2txt pack ./meu_projeto -c auto -o projeto.txt

# Incluir arquivos ocultos e dependências (desativa filtros padrão)
pack2txt pack ./meu_projeto --no-ignore -o completo.txt
```

---

### 2. `pack2txt unpack [arquivo.txt|-]`
Restaura os arquivos originais a partir de um envelope de texto com validação anti-Zip Slip.

```bash
# Extrair arquivo na pasta atual
pack2txt unpack projeto.txt

# Extrair em pasta de destino específica (cria o diretório se não existir)
pack2txt unpack projeto.txt -d ./restaurado

# Forçar sobrescrita de arquivos existentes
pack2txt unpack projeto.txt -d ./restaurado --force

# Modo silencioso (ideal para scripts de automação)
pack2txt unpack projeto.txt -d ./restaurado --quiet

# Extrair a partir da entrada padrão (stdin / pipe)
cat projeto.txt | pack2txt unpack - -d ./restaurado
```

---

### 3. `pack2txt inspect [arquivo.txt|-]`
Inspeciona em memória a árvore de arquivos, permissões, tamanhos originais e taxa de compressão **sem gravar nada no disco**.

```bash
# Inspecionar arquivo salvo
pack2txt inspect projeto.txt

# Inspecionar via pipe
cat projeto.txt | pack2txt inspect -
```

---

### 4. `pack2txt version`
Exibe a versão instalada, compressores registrados e codificadores suportados.

```bash
pack2txt version
```

---

### 5. Atalho Inteligente
Se o primeiro argumento for um diretório ou arquivo existente no disco, o `pack2txt` assume automaticamente o subcomando `pack`:

```bash
# Executa 'pack2txt pack ./meu_projeto -o out.txt'
pack2txt ./meu_projeto -o out.txt
```

---

## ⚙️ Detalhamento Completo de Flags & Parâmetros

### Flags do Comando `pack`
| Flag | Alias | Tipo | Padrão | Descrição e Significado |
|---|---|---|---|---|
| `--output` | `-o` | `string` | `""` | Caminho do arquivo `.txt` onde o envelope será gravado. Se omitido, o envelope é exibido no terminal. |
| `--compressor` | `-c` | `string` | `"brotli"` | Algoritmo de compressão: `brotli`, `zstd`, `gzip`, `none` ou `auto`. |
| `--encoder` | `-e` | `string` | `"b32768"` | Codificador binário-para-texto: `b32768`, `b91`, `b85` ou `b64`. |
| `--stdout` | | `bool` | `false` | Emite **apenas** a string limpa do envelope para `stdout`, suprimindo caixas decorativas (essencial para pipes Unix). |
| `--no-ignore` | | `bool` | `false` | Desativa o filtro padrão de exclusões, incluindo pastas como `.git`, `node_modules`, `.venv`, etc. |

### Flags do Comando `unpack`
| Flag | Alias | Tipo | Padrão | Descrição e Significado |
|---|---|---|---|---|
| `--dest` | `-d` | `string` | `"."` | Diretório de destino onde os arquivos e pastas serão reconstruídos. |
| `--force` | `-f` | `bool` | `false` | Permite sobrescrever arquivos no destino caso já existam. |
| `--quiet` | `-q` | `bool` | `false` | Suprime saídas de cabeçalhos e tabelas de sucesso, mantendo apenas erros. |

---

## 🗜️ Motores de Compressão

| Compressor | Identificador | Nível | Características |
|---|---|---|---|
| **Brotli** ⭐ | `brotli` | Quality 11, LGWin 22 (4 MB) | **Padrão:** Máxima compressão para textos e código-fonte, atingindo as menores saídas. |
| **Zstandard** | `zstd` | `SpeedBestCompression` | Altíssima taxa de compressão com descompressão ultra-rápida. |
| **Gzip** | `gzip` | `gzip.BestCompression` (Level 9) | Compatibilidade ampla com ferramentas padrão Unix. |
| **None** | `none` | 0 (Pass-through) | Pass-through sem compressão (ideal para pacotes de imagens/vídeos). |
| **Auto Optimizer** | `auto` | Concorrente em memória | Executa `brotli`, `zstd` e `gzip` simultaneamente e seleciona o compressor que gerar o menor número de bytes. |

---

## 🔡 Codificadores Textuais

| Encoder | Identificador | Densidade de Bits | Descrição e Uso Recomendado |
|---|---|---|---|
| **Base32768** ⭐ | `b32768` | **15 bits / caractere** | **Padrão:** Mapeia 15 bits em caracteres seguros Unicode BMP (CJK). Otimizado para chats de IA e economia de tokens. |
| **Base91** | `b91` | ~13.5 bits / 2 chars | Compactação avançada utilizando 91 caracteres ASCII imprimíveis. |
| **Base85** | `b85` | 4 bytes -> 5 chars | Padrão clássico Ascii85 (RFC 1924 / Adobe). |
| **Base64** | `b64` | 6 bits / caractere | Padrão clássico RFC 4648 com compatibilidade universal. |

---

## 🔄 Workflows Práticos & Pipes Unix

### 1. Injetar Repositório Diretamente no Prompt de uma IA
```bash
# Copiar repositório direto para o clipboard (macOS)
pack2txt pack ./src --stdout | pbcopy

# Copiar repositório direto para o clipboard (Linux X11)
pack2txt pack ./src --stdout | xclip -selection clipboard

# Copiar repositório direto para o clipboard (Windows PowerShell)
pack2txt pack ./src --stdout | Set-Clipboard
```
*No chat da IA, basta colar o texto gerado acompanhado da instrução:*
> *"Aqui está a base de código empacotada com `pack2txt`. Use o comando `pack2txt unpack` para restaurá-la."*

### 2. Restaurar Diretamente do Clipboard
```bash
# macOS
pbpaste | pack2txt unpack - -d ./projeto_restaurado

# Linux
xclip -selection clipboard -o | pack2txt unpack - -d ./projeto_restaurado
```

### 3. Pipeline de Backup / Transferência Remota
```bash
# Empacotar localmente e descompactar em servidor remoto via SSH
pack2txt pack ./app --stdout | ssh usuario@servidor "pack2txt unpack - -d /var/www/app"
```

---

## 🤖 Integração com Antigravity CLI & Agentes de IA

O projeto inclui suporte nativo e regras de contexto para o **Antigravity CLI**:
- **[`AGENTS.md`](file:///home/douglas/Workspace/gemini/pack2txt/AGENTS.md):** Diretrizes do projeto para agentes de IA.
- **[`.agents/rules/token-efficiency.md`](file:///home/douglas/Workspace/gemini/pack2txt/.agents/rules/token-efficiency.md):** Regras de eficiência de tokens carregadas automaticamente.
- **[`.agents/skills/pack2txt-context/SKILL.md`](file:///home/douglas/Workspace/gemini/pack2txt/.agents/skills/pack2txt-context/SKILL.md):** Skill para empacotar, inspecionar e descompactar código em alta densidade.

---

## 🛡️ Segurança & Proteção Anti-Zip Slip

O descompactador implementa validações estritas de segurança em todas as operações de extração:
1. **Bloqueio de Path Traversal:** Rejeita qualquer tentativa de escrita fora do diretório de destino (`../../`, `..\`).
2. **Normalização de Caminhos:** Remove barras absolutas (`/etc/passwd`) e letras de unidade (`C:\`).
3. **Máscara de Permissões:** Mascara bits perigosos como `setuid` e `setgid` (`mode & 0777`).

---

## 📜 Especificação OpenSpec (SDD)

O projeto foi desenvolvido sob a metodologia **Spec-Driven Development**. Todas as especificações técnicas estão documentadas na pasta [`openspec/`](openspec/):

- **[INDEX.md](openspec/INDEX.md):** Matriz de Rastreabilidade e conformidade.
- **[SPEC-001](openspec/SPEC-001_ENVELOPE_PROTOCOL.md):** Protocolo de Envelope & Gramática ABNF.
- **[SPEC-002](openspec/SPEC-002_SOLID_TAR_STREAMING.md):** Solid TAR Streaming em Memória & Filtros.
- **[SPEC-003](openspec/SPEC-003_COMPRESSION_ENGINE.md):** Motores de Compressão & Auto-Optimizer.
- **[SPEC-004](openspec/SPEC-004_TEXT_ENCODERS.md):** Codificadores Textuais de Alta Densidade.
- **[SPEC-005](openspec/SPEC-005_SECURITY_ZIP_SLIP.md):** Segurança Rigorosa & Defesa Anti-Zip Slip.
- **[SPEC-006](openspec/SPEC-006_CLI_PIPES_UX.md):** Interface CLI, Pipes Unix & Terminal UX.
- **[SPEC-007](openspec/SPEC-007_CI_CD_DISTRIBUTION.md):** CI/CD Multi-Plataforma & Distribuição.

---

## 🧪 Testes & Benchmarks

Execute a suíte de testes unitários com o detector de concorrência (*race detector*):
```bash
go test -v -race ./...
```

Execute os benchmarks de alocação de memória e throughput:
```bash
go test -bench=. -benchmem ./...
```

---

## 📄 Licença

Distribuído sob a licença **MIT**. Consulte [`LICENSE`](LICENSE) para mais detalhes.
