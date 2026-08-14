---
name: pack2txt-context
description: >-
  Instruções para empacotar, inspecionar e descompactar árvores de código em alta densidade
  usando pack2txt (Base32768 + Brotli Q11), otimizando o consumo de tokens em chats e agentes de IA.
---

# Skill: Gestão de Contexto com `pack2txt`

Esta skill orienta agentes de IA e o Antigravity CLI a utilizarem a ferramenta `pack2txt` para transmitir código com economia extrema de contexto e tokens.

---

## 1. Quando Utilizar

Ative e utilize os fluxos desta skill quando:
- Precisar transferir um repositório ou diretório inteiro para o contexto de uma IA sem estourar limites de caracteres.
- Receber um payload `PACK2TXT:v1:...` e precisar inspecionar ou reconstruir os arquivos originais.
- Automatizar pipelines de exportação/importação de código via terminal.

---

## 2. Comandos Principais

### A. Empacotar Diretório para Transferência de Contexto
```bash
# Empacotamento padrão (Brotli Q11 + Base32768)
pack2txt pack ./caminho_da_pasta -o pacote.txt

# Empacotamento direto via stdout para pipeline
pack2txt pack ./caminho_da_pasta --stdout
```

### B. Inspecionar Pacote em Memória (Sem Tocar no Disco)
```bash
pack2txt inspect pacote.txt
```

### C. Descompactar e Restaurar
```bash
pack2txt unpack pacote.txt -d ./pasta_destino
```
