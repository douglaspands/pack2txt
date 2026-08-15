# Instruções para antigravity-cli / Gemini

Este arquivo é carregado automaticamente pelo antigravity-cli/Gemini neste repositório. Ele aponta para as fontes de verdade já existentes — não duplique o conteúdo delas aqui.

## Leitura obrigatória antes de codificar

1. **[`AGENTS.md`](AGENTS.md)** — regras gerais de eficiência de contexto/tokens para todos os agentes de IA.
2. **[`.agents/rules/token-efficiency.md`](.agents/rules/token-efficiency.md)** — regra `always_on`.
3. **[`docs/harness/GEMINI_EFFICIENCY_GUIDE.md`](docs/harness/GEMINI_EFFICIENCY_GUIDE.md)** — guia de eficiência de tokens específico para Gemini (densidade de codificação, tokenização de CJK/BMP, checklist de entrega limpa).
4. **[`openspec/SPEC_LIFECYCLE.md`](openspec/SPEC_LIFECYCLE.md)** — processo obrigatório de Spec-Driven Development.

## Regra mandatória: Spec-Driven Development

**Toda nova funcionalidade, encoder, compressor ou comando novo neste repositório DEVE seguir o ciclo de `openspec/SPEC_LIFECYCLE.md` antes de qualquer código de produção ser escrito:** `/plan` → rascunho `openspec/SPEC-XXX_NOME.md` → aprovação explícita do usuário → implementação TDD → atualização de `openspec/INDEX.md`.

## Uso do próprio `pack2txt` para contexto

Para compartilhar árvores de arquivo, prefira `pack2txt pack <pasta> --stdout` (texto, Base32768+Brotli). Para transporte por **imagem** (colar em apps de chat que só aceitam imagem), use `pack2txt pack <pasta> --image [--camera-safe] -o arquivo.png` — ver `openspec/SPEC-008_IMAGE_CODEC.md`. `unpack`/`inspect` detectam o formato automaticamente.
