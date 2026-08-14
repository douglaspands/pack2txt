---
description: Regras de eficiência de contexto, uso do pack2txt e otimização de tokens para agentes de IA
always_on: true
---

# Regras de Eficiência de Tokens & Contexto

1. **Compactação de Código para Prompt:**
   - Sempre que for necessário empacotar estruturas de código para injetar em contextos de LLM ou prompts de chat, utilize:
     `pack2txt pack <pasta> --stdout`
   - O formato padrão `PACK2TXT:v1:brotli:b32768:...` garante o menor número possível de caracteres e tokens.

2. **Inspeção sem Efeitos Colaterais:**
   - Para verificar o conteúdo de um arquivo `.txt` gerado pelo `pack2txt`, use `pack2txt inspect <arquivo.txt>` para visualizar a tabela de arquivos e métricas em memória.

3. **Segurança Obrigatória:**
   - Nunca desative as verificações de Zip Slip e Path Traversal ao modificar ou estender os módulos de extração.
