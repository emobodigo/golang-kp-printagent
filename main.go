package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Struktur hasil PowerShell Get-Printer (kita hanya parse field yang diperlukan)
type psPrinter struct {
	Name         string `json:"Name"`
	Shared       bool   `json:"Shared"`
	ComputerName string `json:"ComputerName"`
	ShareName    string `json:"ShareName"`
	WorkOffline  bool   `json:"WorkOffline"`
}

var (
	logFile *os.File
)

func main() {
	setupLogging()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/printers", printersHandler)
	http.HandleFunc("/print", printHandler)

	log.Println("PrintAgent (Go) running on http://localhost:8081")
	if err := http.ListenAndServe("127.0.0.1:8081", nil); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}

// ----------------- Logging Setup -----------------
func setupLogging() {
	logPath := "printagent.log"
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("❌ Cannot open log file:", err)
		os.Exit(1)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("=== PrintAgent started at", time.Now().Format(time.RFC3339), "===")
}

// ----------------- Handlers -----------------

func healthHandler(w http.ResponseWriter, r *http.Request) {
	addCors(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"status":"ok","message":"PrintAgent is running"}`)
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

	list, err := listPrinters()
	if err != nil {
		http.Error(w, fmt.Sprintf("Gagal list printer: %v", err), http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	enc.Encode(list)
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

	bodyBytes, _ := io.ReadAll(r.Body)
	values, _ := url.ParseQuery(string(bodyBytes))
	printerName := values.Get("printerName")
	text := values.Get("text")

	if text == "" {
		http.Error(w, "Missing text", http.StatusBadRequest)
		return
	}
	if printerName == "" {
		http.Error(w, "Missing printerName", http.StatusBadRequest)
		return
	}

	// Jika printer is shared (\\HOST\NAME) kita anggap cek WorkOffline lokal tidak valid
	if !strings.HasPrefix(printerName, `\\`) {
		offline, err := isPrinterOffline(printerName)
		if err == nil && offline {
			http.Error(w, fmt.Sprintf("Printer sedang offline: %s", printerName), http.StatusInternalServerError)
			return
		}
		// jika err != nil: fallback assume online
	}

	// Kirim ke spooler (WritePrinter)
	if err := sendToPrinter(printerName, []byte(text)); err != nil {
		http.Error(w, fmt.Sprintf("Print failed: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Printed on %s", printerName)
}

// ----------------- Utilities -----------------

func addCors(w http.ResponseWriter, r *http.Request) {
	// Perhatikan: Access-Control-Allow-Origin set to * ; ubah jika ingin lebih aman
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	// Chrome PNA (Private Network Access) header
	w.Header().Set("Access-Control-Allow-Private-Network", "true")
}

// listPrinters: pake PowerShell Get-Printer -> ConvertTo-Json
// Mengembalikan slice nama printer (untuk shared -> \\HOST\Sharename)
func listPrinters() ([]string, error) {
	// Perintah PowerShell: ambil fields yang kita butuhkan
	ps := `Get-Printer | Select Name,Shared,ComputerName,ShareName,WorkOffline | ConvertTo-Json -Depth 2`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)

	out, err := cmd.Output()
	if err != nil {
		// fallback: coba WMIC (lebih jadul)
		return listPrintersWMIC()
	}

	// PowerShell bisa mengembalikan array atau single object
	outStr := strings.TrimSpace(string(out))
	if outStr == "" {
		return nil, fmt.Errorf("no output from powershell")
	}

	var printers []psPrinter
	// Try decode as array
	if err := json.Unmarshal(out, &printers); err != nil {
		// try single object
		var single psPrinter
		if err2 := json.Unmarshal(out, &single); err2 != nil {
			// fallback error
			return nil, fmt.Errorf("cannot parse powershell output: %v / %v", err, err2)
		}
		printers = append(printers, single)
	}

	result := make([]string, 0, len(printers))
	for _, p := range printers {
		nameLower := strings.ToLower(p.Name)
		// skip virtual printers kecuali shared ones
		if !p.Shared && (strings.Contains(nameLower, "pdf") || strings.Contains(nameLower, "xps") ||
			strings.Contains(nameLower, "onenote") || strings.Contains(nameLower, "fax") ||
			strings.Contains(nameLower, "writer")) {
			continue
		}

		// Jika shared, buat \\ComputerName\ShareName bila ShareName ada
		if p.Shared && p.ShareName != "" {
			full := `\\` + p.ComputerName + `\` + p.ShareName
			result = append(result, full)
		} else {
			result = append(result, p.Name)
		}
	}
	return result, nil
}

// Fallback: parse WMIC output (simple)
func listPrintersWMIC() ([]string, error) {
	cmd := exec.Command("cmd", "/c", "wmic printer get Name,WorkOffline /format:csv")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	res := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(strings.ToLower(line), "node,") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimSpace(parts[1])
		workOffline := strings.TrimSpace(parts[2])
		lname := strings.ToLower(name)
		if strings.Contains(lname, "pdf") || strings.Contains(lname, "xps") || strings.Contains(lname, "onenote") ||
			strings.Contains(lname, "fax") || strings.Contains(lname, "writer") {
			continue
		}
		if strings.Contains(strings.ToLower(workOffline), "true") {
			// skip offline printers
			continue
		}
		res = append(res, name)
	}
	return res, nil
}

// isPrinterOffline: cek WorkOffline via PowerShell untuk printer lokal
func isPrinterOffline(printerName string) (bool, error) {
	ps := fmt.Sprintf(`$p = Get-Printer -Name "%s" -ErrorAction SilentlyContinue; if ($p) { $p.WorkOffline } else { Write-Output "NOTFOUND" }`, escapeForPS(printerName))
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	s := strings.TrimSpace(strings.ToLower(string(out)))
	if s == "true" {
		return true, nil
	}
	if s == "false" {
		return false, nil
	}
	if strings.Contains(s, "notfound") {
		return false, fmt.Errorf("printer not found")
	}
	// unknown -> assume online
	return false, nil
}

func escapeForPS(s string) string {
	// simple escape for double quotes
	return strings.ReplaceAll(s, `"`, "`\"")
}

// ----------------- Winspool (WritePrinter) -----------------
// sendToPrinter sends raw bytes to the specified printer using Win32 API.
func sendToPrinter(printerName string, data []byte) error {
	// Load winspool library
	winspool := syscall.NewLazyDLL("winspool.drv")
	procOpenPrinter := winspool.NewProc("OpenPrinterW")
	procClosePrinter := winspool.NewProc("ClosePrinter")
	procStartDocPrinter := winspool.NewProc("StartDocPrinterW")
	procEndDocPrinter := winspool.NewProc("EndDocPrinter")
	procStartPagePrinter := winspool.NewProc("StartPagePrinter")
	procEndPagePrinter := winspool.NewProc("EndPagePrinter")
	procWritePrinter := winspool.NewProc("WritePrinter")

	// Convert printerName to UTF16
	pnUTF16, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return err
	}

	var hPrinter syscall.Handle
	r1, _, e1 := procOpenPrinter.Call(uintptr(unsafe.Pointer(pnUTF16)), uintptr(unsafe.Pointer(&hPrinter)), 0)
	if r1 == 0 {
		return fmt.Errorf("OpenPrinter failed: %v", e1)
	}
	defer procClosePrinter.Call(uintptr(hPrinter))

	// Prepare DOC_INFO_1 structure
	// typedef struct _DOC_INFO_1 {
	//   LPTSTR pDocName;
	//   LPTSTR pOutputFile;
	//   LPTSTR pDatatype;
	// } DOC_INFO_1, *PDOC_INFO_1;
	docName, _ := syscall.UTF16PtrFromString("PrintAgent Job")
	dataType, _ := syscall.UTF16PtrFromString("RAW")
	type docInfo1 struct {
		pDocName    *uint16
		pOutputFile *uint16
		pDatatype   *uint16
	}
	var di docInfo1
	di.pDocName = docName
	di.pOutputFile = nil
	di.pDatatype = dataType

	r1, _, e1 = procStartDocPrinter.Call(uintptr(hPrinter), uintptr(1), uintptr(unsafe.Pointer(&di)))
	if r1 == 0 {
		return fmt.Errorf("StartDocPrinter failed: %v", e1)
	}
	defer procEndDocPrinter.Call(uintptr(hPrinter))

	r1, _, e1 = procStartPagePrinter.Call(uintptr(hPrinter))
	if r1 == 0 {
		return fmt.Errorf("StartPagePrinter failed: %v", e1)
	}
	defer procEndPagePrinter.Call(uintptr(hPrinter))

	// WritePrinter expects pointer to data and length (DWORD)
	var written uint32
	var writtenPtr uintptr = uintptr(unsafe.Pointer(&written))
	r1, _, e1 = procWritePrinter.Call(uintptr(hPrinter), uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), writtenPtr)
	if r1 == 0 {
		return fmt.Errorf("WritePrinter failed: %v", e1)
	}
	// Optionally check written value
	_ = written

	return nil
}
