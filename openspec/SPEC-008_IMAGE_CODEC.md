# OpenSpec SPEC-008: Transporte por Imagem & Codec de Módulos Tolerante a Perdas

**Document ID:** `SPEC-008`
**Status:** 🟡 **EM DESENVOLVIMENTO** (perfil `digital` completo e testado; perfil `camera-safe` completo para rotação/escala/iluminação/ruído/JPEG, mas correção de perspectiva real — inclinação de câmera genuína — ainda não fechada, ver §3.4 e §7)
**Versão:** 1.0.0
**Módulos de Código:** [`internal/imagecodec/`](file:///home/douglas/Workspace/gemini/pack2txt/internal/imagecodec/), [`internal/fec/`](file:///home/douglas/Workspace/gemini/pack2txt/internal/fec/)
**Módulos de Teste:** `internal/imagecodec/imagecodec_test.go`, `internal/fec/fec_test.go`

---

## 1. Escopo e Propósito

Esta especificação formaliza um **transporte binário por imagem** para o `pack2txt`, alternativo ao envelope textual do `SPEC-001`. O payload comprimido (`internal/compressor`, sempre `auto`) é codificado como uma grade de módulos em escala de cinza dentro de um PNG, protegida por Reed-Solomon, de forma a poder ser colada/enviada em qualquer app de chat compatível com a web e reconstruída bit-a-bit do outro lado — cobrindo tanto o caminho **puramente digital** (anexo de arquivo ou colagem com recompressão JPEG) quanto, num perfil dedicado, uma **foto tirada com câmera de celular**.

Este spec **não altera** o envelope textual `PACK2TXT:v1:...` (`SPEC-001`) nem introduz checksum nele — a garantia de integridade por SHA-256 descrita aqui é exclusiva deste novo container binário de imagem.

---

## 2. Gramática Binária do Cabeçalho

O cabeçalho tem tamanho fixo de **64 bytes**, little-endian, e é a primeira estrutura decodificada (protegida por um bloco RS próprio, ver §5.3):

| Campo          | Offset | Tamanho | Descrição                                                              |
|----------------|-------:|--------:|--------------------------------------------------------------------------|
| `Magic`        | 0      | 4       | `"P2IM"` (`0x50 0x32 0x49 0x4D`)                                        |
| `Version`      | 4      | 1       | `0x01`                                                                   |
| `Profile`      | 5      | 1       | `0x00` = `digital`, `0x01` = `camera-safe`                              |
| `CompressorID` | 6      | 1       | `0x00 none` / `0x01 gzip` / `0x02 zstd` / `0x03 brotli` (algoritmo real escolhido pelo `auto`, nunca "auto" em si) |
| `Levels`       | 7      | 1       | Número de níveis de cinza `N` do módulo (`2`, `4`, `8` ou `16`)          |
| `ModulePx`     | 8      | 2       | `uint16`, tamanho do módulo em pixels                                   |
| `GridCols`     | 10     | 4       | `uint32`, colunas de módulos                                            |
| `GridRows`     | 14     | 4       | `uint32`, linhas de módulos                                             |
| `DataShards`   | 18     | 2       | `uint16`, shards de dados por bloco RS do corpo                         |
| `ParityShards` | 20     | 2       | `uint16`, shards de paridade por bloco RS do corpo                      |
| `ShardSize`    | 22     | 2       | `uint16`, bytes por shard                                               |
| `PayloadLen`   | 24     | 8       | `uint64`, tamanho do payload comprimido pré-FEC, em bytes                |
| `SHA256`       | 32     | 32      | SHA-256 do payload comprimido pré-FEC                                   |

`0x50` (magic PNG/JPEG binário) não colide com `0x89` (PNG) nem `0xFF` (JPEG) nem com o prefixo texto `PACK2TXT:` (`0x50` também, mas o parser de texto já falha o match de string completa `"PACK2TXT:"` antes de tentar o caminho binário — a ordem de checagem em `internal/packer/detect.go` é: (1) magic PNG, (2) magic JPEG, (3) prefixo texto exato, nessa ordem, então não há ambiguidade).

---

## 3. Perfis de Codificação

Dois perfis, selecionados em tempo de codificação via `EncodeOptions.Profile` (CLI: flag `--camera-safe`, ausente = `digital`). O decodificador não precisa saber o perfil de antemão — ele lê `Profile`/`Levels`/`ModulePx` do cabeçalho e sempre tenta a correção de perspectiva/binarização adaptativa como capacidade geral (ver §5).

### 3.1 `digital` (padrão)

Cobre anexo de arquivo (PNG preservado bit-a-bit) e colagem inline com recompressão JPEG por apps de chat, **sem** captura por câmera.

| Parâmetro          | Valor                    |
|---------------------|--------------------------|
| Níveis de cinza `N` | 8 (3 bits/módulo)        |
| `ModulePx`          | 8px                      |
| Faixa de quantização| 16–239 (evita clipping em 0/255) |
| `DataShards`/`ParityShards`/`ShardSize` | 32 / 8 / 16 bytes (25% de overhead RS) |
| Binarização         | Limiar global fixo (suficiente — sem distorção geométrica a compensar) |

Calibrado empiricamente (Fase 0, `internal/imagecodec/calibration_spike_test.go`, tag `spike`): varredura de `N∈{2,4,8,16} × modulePx∈{8,12,16,24,32}` através de JPEG (qualidade 50–90) **e** de um downscale/upscale bilinear (fator 0.5–1.0, simulando um app que limita a dimensão de upload) mediu **SER = 0 em todas as combinações testadas**, inclusive `N=16, modulePx=8`. Isso é consistente com a teoria: blocos de módulo são regiões planas, e a DCT do JPEG concentra quase toda a energia dessas regiões no coeficiente DC — a quantização de AC (o que a qualidade JPEG afeta) tem efeito desprezível fora das bordas do módulo, que não são amostradas (só o "safe zone" central de 50% é lido). Como o resultado "perfeito" é um artefato conhecido de conteúdo sintético em blocos planos (ver risco §7.1 — encoders reais de apps de chat podem se comportar diferente do `image/jpeg` do Go), **`N=8` foi escolhido em vez do `N=16` tecnicamente "livre de erro" na simulação**, como margem de segurança deliberada contra essa incerteza, mantendo ainda assim 3 bits/módulo (60% mais denso que um esquema binário). O overhead RS de 25% (bem acima do necessário pela simulação) cobre a mesma incerteza.

### 3.2 `camera-safe` (opcional, flag `--camera-safe`)

Cobre adicionalmente uma foto tirada com câmera de celular da imagem exibida na tela (ou impressa). Ativa:
- **4 marcadores de canto assimétricos** + homografia de 4 pontos para corrigir perspectiva (não apenas escala/rotação) — ver §5.2.
- **Binarização local/adaptativa obrigatória** (não threshold global) — ver §5.1.
- Módulos maiores, binários (não multi-nível).

| Parâmetro          | Valor                    |
|---------------------|--------------------------|
| Níveis de cinza `N` | 2 (1 bit/módulo)         |
| `ModulePx`          | 24px                     |
| Faixa de quantização| 16–239                   |
| `DataShards`/`ParityShards`/`ShardSize` | 24 / 10 / 16 bytes (~42% de overhead RS) |
| Binarização         | Adaptativa/local (limiar por região, ver §5.1) — **obrigatória**, threshold global não é suficiente sob iluminação de câmera |

Calibrado empiricamente (mesma spike, sweep dedicado): homografia sintética (~15–20° de inclinação), gradiente de brilho, blur e ruído Gaussiano, com correção de perspectiva usando a homografia *conhecida* (isola o "ruído de canal" da questão separada de detecção de cantos, que é um algoritmo de produção — ver §5.2 e riscos §7.2). Em `modulePx∈{16,24,32,48}`:
- Até severidade **"harsh"** (blur ≈ modulePx/5, ruído σ=28, gradiente de brilho ±80, JPEG qualidade 60): **SER = 0** com threshold global em todos os tamanhos de módulo testados.
- Em severidade **"extreme"** (blur ≈ modulePx/3, ruído σ=45, gradiente ±130, JPEG qualidade 45 — bem além do plausível para uma foto razoável): threshold global mostrou SER de 0.6–1.1%, mas a **binarização adaptativa recuperou SER = 0** em todos os tamanhos de módulo — validando empiricamente a decisão de tornar a binarização adaptativa obrigatória neste perfil, não apenas um "extra".

`ModulePx=24` foi escolhido como ponto médio (não o menor `16px` tecnicamente testado como suficiente) porque a simulação usa uma homografia *conhecida*, não detectada — a detecção real de cantos (produção, §5.2) tem sua própria margem de erro não capturada aqui, então módulos maiores dão folga adicional para essa incerteza não simulada.

### 3.3 Consequência de densidade

Para um payload comprimido de referência de 10KB: `digital` produz uma imagem de aproximadamente **1480×1480px**; `camera-safe`, aproximadamente **8184×8184px** (≈27× menos denso em bits/pixel, esperado — módulos binários maiores custam muito em área). Isso é intencional e deve ficar visível no `--help`/README: `camera-safe` é recomendado para payloads pequenos (poucos KB), não para pacotes grandes.

### 3.4 Nota de implementação (pós-implementação, Passo 4) — números finais divergem dos calibrados acima

A implementação real (TDD, `internal/imagecodec`) precisou de parâmetros RS diferentes dos calibrados na Fase 0 acima. A calibração da Fase 0 mediu apenas ruído "de canal" (JPEG/blur/luz) assumindo geometria perfeita; a implementação real precisa **também** absorver o erro residual da própria detecção de cantos/homografia — que não é zero mesmo sem nenhuma distorção real — e esse erro concentra-se de forma desigual em poucos *shards* quando o shard é grande. Dois ajustes foram feitos durante o TDD, ambos documentados com sua causa raiz nos comentários de `internal/imagecodec/profile.go` e `perspective.go`:

1. **`ShardSize` reduzido de 16 para 4 bytes** (com `DataShards` aumentado proporcionalmente: `digital` 32→128, `camera-safe` 24→96) — um shard de 16 bytes (~43 símbolos) é inteiro descartado por um único símbolo errado; encolher o shard faz cada erro "custar" muito menos do orçamento de paridade.
2. **`ParityShards` aumentado** — `digital` de 8 (25%) para 48 (37,5%); `camera-safe` de 10 (~42%) para 56 (~58%) — para cobrir o ruído residual real de geometria, medido via testes de ponta a ponta (CLI real, não só a spike), não só o ruído de canal isolado da Fase 0.
3. **`markerModules` aumentado de 6 para 10** e os "olhos" de identidade dos marcadores de canto passaram de pontos 1×1 módulo para blocos 2×2 módulos bem espaçados — pontos menores se perdiam sob blur/ruído, colapsando a identidade de cantos distintos (TR/BL/BR) para o mesmo valor.
4. **`locateHeaderHomography`/`locateBodyHomography` usam um ajuste de similaridade (rotação+escala+translação, 4 graus de liberdade) em vez do DLT projetivo completo (8 graus de liberdade)** — o DLT completo, mesmo com 4 pontos bem distribuídos, absorvia ruído comum de detecção de canto em termos espúrios de cisalhamento/perspectiva que não deveriam existir, quebrando a decodificação mesmo sem nenhuma distorção real. **Consequência direta: correção de perspectiva real (inclinação de câmera genuína) não está implementada nesta versão** — só é tolerado o caso sem inclinação (rotação/escala/iluminação/ruído/JPEG). Ver o comentário de `TestCameraSafeProfile_SurvivesHomographyAndLighting` em `imagecodec_test.go` para o estado exato e o que falta para fechar essa lacuna.

Consequência de densidade **revisada**: com os números finais, `digital` produz ~1800–2200px de lado e `camera-safe` ~10500–12000px de lado para o mesmo payload de 10KB — maior que o estimado em §3.3, mas dentro da mesma ordem de grandeza e da mesma conclusão qualitativa (camera-safe é para payloads pequenos).

---

## 4. Estratégia de Detecção de Erasure (Reed-Solomon)

`github.com/klauspost/reedsolomon` é um codec de **erasure** (RAID-style): só reconstrói shards explicitamente marcados como ausentes (`nil`), sem localizar erros a partir de bits corrompidos em posição desconhecida. Isso exige uma camada própria de detecção antes de invocar `Reconstruct`:

1. **Payload comprimido é dividido em blocos RS fixos** (`DataShards`/`ParityShards`/`ShardSize` do cabeçalho, por perfil — §3), repetidos ao longo de todo o payload (limite de 255 shards totais por instância RS, `dataShards+parityShards ≤ 256`).
2. **Cada shard recebe um CRC-16** (tabela feita à mão, sem dependência nova — mesmo padrão do `internal/encoder/base91.go` já existente), guardado num diretório de integridade separado, protegido com redundância própria mais pesada que o corpo.
3. **Intercalação espacial:** bytes fisicamente vizinhos na grade pertencem a blocos RS diferentes (permutação determinística de seed fixa entre índice de símbolo RS e coordenada `(x,y)` do módulo — o mesmo princípio dos códigos QR multi-bloco), para que um artefato JPEG localizado (blocos DCT 8×8) não estoure o orçamento de paridade de um único bloco.
4. **Na decodificação:** recalcula-se o CRC de cada shard lido; shard com CRC inválido, **ou** com margem de confiança baixa na quantização de nível de cinza (distância entre o valor amostrado e o nível mais próximo vs. o segundo mais próximo), é marcado como erasure. `Reconstruct` é chamado por bloco, mesmo que a contagem de erasures exceda `ParityShards` daquele bloco (tentativa best-effort).
5. **Garantia real de integridade:** após remontar o payload de todos os blocos, o SHA-256 do cabeçalho é recalculado e comparado. **Isso é o gate de correção real** — se não bater, mesmo que CRC/confiança tenham deixado passar algo, a operação falha explicitamente (`imagecodec.ErrIntegrityMismatch`), nunca retorna dado corrompido silenciosamente.

### 4.3 Cabeçalho: proteção dedicada

O cabeçalho (64 bytes) usa seu próprio bloco RS, deliberadamente sobre-protegido dado seu custo em pixels ser desprezível: `DataShards=8`, `ParityShards=8`, `ShardSize=8` bytes (50% de overhead, tolera perda de até metade dos shards do cabeçalho), posicionado nos módulos adjacentes ao marcador de canto superior-esquerdo.

---

## 5. Alinhamento, Escala e Perspectiva

### 5.1 Binarização

- **Perfil `digital`:** limiar global fixo no ponto médio da faixa de quantização (`(16+239)/2 ≈ 127`) — suficiente, pois não há distorção geométrica/luminosa a compensar (confirmado na calibração, §3.1).
- **Perfil `camera-safe`:** limiar **local/adaptativo** — estimativa de fundo por região (ex.: blur de raio grande, várias vezes o tamanho do módulo) comparada ao valor amostrado de cada módulo, para tolerar vinheta/iluminação desigual de câmera. Threshold global é insuficiente sob gradiente de iluminação forte (confirmado na calibração, §3.2) — esta é uma exigência formal do perfil, não uma otimização opcional.

### 5.2 Padrões de Alinhamento

- **`digital`:** só é necessário compensar **escala** (app pode redimensionar) e **rotação/flip 90/180/270** (defesa barata contra normalização de orientação por algum pipeline). Um padrão "finder" assimétrico em 3 cantos (estilo QR) resolve rotação/flip; um padrão de "timing" ao longo das bordas mede o pitch real do módulo para reamostragem não inteira.
- **`camera-safe`:** uma foto introduz uma transformação projetiva real (homografia), não só escala/rotação. Usa **4 marcadores de canto distintos e assimétricos** (não os 3 do QR clássico, que dependem de extrapolação para o 4º canto) — a homografia é estimada diretamente das 4 correspondências de ponto detectadas (DLT, o mesmo algoritmo prototipado na spike de calibração), e a imagem capturada é reamostrada de volta para a grade canônica antes de demodular. Como o alvo é sempre gerado e fotografado por software/câmera (nunca uma cena 3D arbitrária), não é necessária correção de distorção de lente tipo "alignment patterns" do QR — só a homografia de 4 pontos.
- Efeito moiré ao fotografar a tela de um dispositivo não tem mitigação algorítmica garantida neste spec — fica como orientação de uso (documentar no `--help`/README: fotografar de uma distância que deixe cada módulo com vários pixels de câmera de largura), não como critério de aceitação formal.

---

## 6. Integração com o Pipeline Existente

- `cmd/pack2txt/main.go`: novas flags `--image` (bool, em `pack`) e `--camera-safe` (bool, só tem efeito com `--image`). `-c/--compressor` é ignorado com aviso quando `--image` está ativo (compressor é sempre `auto`).
- `internal/packer/detect.go`: `DetectFormat(data []byte) Format` — sniff de magic bytes (PNG `\x89PNG\r\n\x1a\n`, JPEG `\xFF\xD8\xFF`, ou prefixo texto `PACK2TXT:`), usado por `unpack`/`inspect` para rotear automaticamente. **Sem novo subcomando.**
- `internal/imagecodec.LooksLikeImage([]byte) bool` e `DecodeAuto(io.Reader)` fazem esse sniff no nível do próprio pacote de imagem, reaproveitado por `detect.go`.

---

## 7. Riscos Conhecidos (não bloqueantes para aprovação)

1. Os números calibrados (§3) usam apenas `image/jpeg`/`image/png` do próprio Go e degradações sintéticas — encoders reais de apps de chat (libjpeg-turbo/mozjpeg, auto-enhance, subsampling, resize com filtros diferentes) podem se comportar diferente. Não há validação manual em navegador ou foto física real nesta rodada (decisão do usuário).
2. `camera-safe` foi calibrado assumindo homografia **conhecida** (não detectada) — a detecção real de cantos em `internal/imagecodec/perspective.go` (produção, ainda não implementada) tem sua própria margem de erro não capturada nesta simulação; `ModulePx=24` já inclui folga para isso, mas não é uma garantia formal.
3. Sem limite de tamanho de imagem (decisão do usuário): payloads grandes em `camera-safe` geram imagens muito grandes (ver §3.3) — isso é esperado, não uma regressão, e deve ficar documentado no `--help`/README.
4. Efeito moiré ao fotografar uma tela (§5.2) não tem mitigação algorítmica formal.
5. **[Confirmado na implementação, não mais hipotético]** Correção de perspectiva real (inclinação de câmera genuína, não só rotação/escala) não está implementada — `locateHeaderHomography`/`locateBodyHomography` usam um ajuste de similaridade (4 graus de liberdade), não a homografia projetiva completa (8 graus de liberdade), porque o DLT completo absorvia ruído de detecção de canto em termos espúrios de cisalhamento/perspectiva que quebravam a decodificação mesmo sem distorção real (ver §3.4). Até essa lacuna ser fechada, `camera-safe` cobre rotação/escala/iluminação desigual/ruído/JPEG, mas **não** uma foto tirada em ângulo real.

---

## 8. Critérios de Aceitação (Gherkin Scenarios)

```gherkin
Feature: Image Transport & Loss-Tolerant Module Codec

  Scenario: Round-trip bit-idêntico via PNG (perfil digital)
    Given um payload comprimido arbitrário de até 100KB
    When ele é codificado com o perfil "digital" e decodificado a partir do PNG gerado, sem nenhuma degradação
    Then o payload decodificado deve ser 100% idêntico ao original
    And o SHA-256 do cabeçalho deve corresponder ao payload decodificado

  Scenario: Round-trip bit-idêntico via PNG (perfil camera-safe)
    Given um payload comprimido arbitrário de até 20KB
    When ele é codificado com o perfil "camera-safe" e decodificado a partir do PNG gerado, sem nenhuma degradação
    Then o payload decodificado deve ser 100% idêntico ao original

  Scenario: Round-trip pós-recompressão JPEG (perfil digital)
    Given um payload codificado com o perfil "digital"
    When a imagem passa por um downscale/upscale bilinear de até 50% seguido de reencode JPEG em qualidade >= 75
    Then o payload decodificado deve ser 100% idêntico ao original

  Scenario: Round-trip pós-homografia, iluminação, blur e JPEG (perfil camera-safe)
    Given um payload codificado com o perfil "camera-safe"
    When a imagem sofre uma homografia sintética de até ~20° de inclinação, gradiente de brilho, blur e ruído gaussiano no nível "harsh" (ver §3.2), seguido de reencode JPEG em qualidade >= 60
    Then o payload decodificado deve ser 100% idêntico ao original usando binarização adaptativa

  Scenario: Falha limpa além da capacidade de correção
    Given uma imagem codificada cuja corrupção de pixels excede a capacidade de correção Reed-Solomon de múltiplos blocos
    When o unpack é executado
    Then a operação deve falhar explicitamente com ErrIntegrityMismatch
    And nenhum dado parcial ou corrompido deve ser escrito em disco

  Scenario: Densidade de imagem por perfil
    Given um payload comprimido de exatamente 10KB
    When ele é codificado com o perfil "digital"
    Then a imagem resultante não deve exceder 1600x1600 pixels
    When o mesmo payload é codificado com o perfil "camera-safe"
    Then a imagem resultante não deve exceder 9000x9000 pixels

  Scenario: Auto-detecção de formato no unpack/inspect
    Given um arquivo PNG ou JPEG gerado pelo pack2txt no modo imagem
    When "pack2txt unpack" ou "pack2txt inspect" é executado apontando para esse arquivo, sem nenhuma flag adicional
    Then o formato deve ser detectado automaticamente via magic bytes
    And o resultado deve ser idêntico ao caminho de texto equivalente
```
