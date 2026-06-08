package main

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var exportCmd = &cobra.Command{
	Use:   "export [format] [note-name]",
	Short: "Export a note to HTML or PDF",
	Long: `Exports a note to the specified format.

Supported formats: html, pdf

Examples:
  vault export html my-note
  vault export pdf my-note
  vault export html my-note --output /tmp/note.html
  vault export all html --output ./exports`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] == "all" {
			return exportAll(cmd, args)
		}
		format := args[0]
		noteName := args[1]

		if format != "html" && format != "pdf" {
			return fmt.Errorf("unsupported format: %s (use html, pdf, or all)", format)
		}

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, err := findNote(mgr, noteName)
		if err != nil {
			return fmt.Errorf("find note: %w", err)
		}

		content, err := os.ReadFile(filepath.Join(mgr.NotesDir(), note.Filename))
		if err != nil {
			return fmt.Errorf("read note: %w", err)
		}

		output, _ := cmd.Flags().GetString("output")

		notesDir := mgr.NotesDir()

		switch format {
		case "html":
			return exportHTML(notesDir, note.Filename, string(content), output)
		case "pdf":
			return exportPDF(notesDir, note.Filename, string(content), output)
		}
		return nil
	},
}

func exportAll(cmd *cobra.Command, args []string) error {
	format := args[1]
	if format != "html" && format != "pdf" {
		return fmt.Errorf("unsupported format: %s (use html or pdf)", format)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "" {
		output = "exports"
	}

	if err := os.MkdirAll(output, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	mgr, err := newManager()
	if err != nil {
		return fmt.Errorf("open vault: %w", err)
	}

	notes, err := mgr.ListNotes()
	if err != nil {
		return fmt.Errorf("list notes: %w", err)
	}

	if len(notes) == 0 {
		fmt.Println("No notes to export")
		return nil
	}

	var exported int
	for _, n := range notes {
		content, readErr := os.ReadFile(filepath.Join(mgr.NotesDir(), n.Filename))
		if readErr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠  %s: %v\n", n.Filename, readErr)
			continue
		}

		outPath := filepath.Join(output, strings.TrimSuffix(n.Filename, ".md")+"."+format)
		switch format {
		case "html":
			if err := exportHTML(mgr.NotesDir(), n.Filename, string(content), outPath); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "⚠  %s: %v\n", n.Filename, err)
				continue
			}
		case "pdf":
			if err := exportPDF(mgr.NotesDir(), n.Filename, string(content), outPath); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "⚠  %s: %v\n", n.Filename, err)
				continue
			}
		}
		exported++
	}

	fmt.Printf("✓ Exported %d %s file(s) to %s\n", exported, format, output)
	return nil
}

func exportHTML(notesDir, filename, content, outputPath string) error {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return fmt.Errorf("convert markdown: %w", err)
	}

	title := strings.TrimSuffix(filename, ".md")
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <style>
    body { max-width: 800px; margin: 40px auto; padding: 0 20px;
           font-family: -apple-system, BlinkMacSystemFont, sans-serif;
           line-height: 1.6; color: #333; }
    pre { background: #f5f5f5; padding: 16px; border-radius: 4px; overflow-x: auto; }
    code { background: #f5f5f5; padding: 2px 4px; border-radius: 2px; }
    img { max-width: 100%%; }
    blockquote { border-left: 3px solid #ddd; margin-left: 0; padding-left: 16px; color: #666; }
  </style>
</head>
<body>
%s
</body>
</html>`, html.EscapeString(title), buf.String())

	if outputPath == "" {
		outputPath = filepath.Join(notesDir, strings.TrimSuffix(filename, ".md")+".html")
	}

	if err := os.WriteFile(outputPath, []byte(htmlContent), 0o644); err != nil {
		return fmt.Errorf("write html: %w", err)
	}

	abs, _ := filepath.Abs(outputPath)
	fmt.Printf("✓ Exported to %s\n", abs)
	return nil
}

func exportPDF(notesDir, filename, content, outputPath string) error {
	tmpDir, err := os.MkdirTemp("", "vault-export-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	htmlPath := filepath.Join(tmpDir, strings.TrimSuffix(filename, ".md")+".html")
	if err := exportHTML(notesDir, filename, content, htmlPath); err != nil {
		return err
	}

	if outputPath == "" {
		outputPath = filepath.Join(notesDir, strings.TrimSuffix(filename, ".md")+".pdf")
	}

	if _, err := exec.LookPath("pandoc"); err != nil {
		return fmt.Errorf("pandoc not found, install it first: brew install pandoc")
	}

	cmd := exec.Command("pandoc", htmlPath, "-o", outputPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc: %w", err)
	}

	abs, _ := filepath.Abs(outputPath)
	fmt.Printf("✓ Exported to %s\n", abs)
	return nil
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("output", "o", "", "Output file path")
}
