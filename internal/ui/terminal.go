package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/douglas/pack2txt/internal/packer"
	"github.com/pterm/pterm"
	"golang.org/x/term"
)

// IsTerminal returns true if the file descriptor is an interactive terminal.
func IsTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// FormatBytes returns a human-readable byte size representation.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// RenderPackResult displays a modern summary card for the pack operation.
func RenderPackResult(res *packer.PackResult, isPipe bool) {
	if isPipe {
		return
	}

	pterm.DefaultHeader.WithFullWidth(false).
		WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Println(" 📦 PACK2TXT: EMPACOTAMENTO CONCLUÍDO ")

	pterm.Println()

	data := [][]string{
		{"Arquivos Empacotados", pterm.Green(strconv.Itoa(res.FileCount))},
		{"Tamanho Original", pterm.Yellow(FormatBytes(res.UncompressedSize))},
		{"Tamanho Comprimido", pterm.Cyan(FormatBytes(res.CompressedSize))},
		{"Caracteres do Payload", pterm.Magenta(fmt.Sprintf("%d runes", res.EncodedChars))},
		{"Taxa de Economia", pterm.LightGreen(fmt.Sprintf("%.2f%%", res.SavingsPercent))},
		{"Algoritmo de Compressão", pterm.Bold.Sprint(res.Compressor)},
		{"Codificador de Texto", pterm.Bold.Sprint(res.Encoder)},
		{"Tempo de Execução", pterm.Gray(res.Duration.Truncate(time.Millisecond).String())},
	}

	if res.OutputPath != "" {
		data = append(data, []string{"Arquivo Salvo em", pterm.LightCyan(res.OutputPath)})
	}

	_ = pterm.DefaultTable.WithData(data).WithBoxed(true).Render()
	pterm.Println()
}

// RenderInspectResult displays the file list and archive metrics.
func RenderInspectResult(res *packer.InspectResult) {
	pterm.DefaultHeader.WithFullWidth(false).
		WithBackgroundStyle(pterm.NewStyle(pterm.BgMagenta)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Println(" 🔍 INSPEÇÃO DE PACOTE ")

	pterm.Println()

	tableData := [][]string{
		{"Tipo", "Permissões", "Tamanho", "Data Modificação", "Caminho"},
	}

	for _, entry := range res.Entries {
		t := "📄 File"
		if entry.IsDir {
			t = "📁 Dir"
		}
		tableData = append(tableData, []string{
			t,
			entry.Mode.String(),
			FormatBytes(entry.Size),
			entry.ModTime.Format("2006-01-02 15:04"),
			entry.Path,
		})
	}

	_ = pterm.DefaultTable.WithHasHeader(true).WithData(tableData).WithBoxed(true).Render()

	pterm.Println()

	pterm.DefaultSection.Println("Métricas do Arquivo")
	pterm.Printf("• Total de Arquivos: %s | Pastas: %s\n", pterm.Green(res.FileCount), pterm.Cyan(res.DirCount))
	pterm.Printf("• Tamanho Total Descomprimido: %s\n", pterm.Yellow(FormatBytes(res.UncompressedSize)))
	pterm.Printf("• Tamanho Comprimido (%s): %s (%s de economia)\n",
		pterm.Bold.Sprint(res.Compressor),
		pterm.Cyan(FormatBytes(res.CompressedSize)),
		pterm.LightGreen(fmt.Sprintf("%.2f%%", res.SavingsPercent)),
	)
	pterm.Printf("• Codificador: %s (%s caracteres)\n", pterm.Bold.Sprint(res.Encoder), pterm.Magenta(res.EncodedChars))
	pterm.Println()
}

// RenderUnpackResult displays a summary card after extraction.
func RenderUnpackResult(res *packer.UnpackResult, quiet bool) {
	if quiet {
		return
	}

	absDest, _ := filepath.Abs(res.DestDir)

	pterm.DefaultHeader.WithFullWidth(false).
		WithBackgroundStyle(pterm.NewStyle(pterm.BgGreen)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Println(" 🚀 DESCOMPACTAÇÃO CONCLUÍDA COM SUCESSO ")

	pterm.Println()

	data := [][]string{
		{"Arquivos Extraídos", pterm.Green(strconv.Itoa(res.FileCount))},
		{"Tamanho Total em Disco", pterm.Yellow(FormatBytes(res.TotalSize))},
		{"Diretório de Destino", pterm.LightCyan(absDest)},
		{"Compressor Utilizado", pterm.Bold.Sprint(res.Compressor)},
		{"Encoder Utilizado", pterm.Bold.Sprint(res.Encoder)},
		{"Tempo de Extração", pterm.Gray(res.Duration.Truncate(time.Millisecond).String())},
	}

	_ = pterm.DefaultTable.WithData(data).WithBoxed(true).Render()
	pterm.Println()
}

// RenderError prints a formatted error message.
func RenderError(err error) {
	pterm.Error.Println(err)
}
