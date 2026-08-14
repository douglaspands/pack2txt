# OpenSpec SPEC-006: Interface CLI, Pipes Unix & Experiência de Terminal

**Document ID:** `SPEC-006`  
**Status:** ✅ **CONCLUÍDO / IMPLEMENTADO**  
**Versão:** 1.0.0  
**Módulos de Código:** [`cmd/pack2txt/main.go`](file:///home/douglas/Workspace/gemini/pack2txt/cmd/pack2txt/main.go), [`internal/ui/terminal.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/ui/terminal.go)  
**Módulos de Teste:** [`internal/ui/terminal_test.go`](file:///home/douglas/Workspace/gemini/pack2txt/internal/ui/terminal_test.go)  

---

## 1. Escopo e Propósito

Esta especificação define os comandos, flags, regras de composição com pipes Unix (`stdout`/`stdin`) e a renderização visual do terminal.

---

## 2. Comandos da CLI

### 2.1 `pack <origem> [flags]`
- **Atalho:** `pack2txt <caminho>` assume o comando `pack`.
- **Flags:**
  - `-o, --output <arquivo>`: Grava em arquivo `.txt`.
  - `-c, --compressor <nome>`: Escolhe o compressor (`brotli`, `zstd`, `gzip`, `none`, `auto`). Padrão: `brotli`.
  - `-e, --encoder <nome>`: Escolhe o codificador (`b32768`, `b91`, `b85`, `b64`). Padrão: `b32768`.
  - `--stdout`: Emite apenas o envelope limpo para `stdout`.
  - `--no-ignore`: Desativa filtros de exclusão padrão.

### 2.2 `unpack [arquivo.txt|-] [flags]`
- **Flags:**
  - `-d, --dest <pasta>`: Diretório de destino (padrão: `.`).
  - `-f, --force`: Sobrescreve arquivos existentes sem erro.
  - `-q, --quiet`: Desativa a saída visual.

### 2.3 `inspect [arquivo.txt|-]`
- Lê o envelope e lista em memória a árvore de arquivos, permissões, tamanhos e taxa de economia sem tocar no disco.

---

## 3. Separação de Streams para Pipes Unix

- **Modo Pipe Ativo:** Quando a saída padrão for redirecionada (`| pbcopy`, `> out.txt`) ou `--stdout` for especificado, a CLI emite **apenas** o payload puro em `os.Stdout`.
- **Mensagens de Log/UI:** Todas as tabelas, decorações e mensagens de erro são enviadas para `os.Stderr` ou suprimidas.

---

## 4. Critérios de Aceitação (Gherkin Scenarios)

```gherkin
Feature: CLI Interface and Unix Pipes

  Scenario: Encadear pack e unpack via Pipe Unix
    Given um diretório com arquivos de código
    When o comando "pack2txt pack ./origem --stdout | pack2txt unpack - -d ./destino" é executado
    Then os arquivos devem ser restaurados em "./destino" sem erros
    And a saída do pipe não deve conter caracteres ANSI de formatação

  Scenario: Atalho de diretório
    Given um diretório existente "meu_projeto"
    When "pack2txt meu_projeto -o out.txt" é executado
    Then o comando deve assumir a operação "pack" e gerar "out.txt"
```
