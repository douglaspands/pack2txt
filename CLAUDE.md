# Instruções para o Claude Code

Este arquivo é carregado automaticamente pelo Claude Code neste repositório. Ele aponta para as fontes de verdade já existentes no projeto — não duplique o conteúdo delas aqui.

## Leitura obrigatória antes de codificar

1. **[`AGENTS.md`](AGENTS.md)** — regras gerais de eficiência de contexto/tokens para todos os agentes de IA (Claude, antigravity-cli/Gemini e outros).
2. **[`.agents/rules/token-efficiency.md`](.agents/rules/token-efficiency.md)** — regra `always_on`, aplicada automaticamente pelo harness.
3. **[`openspec/SPEC_LIFECYCLE.md`](openspec/SPEC_LIFECYCLE.md)** — processo obrigatório de Spec-Driven Development.

## Regra mandatória: Spec-Driven Development

**Toda nova funcionalidade, encoder, compressor ou comando NOVO neste repositório DEVE seguir o ciclo de `openspec/SPEC_LIFECYCLE.md` antes de qualquer código de produção ser escrito:** `/plan` → rascunho `openspec/SPEC-XXX_NOME.md` → aprovação explícita do usuário → implementação TDD → atualização de `openspec/INDEX.md`. Não pule esse gate mesmo que a tarefa pareça pequena — é uma exigência do projeto, não uma sugestão.

## Uso do próprio `pack2txt` para contexto

Ao compartilhar árvores de arquivo com o usuário ou entre agentes, prefira `pack2txt pack <pasta> --stdout` (texto, Base32768+Brotli) em vez de dumps soltos de código. Para transporte por **imagem** (colar em apps de chat que só aceitam imagem), use `pack2txt pack <pasta> --image [--camera-safe] -o arquivo.png` — ver `openspec/SPEC-008_IMAGE_CODEC.md` para o formato e as limitações conhecidas (perfil `camera-safe` ainda não corrige perspectiva real de câmera). `unpack`/`inspect` detectam o formato automaticamente, sem flag adicional.
