# Guia de Performance, Eficiência e Boas Práticas do Harness

**Projeto:** `pack2txt`  
**Escopo:** Diretrizes de Engenharia de Software de Alta Performance em Go (Golang)  
**Versão:** 1.0.0  

---

## 1. Princípios Fundamentais de Performance

A ferramenta `pack2txt` foi desenhada para operar com latência mínima, consumo previsível de memória e throughput máximo. Para atingir esses objetivos, todas as implementações no projeto DEVEM aderir às seguintes diretrizes:

---

## 2. Arquitetura Zero-Disk (Streaming 100% em Memória)

### 2.1 Eliminação de Arquivos Temporários
* **Regra:** Sob nenhuma circunstância a CLI deve criar arquivos temporários em disco (`/tmp`, `os.CreateTemp`) durante o empacotamento ou inspeção.
* **Justificativa:**
  - Reduz operações de I/O de disco a zero durante o pipeline de compressão e codificação.
  - Previne vazamento de dados sensíveis em diretórios temporários do sistema.
  - Viabiliza execução ultra-rápida em contêineres efêmeros e ambientes restritos (read-only filesystems).

### 2.2 Reutilização de Buffers e `sync.Pool`
Para evitar sobrecarga no Garbage Collector (GC) ao processar múltiplos arquivos ou pacotes grandes:
```go
var bufferPool = sync.Pool{
    New: func() any {
        return bytes.NewBuffer(make([]byte, 0, 64*1024)) // Pré-aloca 64 KB
    },
}

func getBuffer() *bytes.Buffer {
    buf := bufferPool.Get().(*bytes.Buffer)
    buf.Reset()
    return buf
}

func putBuffer(buf *bytes.Buffer) {
    if buf.Cap() <= 4*1024*1024 { // Evita reter buffers gigantes na pool
        bufferPool.Put(buf)
    }
}
```

---

## 3. Otimização dos Algoritmos de Compressão

### 3.1 Brotli (Nível Máximo Q11)
* **Configuração:** `brotli.WriterOptions{Quality: 11, LGWin: 22}` (4 MB sliding window).
* **Uso de Memória Controlado:** A janela de 22 bits oferece o equilíbrio ideal entre deduplicação inter-arquivos e consumo de memória RAM (< 16 MB de overhead).

### 3.2 Zstandard (`klauspost/compress/zstd`)
* **Configuração:** `zstd.WithEncoderLevel(zstd.SpeedBestCompression)` e `zstd.WithEncoderConcurrency(1)` para execuções deterministicamente sequenciais sem overhead de goroutines desnecessárias em arquivos pequenos.
* Reutilização obrigatória dos encoders e decoders via pooling:
```go
var zstdEncoderPool = sync.Pool{
    New: func() any {
        enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
        return enc
    },
}
```

### 3.3 Auto-Optimizer Concorrente
No modo `-c auto`:
* O buffer TAR é passado simultaneamente para `brotli`, `zstd` e `gzip` executando em goroutines concorrentes (`sync.WaitGroup` ou `errgroup`).
* O compressor que gerar o menor `len(outputBytes)` é selecionado em poucos milissegundos.
* Se a base de dados for superior a 50 MB, o modo auto avalia uma amostragem inicial para selecionar o melhor compressor sem consumir memória excessiva.

---

## 4. Otimização de Codificação Binário-para-Texto

### 4.1 Pré-Alocação de Slices e Strings
Evitar redimensionamentos dinâmicos sucessivos durante `Encode` e `Decode`:

| Algoritmo | Fórmula de Capacidade Exata |
|---|---|
| **Base64** | `cap = ((len(src) + 2) / 3) * 4` |
| **Base85** | `cap = ((len(src) + 3) / 4) * 5` |
| **Base91** | `cap = int(float64(len(src)) * 1.23) + 16` |
| **Base32768** | `capRunes = ((len(src) * 8) + 14) / 15` |

### 4.2 Manipulação Eficiente de Runas UTF-8 em Base32768
* Utilizar `strings.Builder` com `builder.Grow(estimatedBytes)` ou codificação direta em `[]byte` via `utf8.EncodeRune` para evitar alocações intermediárias de strings na heap.

---

## 5. Entrada/Saída (I/O) e Integração com Pipes Unix

### 5.1 Separação Estrita de Streams (`stdout` vs `stderr`)
* **`stdout`:** Reservado **estritamente** para dados puros quando executado com `--stdout` ou redirecionado via pipe (`| pbcopy`, `> output.txt`).
* **`stderr`:** Utilizado para mensagens de progresso, tabelas estatísticas, avisos e tempos de execução.
* **Detecção Automática:**
```go
func isTerminal(f *os.File) bool {
    // Usa term.IsTerminal(int(f.Fd()))
}
```
Se `os.Stdout` não for um terminal (pipe ativo), a CLI desativa automaticamente decorações ANSI e emite apenas o envelope puro.

### 5.2 Leitura Eficiente de `stdin`
* Leitura em streaming com `bufio.Reader` e limite de segurança para prevenir estouro de memória (`io.LimitReader`).

---

## 6. Checklist de Qualidade e Benchmark de Performance

Antes de considerar qualquer módulo pronto:
1. **Benchmark de Alocações:** Executar `go test -bench=. -benchmem ./...` e verificar `0 B/op` ou alocações mínimas nos loops de codificação.
2. **Checagem de Race Conditions:** Executar `go test -race ./...`.
3. **Validação de Vazamento de Memória:** Executar testes contínuos com `pprof` garantindo que a memória retorne à linha de base após grandes operações.
