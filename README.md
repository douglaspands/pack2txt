# pack2txt 📦✨

> **Text-Based Solid Archive Tool com Compressão Máxima e Codificação de Alta Densidade (Base32768) para LLMs e Agentes de IA.**

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![OpenSpec](https://img.shields.io/badge/OpenSpec-v1-green.svg)](docs/openspec/PACK2TXT_SPEC_v1.md)

---

## 🎯 Por que o `pack2txt`?

Ao transferir repositórios de código ou estruturas de pastas para chats de Inteligência Artificial (Google Gemini, Claude, ChatGPT) ou pipelines de automação, o desenvolvedor costuma enfrentar **limites rígidos de caracteres e desperdício de tokens de contexto**.

O **`pack2txt`** resolve esse problema ao combinar:
1. **Solid TAR 100% em Memória:** Serializa múltiplos arquivos em um único fluxo contínuo (aproveitando similaridades entre arquivos).
2. **Compressão Extrema:** **Brotli Q11** (janela de 4 MB) e **Zstandard** atingem taxas de compressão superiores a 80-99% em código-fonte.
3. **Codificação Base32768 (Default):** Mapeia 15 bits por caractere Unicode BMP, reduzindo a contagem de caracteres em **mais de 50%** em relação ao Base64 convencional.
4. **Proteção Anti-Zip Slip:** Validação rigorosa de segurança impedindo qualquer vazamento de caminho (`../`) na extração.
5. **Composabilidade Unix:** Suporte nativo a `stdin`, `stdout` e pipes (`| pbcopy`, `cat archive.txt | pack2txt unpack`).

---

## 📊 Comparativo de Eficiência Textual

| Codificador | Bits / Caractere | Contagem de Caracteres (10 KB de dados) | Economia vs Base64 |
|---|---|---|---|
| **Base64** | 6 bits | ~13.336 caracteres | 0% (Linha de base) |
| **Base85** | 6.4 bits | ~12.500 caracteres | ~6.3% |
| **Base91** | ~6.7 bits | ~12.298 caracteres | ~7.8% |
| **Base32768** ⭐ | **15 bits** | **~5.334 caracteres** | **~60.0% de economia!** |

---

## 🚀 Instalação

### Via Go
```bash
go install github.com/douglas/pack2txt/cmd/pack2txt@latest
```

### Compilando do Código-Fonte
```bash
git clone https://github.com/douglas/pack2txt.git
cd pack2txt
go build -o pack2txt ./cmd/pack2txt
```

---

## 💻 Guia de Uso

### 1. Empacotar uma Pasta (`pack`)
Por padrão, utiliza **Brotli Q11** e **Base32768**, ignorando pastas de build/dependências (`.git`, `node_modules`, `__pycache__`, `.venv`, etc.):

```bash
# Salvar em arquivo .txt
pack2txt pack ./meu_projeto -o projeto.txt

# Atalho conveniente (assume 'pack' automaticamente)
pack2txt ./meu_projeto -o projeto.txt

# Modo Auto-Optimizer (escolhe o compressor que gerar o menor payload)
pack2txt pack ./meu_projeto -c auto -o projeto.txt
```

### 2. Inspecionar o Pacote sem Extrair (`inspect`)
Veja a lista de arquivos, tamanhos originais, datas e taxa de compressão em memória:

```bash
pack2txt inspect projeto.txt
```

### 3. Descompactar e Restaurar os Arquivos (`unpack`)
Reconstrói os arquivos originais com 100% de integridade e fidelidade de permissões:

```bash
# Extrair na pasta atual
pack2txt unpack projeto.txt

# Extrair em pasta específica (cria o diretório se necessário)
pack2txt unpack projeto.txt -d ./restaurado

# Forçar sobrescrita se os arquivos já existirem
pack2txt unpack projeto.txt -d ./restaurado --force
```

---

## 🔄 Pipelines Unix & Clipboard

O `pack2txt` se integra perfeitamente a scripts e atalhos do sistema:

### Copiar projeto direto para a Área de Transferência (macOS / Linux)
```bash
# macOS
pack2txt pack ./src --stdout | pbcopy

# Linux (X11)
pack2txt pack ./src --stdout | xclip -selection clipboard

# Windows (PowerShell)
pack2txt pack ./src --stdout | Set-Clipboard
```

### Restaurar colando da Área de Transferência
```bash
# macOS
pbpaste | pack2txt unpack - -d ./destino

# Linux
xclip -selection clipboard -o | pack2txt unpack - -d ./destino
```

---

## 📜 Especificação do Envelope OpenSpec (v1)

O texto gerado segue o contrato canônico:
```text
PACK2TXT:v1:<compressor>:<encoder>:<payload>
```
* **Exemplo:** `PACK2TXT:v1:brotli:b32768:一丁丂七丄丅丆万丈三上下不与...`
* **Compressores Suportados:** `brotli` (padrão), `zstd`, `gzip`, `none`, `auto`.
* **Codificadores Suportados:** `b32768` (padrão), `b91`, `b85`, `b64`.

> **Heurística de Fallback:** Se você colar uma string codificada pura sem o cabeçalho `PACK2TXT:`, o `unpack` detecta automaticamente o charset e descomprime os dados.

---

## 🛡️ Segurança e Proteção Anti-Zip Slip

O descompactador implementa sanitização rigorosa de caminhos:
- Rejeita qualquer entrada contendo sequências de path traversal (`../../`).
- Normaliza caminhos absolutos e barras do Windows/POSIX.
- Mascara bits de permissão perigosos (`setuid`/`setgid`).

---

## 🧪 Testes e Benchmarks

Execute toda a suíte de testes com detecção de concorrência (*race detector*):
```bash
go test -v -race ./...
```

Execute os benchmarks de alocação de memória e throughput:
```bash
go test -bench=. -benchmem ./...
```

---

## 📄 Licença

Distribuído sob a licença MIT. Consulte `LICENSE` para mais detalhes.
