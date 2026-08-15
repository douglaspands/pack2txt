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

4. **Transporte por Imagem (destino só aceita imagem):**
   - Use `pack2txt pack <pasta> --image -o pacote.png` (perfil `digital`, padrão) ou `--camera-safe` (perfil robusto a foto de câmera, imagem maior) quando o canal de destino só aceita imagem colada, não texto.
   - `unpack`/`inspect` detectam PNG/JPEG automaticamente, sem flag extra.
   - Ver `openspec/SPEC-008_IMAGE_CODEC.md` para os dois perfis e limitações conhecidas (correção de perspectiva real de câmera ainda não implementada).

5. **Spec-Driven Development obrigatório:**
   - Nova funcionalidade/encoder/compressor/comando? Siga `openspec/SPEC_LIFECYCLE.md` antes de escrever código de produção.
