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
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/* ---------------------------------------------------------
   COLORI E TEMA
--------------------------------------------------------- */

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

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource  { return theme.DefaultTheme().Icon(name) }
func (m myTheme) Font(style fyne.TextStyle) fyne.Resource     { return theme.DefaultTheme().Font(style) }
func (m myTheme) Size(name fyne.ThemeSizeName) float32        { return theme.DefaultTheme().Size(name) }

func parseHex(s string) color.Color {
	c := color.RGBA{A: 0xff}
	if len(s) != 7 {
		return c
	}
	fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	return c
}

/* ---------------------------------------------------------
   STATO APPLICAZIONE
--------------------------------------------------------- */

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

/* ---------------------------------------------------------
   MAIN
--------------------------------------------------------- */

func main() {

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

	// Verifica FFmpeg dopo il caricamento della UI (altrimenti crash)
	go func() {
		time.Sleep(250 * time.Millisecond)
		if findFFmpeg() == "" {
			w.QueueUpdate(func() {
				dialog.ShowInformation(
					"FFmpeg non trovato",
					"FFmpeg non è stato trovato.\nScarica ffmpeg.exe e mettilo nella stessa cartella.",
					w,
				)
			})
		}
	}()

	// Drag & drop
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

	// Chiusura sicura
	w.SetCloseIntercept(func() {
		if state.isRunning {
			dialog.ShowConfirm("Compressione in corso", "Vuoi davvero uscire?",
				func(confirmed bool) {
					if confirmed {
						state.stopCompression()
						w.Close()
					}
				}, w)
		} else {
			w.Close()
		}
	})

	w.ShowAndRun()
}

/* ---------------------------------------------------------
   COSTRUZIONE UI
--------------------------------------------------------- */

func (s *AppState) buildUI() {

	lblFiles := widget.NewLabel("File da Processare (Drag & Drop):")
	s.fileList = widget.NewMultiLineEntry()
	s.fileList.SetPlaceHolder("Trascina qui i file...")
	s.fileList.TextStyle.Monospace = true
	s.fileList.OnChanged = func(_ string) { s.updatePreview() }

	btnBrowse := widget.NewButton("Aggiungi File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				cur := s.fileList.Text
				if cur != "" && !strings.HasSuffix(cur, "\n") {
					cur += "\n"
				}
				s.fileList.SetText(cur + reader.URI().Path() + "\n")
			}
		}, s.window)
		fd.Show()
	})

	btnClear := widget.NewButton("Pulisci Lista", func() { s.fileList.SetText("") })

	filesContainer := container.NewBorder(
		lblFiles,
		container.NewGridWithColumns(2, btnBrowse, btnClear),
		nil, nil,
		container.NewScroll(s.fileList),
	)

	s.comboCodec = widget.NewSelect([]string{"AV1", "H265", "H264", "Crea proxy"},
		func(val string) { s.updateUIForCodec(val) })
	s.comboCodec.SetSelected("AV1")

	s.comboPreset = widget.NewSelect([]string{"p1", "p2", "p3", "p4", "p5", "p6", "p7"},
		func(_ string) { s.updatePreview() })
	s.comboPreset.SetSelected("p6")

	s.comboScale = widget.NewSelect([]string{"Nessuno", "4k", "2k", "1080p", "720p", "576p", "480p"},
		func(_ string) { s.updatePreview() })
	s.comboScale.SetSelected("Nessuno")

	s.labelCQ = widget.NewLabel("0")
	s.labelCQ.TextStyle = fyne.TextStyle{Bold: true}
	s.labelCQ.Alignment = fyne.TextAlignCenter

	s.sliderCQ = widget.NewSlider(0, 51)
	s.sliderCQ.Step = 1
	s.sliderCQ.OnChanged = func(f float64) {
		s.labelCQ.SetText(fmt.Sprintf("%d", int(f)))
		s.updatePreview()
	}

	optionsGrid := container.NewGridWithColumns(4,
		container.NewVBox(widget.NewLabel("Codec"), s.comboCodec),
		container.NewVBox(widget.NewLabel("Preset"), s.comboPreset),
		container.NewVBox(widget.NewLabel("Qualità (CQ)"),
			container.NewBorder(nil, nil, nil, s.labelCQ, s.sliderCQ)),
		container.NewVBox(widget.NewLabel("Scaling"), s.comboScale),
	)

	s.checkManual = widget.NewCheck("Modifica Manuale", func(b bool) {
		if b {
			s.preview.Enable()
			s.comboCodec.Disable()
			s.comboPreset.Disable()
			s.comboScale.Disable()
			s.sliderCQ.Disable()
			s.updatePreviewManualTemplate()
		} else {
			s.preview.Disable()
			s.comboCodec.Enable()
			s.comboPreset.Enable()
			s.comboScale.Enable()
			s.sliderCQ.Enable()
			s.updateUIForCodec(s.comboCodec.Selected)
		}
	})

	s.preview = widget.NewMultiLineEntry()
	s.preview.Disable()
	s.preview.TextStyle.Monospace = true

	previewContainer := container.NewVBox(
		container.NewBorder(nil, nil,
			widget.NewLabelWithStyle("Anteprima Comando:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			s.checkManual),
		s.preview,
	)

	s.console = widget.NewMultiLineEntry()
	s.console.TextStyle.Monospace = true
	s.console.Disable()

	s.progressLbl = widget.NewLabel("In attesa...")
	s.progressBar = widget.NewProgressBar()

	s.btnStart = widget.NewButton("Avvia Compressione", s.startCompression)
	s.btnStop = widget.NewButton("Ferma Compressione", s.stopCompression)
	s.btnStop.Disable()

	cardInput := widget.NewCard("", "", filesContainer)
	cardOptions := widget.NewCard("", "", container.NewVBox(optionsGrid, previewContainer))
	cardConsole := widget.NewCard("", "",
		container.NewBorder(
			container.NewHBox(widget.NewLabel("Console Output"),
				layout.NewSpacer(),
				widget.NewButton("Pulisci", func() { s.console.SetText("") })),
			nil, nil, nil,
			container.NewScroll(s.console),
		),
	)

	content := container.NewVBox(
		cardInput,
		cardOptions,
		cardConsole,
		container.NewGridWithColumns(2, s.btnStart, s.btnStop),
		s.progressLbl,
		s.progressBar,
	)

	s.window.SetContent(container.NewPadded(content))
	s.updatePreview()
}

/* ---------------------------------------------------------
   LOGICA COMPRESSIONE
--------------------------------------------------------- */

func (s *AppState) log(msg string) {
	s.console.SetText(s.console.Text + msg)
	s.console.Refresh()
}

func (s *AppState) updateUIForCodec(codec string) {
	if codec == "Crea proxy" {
		s.comboScale.Disable()
		s.comboPreset.Disable()
		s.sliderCQ.Disable()
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

func (s *AppState) generateOutputPath(input string) string {
	dir := filepath.Dir(input)
	ext := filepath.Ext(input)
	name := strings.TrimSuffix(filepath.Base(input), ext)

	if s.comboCodec.Selected == "Crea proxy" {
		return filepath.Join(dir, "proxy", filepath.Base(input))
	}

	cq := int(s.sliderCQ.Value)
	return filepath.Join(dir, fmt.Sprintf("%s_%s_CQ%d%s", name, s.comboCodec.Selected, cq, ext))
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
	s.preview.SetText(strings.Join(s.buildCommand(files[0]), " "))
}

func (s *AppState) updatePreviewManualTemplate() {
	files := s.getFiles()
	if len(files) == 0 {
		s.preview.SetText("Aggiungi file per generare template...")
		return
	}
	cmd := s.buildCommand(files[0])
	in := files[0]
	out := s.generateOutputPath(in)
	cmdStr := strings.Join(cmd, " ")
	cmdStr = strings.Replace(cmdStr, in, "%%INPUT%%", 1)
	cmdStr = strings.Replace(cmdStr, out, "%%OUTPUT%%", 1)
	s.preview.SetText(cmdStr)
}

func (s *AppState) getFiles() []string {
	lines := strings.Split(s.fileList.Text, "\n")
	out := []string{}
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func (s *AppState) buildCommand(input string) []string {
	ffmpeg := findFFmpeg()
	if ffmpeg == "" {
		return []string{}
	}

	output := s.generateOutputPath(input)
	codec := s.comboCodec.Selected

	if codec == "Crea proxy" {
		return []string{
			ffmpeg, "-y", "-hwaccel", "cuda", "-hwaccel_output_format", "cuda",
			"-i", input,
			"-c:v", "av1_nvenc",
			"-vf", "scale_cuda=-2:576",
			"-preset", "p1", "-cq", "0", "-tune", "ll", "-g", "30",
			"-c:a", "copy",
			output,
		}
	}

	preset := s.comboPreset.Selected
	cq := fmt.Sprintf("%d", int(s.sliderCQ.Value))
	scale := s.comboScale.Selected

	args := []string{
		ffmpeg, "-y", "-hwaccel", "cuda", "-hwaccel_output_format", "cuda",
		"-i", input,
	}

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

	if codec == "AV1" && scale != "Nessuno" {
		height := map[string]string{
			"4k": "2160", "2k": "1440", "1080p": "1080",
			"720p": "720", "576p": "576", "480p": "480",
		}[scale]
		args = append(args, "-vf", "scale_cuda=-2:"+height)
	}

	args = append(args,
		"-tune", "hq",
		"-rc-lookahead", "64",
		"-spatial-aq", "1",
		"-temporal-aq", "1",
		"-g", "120",
		"-bf", "2",
		"-movflags", "+faststart",
		"-c:a", "copy",
		output,
	)

	return args
}

func (s *AppState) startCompression() {
	files := s.getFiles()
	if len(files) == 0 {
		dialog.ShowError(fmt.Errorf("Nessun file selezionato"), s.window)
		return
	}
	if findFFmpeg() == "" {
		dialog.ShowError(fmt.Errorf("FFmpeg non trovato"), s.window)
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
		total := len(files)
		manual := s.checkManual.Checked
		template := s.preview.Text

		for i, f := range files {
			if !s.isRunning {
				break
			}

			out := s.generateOutputPath(f)
			_ = os.MkdirAll(filepath.Dir(out), 0755)

			var args []string
			if manual {
				if !strings.Contains(template, "%%INPUT%%") ||
					!strings.Contains(template, "%%OUTPUT%%") {

					s.log("ERRORE: Template senza %%INPUT%% o %%OUTPUT%%\n")
					break
				}
				cmdStr := strings.ReplaceAll(template, "%%INPUT%%", `"`+f+`"`)
				cmdStr = strings.ReplaceAll(cmdStr, "%%OUTPUT%%", `"`+out+`"`)
				args = parseCommand(cmdStr)
				if len(args) > 0 {
					args[0] = findFFmpeg()
				}
			} else {
				args = s.buildCommand(f)
			}

			if len(args) == 0 {
				s.log("ERRORE: impossibile generare comando\n")
				continue
			}

			s.runFFmpeg(args, i+1, total, filepath.Base(f))
		}

		// UI final
s.window.QueueUpdate(func() {
    s.progressBar.SetValue(p)
    s.progressLbl.SetText(
        fmt.Sprintf("File %d/%d (%.0f%%): %s",
            index, total, p*100, filename))
})
	}()
}

func (s *AppState) stopCompression() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isRunning = false
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Signal(syscall.SIGINT)
		time.AfterFunc(1500*time.Millisecond, func() {
			if s.cmd != nil && s.cmd.Process != nil {
				_ = s.cmd.Process.Kill()
			}
		})
	}
	s.log("--- Interruzione richiesta ---\n")
}

func (s *AppState) runFFmpeg(args []string, index, total int, filename string) {

	s.log(fmt.Sprintf("\n===== File %d/%d: %s =====\n", index, total, filename))

	cmd := exec.Command(args[0], args[1:]...)

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		s.log("ERRORE avvio: " + err.Error())
		return
	}

	durationRe := regexp.MustCompile(`Duration: (\d{2}:\d{2}:\d{2}\.\d{2})`)
	timeRe := regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2})`)

	sc := bufio.NewScanner(stderr)
	var duration float64

	for sc.Scan() {
		if !s.isRunning {
			break
		}

		line := sc.Text()

		if duration == 0 {
			m := durationRe.FindStringSubmatch(line)
			if len(m) > 1 {
				duration = parseTime(m[1])
			}
		}

		m := timeRe.FindStringSubmatch(line)
		if len(m) > 1 && duration > 0 {
			curr := parseTime(m[1])
			p := curr / duration
			if p > 1 {
				p = 1
			}
			s.window.Canvas().Invoke(func() {
				s.progressBar.SetValue(p)
				s.progressLbl.SetText(
					fmt.Sprintf("File %d/%d (%.0f%%): %s",
						index, total, p*100, filename))
			})
		}

		s.window.QueueUpdate(func() {s.log(line + "\n")})
	}

	_ = cmd.Wait()

	if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 0 {
		s.log("✔ Completato\n")
	} else {
		s.log("✖ Errore durante la compressione\n")
	}
}

/* ---------------------------------------------------------
   UTILITY
--------------------------------------------------------- */

func parseTime(t string) float64 {
	parts := strings.Split(t, ":")
	if len(parts) != 3 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	secParts := strings.Split(parts[2], ".")
	s, _ := strconv.Atoi(secParts[0])
	ms := 0
	if len(secParts) > 1 {
		ms, _ = strconv.Atoi(secParts[1])
	}
	return float64(h*3600+m*60+s) + float64(ms)/100
}

func findFFmpeg() string {
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}

	// Cartella eseguibile
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Cartella corrente
	if _, err := os.Stat(name); err == nil {
		p, _ := filepath.Abs(name)
		return p
	}

	// PATH
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	return ""
}

func parseCommand(str string) []string {
	var args []string
	var cur strings.Builder
	inQuotes := false

	for _, r := range str {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if inQuotes {
				cur.WriteRune(r)
			} else if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}

	if cur.Len() > 0 {
		args = append(args, cur.String())
	}

	return args
}
