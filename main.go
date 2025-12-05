package main

import (
	"bufio"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// --- CONFIGURAZIONE COLORI ---
var (
	ColorAppBg       = parseHex("#212121")
	ColorFrameBg     = parseHex("#2B2B2B")
	ColorInputBg     = parseHex("#1b1b1b")
	ColorAccent      = parseHex("#6BFF00")
	ColorAccentHover = parseHex("#A8E618")
	ColorStop        = parseHex("#880000")
	ColorText        = parseHex("#ffffff")
	ColorTextDark    = parseHex("#000000")
)

// --- TEMA PERSONALIZZATO ---
type myTheme struct{}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return ColorAppBg
	case theme.ColorNameInputBackground:
		return ColorInputBg
	case theme.ColorNameButton, theme.ColorNameOverlayBackground:
		return ColorFrameBg
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameSelection:
		return ColorAccent
	case theme.ColorNameForeground:
		return ColorText
	case theme.ColorNameScrollBar:
		return ColorAccent
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}
func (m myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}
func (m myTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func parseHex(s string) color.Color {
	c := color.RGBA{A: 0xff}
	if len(s) != 7 {
		return c
	}
	fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	return c
}

// --- STATO DELL'APPLICAZIONE ---
type AppState struct {
	window      fyne.Window
	app         fyne.App
	fileList    *widget.Entry
	console     *widget.Entry
	preview     *widget.Entry
	progressBar *widget.ProgressBar
	progressLbl *widget.Label
	btnStart    *widget.Button
	btnStop     *widget.Button

	comboCodec  *widget.Select
	comboPreset *widget.Select
	comboScale  *widget.Select
	sliderCQ    *widget.Slider
	labelCQ     *widget.Label
	checkManual *widget.Check

	isRunning bool
	cmd       *exec.Cmd
	stopChan  chan bool
	wg        sync.WaitGroup
	mu        sync.Mutex
}

func main() {
	// Previeni panic da chiudere l'applicazione
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Errore fatale: %v\n", r)
		}
	}()

	myApp := app.NewWithID("com.ffmpeg.gui")
	myApp.Settings().SetTheme(&myTheme{})

	w := myApp.NewWindow("FFmpeg GUI")
	w.Resize(fyne.NewSize(900, 750))
	w.CenterOnScreen()

	state := &AppState{
		window:   w,
		app:      myApp,
		stopChan: make(chan bool),
	}

	state.buildUI()

    // Verifica FFmpeg all'avvio
	// Il controllo viene eseguito subito e il dialogo (se necessario) 
	// verrà mostrato automaticamente sopra la finestra all'avvio.
	ffmpegPath := findFFmpeg()
	if ffmpegPath == "" {
		dialog.ShowInformation(
			"FFmpeg non trovato",
			"FFmpeg non è stato trovato nel sistema.\n\n"+
				"Per usare questa applicazione, devi:\n"+
				"1. Scaricare ffmpeg.exe da ffmpeg.org\n"+
				"2. Posizionarlo nella stessa cartella di questo programma\n"+
				"   OPPURE aggiungerlo al PATH di sistema\n\n"+
				"L'applicazione può essere usata una volta che FFmpeg è disponibile.",
			w,
		)
	}

	// Gestione Drag & Drop
	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		var sb strings.Builder
		sb.WriteString(state.fileList.Text)
		for _, u := range uris {
			path := u.Path()
			if !strings.Contains(state.fileList.Text, path) {
				if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") {
					sb.WriteString("\n")
				}
				sb.WriteString(path + "\n")
			}
		}
		state.fileList.SetText(sb.String())
		state.updatePreview()
	})

	// Gestione chiusura con stop processi
	w.SetCloseIntercept(func() {
		if state.isRunning {
			dialog.ShowConfirm(
				"Compressione in corso",
				"Una compressione è in corso. Vuoi davvero uscire?",
				func(confirmed bool) {
					if confirmed {
						state.stopCompression()
						w.Close()
					}
				},
				w,
			)
		} else {
			w.Close()
		}
	})

	w.ShowAndRun()
}

func (s *AppState) buildUI() {
	// --- SEZIONE INPUT FILE ---
	lblFiles := widget.NewLabel("File da Processare (Trascina qui i file):")
	s.fileList = widget.NewMultiLineEntry()
	s.fileList.SetPlaceHolder("Trascina qui i file o usa il pulsante Aggiungi...")
	s.fileList.OnChanged = func(str string) { s.updatePreview() }
	s.fileList.TextStyle.Monospace = true

	btnBrowse := widget.NewButton("Aggiungi File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				current := s.fileList.Text
				if len(current) > 0 && !strings.HasSuffix(current, "\n") {
					current += "\n"
				}
				s.fileList.SetText(current + reader.URI().Path() + "\n")
			}
		}, s.window)
		fd.Show()
	})
	btnBrowse.Importance = widget.HighImportance

	btnClear := widget.NewButton("Pulisci Lista", func() {
		s.fileList.SetText("")
	})

	filesContainer := container.NewBorder(
		lblFiles,
		container.NewGridWithColumns(2, btnBrowse, btnClear),
		nil, nil,
		container.NewScroll(s.fileList),
	)

	// --- SEZIONE OPZIONI ---
	s.comboCodec = widget.NewSelect([]string{"AV1", "H265", "H264", "Crea proxy"}, func(val string) {
		s.updateUIForCodec(val)
	})
	s.comboCodec.SetSelected("AV1")

	s.comboPreset = widget.NewSelect([]string{"p1", "p2", "p3", "p4", "p5", "p6", "p7"}, func(val string) { s.updatePreview() })
	s.comboPreset.SetSelected("p6")

	s.comboScale = widget.NewSelect([]string{"Nessuno", "4k", "2k", "1080p", "720p", "576p", "480p"}, func(val string) { s.updatePreview() })
	s.comboScale.SetSelected("Nessuno")

	s.labelCQ = widget.NewLabel("0")
	s.labelCQ.TextStyle = fyne.TextStyle{Bold: true}
	s.labelCQ.Alignment = fyne.TextAlignCenter

	s.sliderCQ = widget.NewSlider(0, 51)
	s.sliderCQ.Step = 1
	s.sliderCQ.Value = 0
	s.sliderCQ.OnChanged = func(f float64) {
		s.labelCQ.SetText(fmt.Sprintf("%.0f", f))
		s.updatePreview()
	}

	optionsGrid := container.NewGridWithColumns(4,
		container.NewVBox(widget.NewLabel("Codec"), s.comboCodec),
		container.NewVBox(widget.NewLabel("Preset"), s.comboPreset),
		container.NewVBox(widget.NewLabel("Qualità (CQ)"), container.NewBorder(nil, nil, nil, s.labelCQ, s.sliderCQ)),
		container.NewVBox(widget.NewLabel("Scaling"), s.comboScale),
	)

	// --- ANTEPRIMA ---
	s.checkManual = widget.NewCheck("Modifica Manuale", func(b bool) {
		if b {
			s.preview.Enable()
			s.comboCodec.Disable()
			s.comboPreset.Disable()
			s.comboScale.Disable()
			s.sliderCQ.Disable()
			s.updatePreviewManualTemplate()
			s.log("INFO: Modalità manuale. Usa %%INPUT%% e %%OUTPUT%% nel comando.\n")
		} else {
			s.preview.Disable()
			s.comboCodec.Enable()
			s.comboPreset.Enable()
			s.sliderCQ.Enable()
			s.updateUIForCodec(s.comboCodec.Selected)
			s.log("INFO: Modalità automatica ripristinata.\n")
		}
	})

	s.preview = widget.NewMultiLineEntry()
	s.preview.TextStyle.Monospace = true
	s.preview.Disable()
	s.preview.SetMinRowsVisible(3)

	previewContainer := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabelWithStyle("Anteprima Comando:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), s.checkManual),
		s.preview,
	)

	// --- CONSOLE E PROGRESSO ---
	s.console = widget.NewMultiLineEntry()
	s.console.TextStyle.Monospace = true
	s.console.Disable()
	s.console.SetMinRowsVisible(8)

	s.progressLbl = widget.NewLabel("In attesa...")
	s.progressBar = widget.NewProgressBar()

	s.btnStart = widget.NewButton("Avvia Compressione", s.startCompression)
	s.btnStart.Importance = widget.HighImportance

	s.btnStop = widget.NewButton("Ferma Compressione", s.stopCompression)
	s.btnStop.Disable()

	actions := container.NewGridWithColumns(2, s.btnStart, s.btnStop)

	// --- LAYOUT PRINCIPALE ---
	cardInput := widget.NewCard("", "", filesContainer)
	cardOptions := widget.NewCard("", "", container.NewVBox(optionsGrid, previewContainer))
	cardConsole := widget.NewCard("", "", container.NewBorder(
		container.NewHBox(widget.NewLabel("Console Output"), layout.NewSpacer(), widget.NewButton("Pulisci", func() { s.console.SetText("") })),
		nil, nil, nil,
		container.NewScroll(s.console),
	))

	content := container.NewVBox(
		cardInput,
		cardOptions,
		cardConsole,
		actions,
		s.progressLbl,
		s.progressBar,
	)

	s.window.SetContent(container.NewPadded(content))
	s.updatePreview()
}

// --- LOGICA ---

func (s *AppState) log(msg string) {
	s.console.SetText(s.console.Text + msg)
	s.console.Refresh()
}

func (s *AppState) updateUIForCodec(codec string) {
	if codec == "Crea proxy" {
		s.comboScale.Disable()
		s.comboPreset.Disable()
		s.sliderCQ.Disable()
		s.log("INFO: Modalità 'Crea proxy' selezionata. Le impostazioni sono fisse.\n")
	} else {
		s.comboPreset.Enable()
		s.sliderCQ.Enable()
		if codec == "AV1" {
			s.comboScale.Enable()
		} else {
			s.comboScale.Disable()
			s.comboScale.SetSelected("Nessuno")
		}
	}
	s.updatePreview()
}

func (s *AppState) generateOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	name := strings.TrimSuffix(filepath.Base(inputPath), ext)

	if s.comboCodec.Selected == "Crea proxy" {
		proxyDir := filepath.Join(dir, "proxy")
		return filepath.Join(proxyDir, filepath.Base(inputPath))
	}

	cq := int(s.sliderCQ.Value)
	suffix := fmt.Sprintf("_%s_CQ%d%s", s.comboCodec.Selected, cq, ext)
	return filepath.Join(dir, name+suffix)
}

func (s *AppState) updatePreview() {
	if s.checkManual.Checked {
		return
	}

	files := s.getFiles()
	if len(files) == 0 {
		s.preview.SetText("Aggiungi file per vedere l'anteprima...")
		return
	}

	cmd := s.buildCommand(files[0])
	s.preview.SetText(strings.Join(cmd, " "))
}

func (s *AppState) updatePreviewManualTemplate() {
	files := s.getFiles()
	if len(files) == 0 {
		s.preview.SetText("Aggiungi file per generare il template...")
		return
	}

	cmd := s.buildCommand(files[0])
	input := files[0]
	output := s.generateOutputPath(input)

	cmdStr := strings.Join(cmd, " ")
	cmdStr = strings.Replace(cmdStr, input, "%%INPUT%%", 1)
	cmdStr = strings.Replace(cmdStr, output, "%%OUTPUT%%", 1)

	s.preview.SetText(cmdStr)
}

func (s *AppState) getFiles() []string {
	lines := strings.Split(s.fileList.Text, "\n")
	var res []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			res = append(res, strings.TrimSpace(l))
		}
	}
	return res
}

func (s *AppState) buildCommand(inputFile string) []string {
	ffmpegPath := findFFmpeg()
	if ffmpegPath == "" {
		return []string{}
	}

	outputFile := s.generateOutputPath(inputFile)
	codec := s.comboCodec.Selected

	// Modalità Proxy (identica all'originale Python)
	if codec == "Crea proxy" {
		return []string{
			ffmpegPath, "-y", "-hwaccel", "cuda", "-hwaccel_output_format", "cuda",
			"-i", inputFile, "-c:v", "av1_nvenc", "-vf", "scale_cuda=-2:576",
			"-preset", "p1", "-cq", "0", "-tune", "ll", "-g", "30", "-c:a", "copy", outputFile,
		}
	}

	// Costruzione comando standard
	preset := s.comboPreset.Selected
	cq := fmt.Sprintf("%d", int(s.sliderCQ.Value))
	scale := s.comboScale.Selected

	args := []string{ffmpegPath, "-y", "-hwaccel", "cuda", "-hwaccel_output_format", "cuda", "-i", inputFile}

	// Configurazione codec (identica all'originale)
	enc := ""
	rc := ""
	switch codec {
	case "AV1":
		enc = "av1_nvenc"
		rc = "vbr"
	case "H265":
		enc = "hevc_nvenc"
		rc = "vbr_hq"
	case "H264":
		enc = "h264_nvenc"
		rc = "vbr_hq"
	}

	args = append(args, "-c:v", enc, "-preset", preset, "-rc", rc, "-b:v", "0", "-cq", cq)

	// Scaling (solo per AV1, identico all'originale)
	if codec == "AV1" && scale != "Nessuno" {
		height := "1080"
		switch scale {
		case "4k":
			height = "2160"
		case "2k":
			height = "1440"
		case "1080p":
			height = "1080"
		case "720p":
			height = "720"
		case "576p":
			height = "576"
		case "480p":
			height = "480"
		}
		args = append(args, "-vf", "scale_cuda=-2:"+height)
	}

	// Opzioni finali (identiche all'originale)
	args = append(args, "-tune", "hq", "-rc-lookahead", "64", "-spatial-aq", "1", 
		"-temporal-aq", "1", "-g", "120", "-bf", "2", "-movflags", "+faststart")
	args = append(args, "-c:a", "copy", outputFile)

	return args
}

func (s *AppState) startCompression() {
	files := s.getFiles()
	if len(files) == 0 {
		dialog.ShowError(fmt.Errorf("Nessun file selezionato"), s.window)
		return
	}

	ffmpegPath := findFFmpeg()
	if ffmpegPath == "" {
		dialog.ShowError(fmt.Errorf("FFmpeg non trovato!"), s.window)
		return
	}

	s.isRunning = true
	s.btnStart.Disable()
	s.btnStop.Enable()
	s.fileList.Disable()
	s.comboCodec.Disable()
	s.comboPreset.Disable()
	s.comboScale.Disable()
	s.sliderCQ.Disable()
	s.checkManual.Disable()

	go func() {
		totalFiles := len(files)
		manual := s.checkManual.Checked
		manualTemplate := s.preview.Text

		for i, f := range files {
			if !s.isRunning {
				break
			}

			outPath := s.generateOutputPath(f)
			os.MkdirAll(filepath.Dir(outPath), os.ModePerm)

			var cmdArgs []string
			if manual {
				if !strings.Contains(manualTemplate, "%%INPUT%%") || !strings.Contains(manualTemplate, "%%OUTPUT%%") {
					s.log("ERRORE: Il template deve contenere %%INPUT%% e %%OUTPUT%%\n")
					break
				}
				cmdStr := strings.Replace(manualTemplate, "%%INPUT%%", fmt.Sprintf(`"%s"`, f), -1)
				cmdStr = strings.Replace(cmdStr, "%%OUTPUT%%", fmt.Sprintf(`"%s"`, outPath), -1)
				cmdArgs = parseCommand(cmdStr)
				if len(cmdArgs) > 0 {
					cmdArgs[0] = ffmpegPath
				}
			} else {
				cmdArgs = s.buildCommand(f)
			}

			if len(cmdArgs) == 0 {
				s.log(fmt.Sprintf("ERRORE: Impossibile generare comando per %s\n", filepath.Base(f)))
				continue
			}

			s.runFFmpeg(cmdArgs, i+1, totalFiles, filepath.Base(f))
		}

		s.isRunning = false
		s.btnStart.Enable()
		s.btnStop.Disable()
		s.fileList.Enable()
		s.comboCodec.Enable()
		s.updateUIForCodec(s.comboCodec.Selected)
		s.checkManual.Enable()
		s.progressBar.SetValue(0)
		s.progressLbl.SetText("Pronto.")
		s.log("\n--- Coda Completata ---\n")
	}()
}

func (s *AppState) stopCompression() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isRunning = false
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	s.log("\n--- Interruzione richiesta dall'utente ---\n")
}

func (s *AppState) runFFmpeg(args []string, index, total int, filename string) {
	s.log(fmt.Sprintf("\n===================\nFile %d/%d: %s\n===================\n", index, total, filename))

	cmd := exec.Command(args[0], args[1:]...)
	hideWindowsConsole(cmd)

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		s.log(fmt.Sprintf("ERRORE avvio: %v\n", err))
		return
	}

	durationRegex := regexp.MustCompile(`Duration: (\d{2}:\d{2}:\d{2}\.\d{2})`)
	timeRegex := regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2})`)

	scanner := bufio.NewScanner(stderr)
	var durationSec float64

	for scanner.Scan() {
		if !s.isRunning {
			break
		}
		line := scanner.Text()

		if durationSec == 0 {
			if matches := durationRegex.FindStringSubmatch(line); len(matches) > 1 {
				durationSec = parseTime(matches[1])
			}
		}

		if matches := timeRegex.FindStringSubmatch(line); len(matches) > 1 && durationSec > 0 {
			currentSec := parseTime(matches[1])
			progress := currentSec / durationSec
			if progress > 1.0 {
				progress = 1.0
			}
			s.progressBar.SetValue(progress)
			s.progressLbl.SetText(fmt.Sprintf("Processando %d/%d (%.0f%%): %s", index, total, progress*100, filename))
		}
	}

	cmd.Wait()
	
	if s.isRunning {
		if cmd.ProcessState.ExitCode() == 0 {
			s.log(fmt.Sprintf("--- File %s completato con successo ---\n", filename))
		} else {
			s.log(fmt.Sprintf("--- ERRORE: Codice uscita %d per %s ---\n", cmd.ProcessState.ExitCode(), filename))
		}
	}
}

// --- UTILITY ---

func parseTime(t string) float64 {
	parts := strings.Split(t, ":")
	if len(parts) != 3 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	sParts := strings.Split(parts[2], ".")
	s, _ := strconv.Atoi(sParts[0])
	ms := 0
	if len(sParts) > 1 {
		ms, _ = strconv.Atoi(sParts[1])
	}
	return float64(h*3600+m*60+s) + float64(ms)/100.0
}

func findFFmpeg() string {
	fname := "ffmpeg"
	if runtime.GOOS == "windows" {
		fname = "ffmpeg.exe"
	}

	// 1. Cerca nella cartella dell'eseguibile
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		localPath := filepath.Join(exeDir, fname)
		if _, err := os.Stat(localPath); err == nil {
			return localPath
		}
	}

	// 2. Cerca nella cartella corrente
	if _, err := os.Stat(fname); err == nil {
		path, _ := filepath.Abs(fname)
		return path
	}

	// 3. Cerca nel PATH di sistema
	if path, err := exec.LookPath(fname); err == nil {
		return path
	}

	return ""
}

func hideWindowsConsole(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
	}
}

func parseCommand(cmdStr string) []string {
	// Parser semplice per comandi con quote
	var args []string
	var current strings.Builder
	inQuote := false

	for _, r := range cmdStr {
		switch r {
		case '"':
			inQuote = !inQuote
		case ' ':
			if inQuote {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
