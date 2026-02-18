package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Struktur hasil PowerShell Get-Printer
type psPrinter struct {
	Name         string `json:"Name"`
	Shared       bool   `json:"Shared"`
	ComputerName string `json:"ComputerName"`
	ShareName    string `json:"ShareName"`
	WorkOffline  bool   `json:"WorkOffline"`
}

var (
	logger *log.Logger
)

const (
	maxTextSize     = 10 * 1024 * 1024 // 10MB
	psTimeout       = 15 * time.Second
	serverAddr      = "127.0.0.1:8081"
	shutdownTimeout = 10 * time.Second
)

func main() {
	// Setup logging
	logFile, err := setupLogging()
	if err != nil {
		fmt.Printf("❌ Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Setup HTTP routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/printers", printersHandler)
	http.HandleFunc("/print", printHandler)

	// Create server with timeouts
	server := &http.Server{
		Addr:         serverAddr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		logger.Println("🛑 Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Printf("❌ Server shutdown error: %v", err)
		}
	}()

	logger.Printf("🚀 PrintAgent started on http://%s", serverAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("❌ ListenAndServe error: %v", err)
	}

	logger.Println("✅ Server stopped gracefully")
}

// ----------------- Logging Setup -----------------
func setupLogging() (*os.File, error) {
	logPath := "printagent.log"
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}

	mw := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(mw, "", log.LstdFlags|log.Lshortfile)
	logger.Printf("=== PrintAgent started at %s ===", time.Now().Format(time.RFC3339))

	return logFile, nil
}

// ----------------- Handlers -----------------

func healthHandler(w http.ResponseWriter, r *http.Request) {
	addCors(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "ok",
		"message": "PrintAgent is running",
		"time":    time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)
}

func printersHandler(w http.ResponseWriter, r *http.Request) {
	addCors(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger.Println("📋 Fetching printer list...")
	list, err := listPrinters()
	if err != nil {
		logger.Printf("❌ Failed to list printers: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list printers: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Printf("✅ Found %d printer(s)", len(list))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"printers": list,
		"count":    len(list),
	})
}

func printHandler(w http.ResponseWriter, r *http.Request) {
	addCors(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and parse body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		return
	}

	values, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		logger.Printf("❌ Failed to parse form data: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	printerName := strings.TrimSpace(values.Get("printerName"))
	text := values.Get("text")

	// Validasi input
	if printerName == "" {
		logger.Println("⚠️ Print request without printer name")
		http.Error(w, "Missing printerName", http.StatusBadRequest)
		return
	}

	if text == "" {
		logger.Printf("⚠️ Print request with empty text for printer: %s", printerName)
		http.Error(w, "Missing text", http.StatusBadRequest)
		return
	}

	if len(text) > maxTextSize {
		logger.Printf("⚠️ Text too large (%d bytes) for printer: %s", len(text), printerName)
		http.Error(w, fmt.Sprintf("Text too large (max %d MB)", maxTextSize/(1024*1024)), http.StatusBadRequest)
		return
	}

	// Cek status offline untuk printer lokal
	if !strings.HasPrefix(printerName, `\\`) {
		offline, err := isPrinterOffline(printerName)
		if err != nil {
			// Jangan block printing jika check gagal
			logger.Printf("⚠️ Cannot check printer status for %s: %v (continuing anyway)", printerName, err)
		} else if offline {
			logger.Printf("❌ Printer is offline: %s", printerName)
			http.Error(w, fmt.Sprintf("Printer is offline: %s", printerName), http.StatusServiceUnavailable)
			return
		}
	} else {
		// Skip offline check untuk network printers
		logger.Printf("⏭️ Skipping offline check for network printer: %s", printerName)
	}

	// Kirim ke printer
	logger.Printf("🖨️ Printing %d bytes to: %s", len(text), printerName)
	if err := sendToPrinter(printerName, []byte(text)); err != nil {
		logger.Printf("❌ Print failed - Printer: %s, Error: %v", printerName, err)
		http.Error(w, fmt.Sprintf("Print failed: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Printf("✅ Print success - Printer: %s, Size: %d bytes", printerName, len(text))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Printed on %s", printerName),
		"printer": printerName,
		"size":    fmt.Sprintf("%d bytes", len(text)),
	})
}

// ----------------- Utilities -----------------

func addCors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Private-Network", "true")
}

// listPrinters: menggunakan PowerShell Get-Printer dengan timeout
func listPrinters() ([]string, error) {
	ps := `Get-Printer | Select Name,Shared,ComputerName,ShareName,WorkOffline | ConvertTo-Json -Depth 2`

	ctx, cancel := context.WithTimeout(context.Background(), psTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	out, err := cmd.Output()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logger.Println("⚠️ PowerShell timeout, falling back to WMIC")
		} else {
			logger.Printf("⚠️ PowerShell failed: %v, falling back to WMIC", err)
		}
		return listPrintersWMIC()
	}

	outStr := strings.TrimSpace(string(out))
	if outStr == "" || outStr == "null" {
		return []string{}, nil
	}

	var printers []psPrinter
	// Try decode as array
	if err := json.Unmarshal(out, &printers); err != nil {
		// Try single object
		var single psPrinter
		if err2 := json.Unmarshal(out, &single); err2 != nil {
			logger.Printf("⚠️ Cannot parse PowerShell output, falling back to WMIC")
			return listPrintersWMIC()
		}
		printers = append(printers, single)
	}

	result := make([]string, 0, len(printers))
	for _, p := range printers {
		nameLower := strings.ToLower(p.Name)

		// Skip virtual printers (kecuali shared)
		if !p.Shared && isVirtualPrinter(nameLower) {
			continue
		}

		// Skip offline printers
		if p.WorkOffline {
			logger.Printf("⏭️ Skipping offline printer: %s", p.Name)
			continue
		}

		// Untuk shared printer, gunakan format UNC
		if p.Shared && p.ShareName != "" && p.ComputerName != "" {
			uncPath := `\\` + p.ComputerName + `\` + p.ShareName
			result = append(result, uncPath)
		} else {
			result = append(result, p.Name)
		}
	}

	return result, nil
}

// isVirtualPrinter: cek apakah printer adalah virtual printer
func isVirtualPrinter(nameLower string) bool {
	virtualKeywords := []string{"pdf", "xps", "onenote", "fax", "writer", "send to", "microsoft print"}
	for _, keyword := range virtualKeywords {
		if strings.Contains(nameLower, keyword) {
			return true
		}
	}
	return false
}

// listPrintersWMIC: fallback menggunakan WMIC
func listPrintersWMIC() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), psTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cmd", "/c", "wmic printer get Name,WorkOffline /format:csv")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("WMIC failed: %v", err)
	}

	lines := strings.Split(string(out), "\n")
	result := []string{}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || i == 0 || strings.HasPrefix(strings.ToLower(line), "node,") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(parts[1])
		if name == "" {
			continue
		}

		// Skip virtual printers
		if isVirtualPrinter(strings.ToLower(name)) {
			continue
		}

		// Cek WorkOffline (kolom ke-3 jika ada)
		if len(parts) >= 3 {
			workOffline := strings.TrimSpace(parts[2])
			if strings.EqualFold(workOffline, "true") {
				logger.Printf("⏭️ Skipping offline printer (WMIC): %s", name)
				continue
			}
		}

		result = append(result, name)
	}

	return result, nil
}

// isPrinterOffline: cek status WorkOffline untuk printer lokal
func isPrinterOffline(printerName string) (bool, error) {
	// Escape printer name untuk PowerShell
	escapedName := escapeForPS(printerName)

	// Gunakan script yang lebih robust
	ps := fmt.Sprintf(`
		try {
			$p = Get-Printer -Name '%s' -ErrorAction Stop
			Write-Output $p.WorkOffline
		} catch {
			Write-Output "NOTFOUND"
		}
	`, escapedName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps)

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	outStr := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())

	// Log untuk debugging
	logger.Printf("🔍 Checking printer: %s", printerName)
	if errStr != "" {
		logger.Printf("⚠️ PowerShell stderr: %s", errStr)
	}

	if err != nil {
		// Jika error, coba fallback ke WMIC
		logger.Printf("⚠️ PowerShell failed for %s: %v, trying WMIC fallback", printerName, err)
		return isPrinterOfflineWMIC(printerName)
	}

	outLower := strings.ToLower(outStr)

	if outLower == "true" {
		logger.Printf("📴 Printer offline: %s", printerName)
		return true, nil
	}
	if outLower == "false" {
		logger.Printf("✅ Printer online: %s", printerName)
		return false, nil
	}
	if strings.Contains(outLower, "notfound") {
		logger.Printf("❓ Printer not found: %s", printerName)
		return false, fmt.Errorf("printer not found: %s", printerName)
	}

	// Unknown status, assume online
	logger.Printf("⚠️ Unknown printer status for %s: %s (assuming online)", printerName, outStr)
	return false, nil
}

func isPrinterOfflineWMIC(printerName string) (bool, error) {
	// Escape printer name untuk WMIC
	escapedName := strings.ReplaceAll(printerName, `\`, `\\`)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// WMIC query
	query := fmt.Sprintf(`wmic printer where "Name='%s'" get WorkOffline /value`, escapedName)
	cmd := exec.CommandContext(ctx, "cmd", "/c", query)

	out, err := cmd.Output()
	if err != nil {
		logger.Printf("⚠️ WMIC also failed for %s: %v (assuming online)", printerName, err)
		return false, nil // Assume online if we can't check
	}

	outStr := strings.ToLower(strings.TrimSpace(string(out)))

	// Parse WMIC output: "WorkOffline=TRUE" or "WorkOffline=FALSE"
	if strings.Contains(outStr, "workoffline=true") {
		logger.Printf("📴 Printer offline (WMIC): %s", printerName)
		return true, nil
	}

	logger.Printf("✅ Printer online (WMIC): %s", printerName)
	return false, nil
}

func escapeForPS(s string) string {
	// Di dalam single quotes PowerShell, hanya ' yang perlu di-escape dengan ''
	s = strings.ReplaceAll(s, "'", "''")
	return s
}

// ----------------- Winspool (WritePrinter) -----------------

func sendToPrinter(printerName string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	// Load winspool library
	winspool := syscall.NewLazyDLL("winspool.drv")
	procOpenPrinter := winspool.NewProc("OpenPrinterW")
	procClosePrinter := winspool.NewProc("ClosePrinter")
	procStartDocPrinter := winspool.NewProc("StartDocPrinterW")
	procEndDocPrinter := winspool.NewProc("EndDocPrinter")
	procStartPagePrinter := winspool.NewProc("StartPagePrinter")
	procEndPagePrinter := winspool.NewProc("EndPagePrinter")
	procWritePrinter := winspool.NewProc("WritePrinter")

	// Convert printer name to UTF16
	pnUTF16, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return fmt.Errorf("invalid printer name: %v", err)
	}

	// Open printer
	var hPrinter syscall.Handle
	r1, _, e1 := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(pnUTF16)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if r1 == 0 {
		return fmt.Errorf("OpenPrinter failed: %v", e1)
	}
	defer procClosePrinter.Call(uintptr(hPrinter))

	// Prepare DOC_INFO_1 structure
	docName, _ := syscall.UTF16PtrFromString("PrintAgent Job")
	dataType, _ := syscall.UTF16PtrFromString("RAW")

	type docInfo1 struct {
		pDocName    *uint16
		pOutputFile *uint16
		pDatatype   *uint16
	}

	di := docInfo1{
		pDocName:    docName,
		pOutputFile: nil,
		pDatatype:   dataType,
	}

	// Start document
	r1, _, e1 = procStartDocPrinter.Call(
		uintptr(hPrinter),
		uintptr(1),
		uintptr(unsafe.Pointer(&di)),
	)
	if r1 == 0 {
		return fmt.Errorf("StartDocPrinter failed: %v", e1)
	}
	defer procEndDocPrinter.Call(uintptr(hPrinter))

	// Start page
	r1, _, e1 = procStartPagePrinter.Call(uintptr(hPrinter))
	if r1 == 0 {
		return fmt.Errorf("StartPagePrinter failed: %v", e1)
	}
	defer procEndPagePrinter.Call(uintptr(hPrinter))

	// Write data to printer
	var written uint32
	r1, _, e1 = procWritePrinter.Call(
		uintptr(hPrinter),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&written)),
	)
	if r1 == 0 {
		return fmt.Errorf("WritePrinter failed: %v", e1)
	}

	// Verify all data was written
	if int(written) != len(data) {
		return fmt.Errorf("incomplete write: wrote %d of %d bytes", written, len(data))
	}

	return nil
}
