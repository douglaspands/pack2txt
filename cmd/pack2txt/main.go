package main

import (
	"fmt"
	"io"
	"os"

	"github.com/douglas/pack2txt/internal/compressor"
	"github.com/douglas/pack2txt/internal/encoder"
	"github.com/douglas/pack2txt/internal/packer"
	"github.com/douglas/pack2txt/internal/ui"
	"github.com/spf13/cobra"
)

var (
	Version   = "1.0.0"
	BuildDate = "2026-08-14"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		ui.RenderError(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "pack2txt",
	Short: "pack2txt: Text-based Solid Archive Tool com compressão máxima e codificação de alta densidade",
	Long: `pack2txt é uma ferramenta CLI de alta performance para empacotar estruturas
completas de arquivos em envelopes textuais otimizados para chats de IA (OpenSpec v1).

Utiliza Solid TAR em memória, compressão máxima (Brotli Q11 / Zstd / Auto) e codificadores
de altíssima densidade (Base32768 por padrão, Base91, Base85, Base64).`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		// Shortcut: se o primeiro argumento for um caminho de arquivo/pasta existente, assume o comando 'pack'
		targetPath := args[0]
		if _, err := os.Stat(targetPath); err == nil {
			return runPack(targetPath, packFlags)
		}

		return cmd.Help()
	},
}

type packFlagsStruct struct {
	outputPath string
	compName   string
	encName    string
	stdout     bool
	noIgnore   bool
}

var packFlags packFlagsStruct

var packCmd = &cobra.Command{
	Use:   "pack <origem>",
	Short: "Empacota arquivos e pastas em um envelope de texto puro",
	Long: `Compacta a pasta ou arquivo em Solid TAR na memória, aplica compressão máxima
(Brotli Q11 ou Auto) e codifica em texto Base32768 (ou Base91, Base85, Base64).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPack(args[0], packFlags)
	},
}

func runPack(sourcePath string, flags packFlagsStruct) error {
	isInteractive := ui.IsTerminal(os.Stdout)
	isPipe := flags.stdout || !isInteractive

	opts := packer.PackOptions{
		SourcePath: sourcePath,
		Compressor: flags.compName,
		Encoder:    flags.encName,
		NoIgnore:   flags.noIgnore,
		OutputPath: flags.outputPath,
	}

	result, err := packer.Pack(opts)
	if err != nil {
		return err
	}

	if isPipe {
		// Output clean payload directly to stdout for Unix pipes
		fmt.Fprintln(os.Stdout, result.Envelope)
	} else {
		// Render beautiful terminal UI
		ui.RenderPackResult(result, false)
		if flags.outputPath == "" {
			fmt.Println(result.Envelope)
		}
	}

	return nil
}

type unpackFlagsStruct struct {
	destDir   string
	overwrite bool
	quiet     bool
}

var unpackFlags unpackFlagsStruct

var unpackCmd = &cobra.Command{
	Use:   "unpack [arquivo.txt|-]",
	Short: "Restaura os arquivos originais a partir de um envelope de texto",
	Long: `Lê o envelope de texto (de um arquivo, argumento direto ou stdin),
decodifica, descomprime e reconstrói os arquivos em disco com proteção anti-Zip Slip.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var inputPath string
		var inputRead io.Reader

		if len(args) == 0 || args[0] == "-" {
			// Read from stdin if not interactive or piped
			if !ui.IsTerminal(os.Stdin) || (len(args) > 0 && args[0] == "-") {
				inputRead = os.Stdin
			} else {
				return fmt.Errorf("informe o caminho do arquivo .txt ou envie dados via pipe (stdin)")
			}
		} else {
			inputPath = args[0]
		}

		opts := packer.UnpackOptions{
			InputPath: inputPath,
			InputRead: inputRead,
			DestDir:   unpackFlags.destDir,
			Overwrite: unpackFlags.overwrite,
		}

		res, err := packer.Unpack(opts)
		if err != nil {
			return err
		}

		ui.RenderUnpackResult(res, unpackFlags.quiet)
		return nil
	},
}

var inspectCmd = &cobra.Command{
	Use:   "inspect [arquivo.txt|-]",
	Short: "Inspeciona o conteúdo e as métricas do arquivo sem extraí-lo",
	Long:  `Lê o envelope de texto e exibe a lista de arquivos, tamanhos e taxa de compressão em memória.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var inputPath string
		var inputRead io.Reader

		if len(args) == 0 || args[0] == "-" {
			if !ui.IsTerminal(os.Stdin) || (len(args) > 0 && args[0] == "-") {
				inputRead = os.Stdin
			} else {
				return fmt.Errorf("informe o caminho do arquivo .txt ou envie dados via pipe (stdin)")
			}
		} else {
			inputPath = args[0]
		}

		opts := packer.InspectOptions{
			InputPath: inputPath,
			InputRead: inputRead,
		}

		res, err := packer.Inspect(opts)
		if err != nil {
			return err
		}

		ui.RenderInspectResult(res)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Exibe a versão do pack2txt",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pack2txt v%s (OpenSpec v1, Data: %s)\n", Version, BuildDate)
		fmt.Printf("Compressores: %v\n", compressor.Available())
		fmt.Printf("Codificadores: %v (Padrão: %s)\n", encoder.Available(), encoder.Default().Name())
	},
}

func init() {
	// Flags do comando pack
	packCmd.Flags().StringVarP(&packFlags.outputPath, "output", "o", "", "Caminho do arquivo de saída .txt")
	packCmd.Flags().StringVarP(&packFlags.compName, "compressor", "c", compressor.NameBrotli, "Algoritmo de compressão (brotli | zstd | gzip | none | auto)")
	packCmd.Flags().StringVarP(&packFlags.encName, "encoder", "e", encoder.NameBase32768, "Codificador textual (b32768 | b91 | b85 | b64)")
	packCmd.Flags().BoolVar(&packFlags.stdout, "stdout", false, "Emite apenas o texto do envelope na saída padrão (ideal para pipes)")
	packCmd.Flags().BoolVar(&packFlags.noIgnore, "no-ignore", false, "Não ignora pastas e arquivos de build/dev (.git, node_modules, etc.)")

	// Compartilha flags no root para comando atalho
	rootCmd.Flags().StringVarP(&packFlags.outputPath, "output", "o", "", "Caminho do arquivo de saída .txt")
	rootCmd.Flags().StringVarP(&packFlags.compName, "compressor", "c", compressor.NameBrotli, "Algoritmo de compressão (brotli | zstd | gzip | none | auto)")
	rootCmd.Flags().StringVarP(&packFlags.encName, "encoder", "e", encoder.NameBase32768, "Codificador textual (b32768 | b91 | b85 | b64)")
	rootCmd.Flags().BoolVar(&packFlags.stdout, "stdout", false, "Emite apenas o texto do envelope na saída padrão")
	rootCmd.Flags().BoolVar(&packFlags.noIgnore, "no-ignore", false, "Não ignora pastas padrão de desenvolvimento")

	// Flags do comando unpack
	unpackCmd.Flags().StringVarP(&unpackFlags.destDir, "dest", "d", ".", "Diretório de destino para extração")
	unpackCmd.Flags().BoolVarP(&unpackFlags.overwrite, "force", "f", false, "Sobrescreve arquivos existentes sem erro")
	unpackCmd.Flags().BoolVarP(&unpackFlags.quiet, "quiet", "q", false, "Suprime a saída visual")

	rootCmd.AddCommand(packCmd)
	rootCmd.AddCommand(unpackCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(versionCmd)
}
