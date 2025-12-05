# FFmpeg GUI - Versione Go

Conversione della GUI FFmpeg da Python a Go usando Fyne.

## 📋 Requisiti

- **Go 1.21+** installato
- **FFmpeg** con supporto NVENC (GPU NVIDIA)
- **Windows** (testato su Windows 10/11)

## 🚀 Build Locale

### Metodo 1: Con Fyne CLI (Raccomandato - include icona)

```bash
# Installa dipendenze
go mod init ffmpeg-gui-go
go get fyne.io/fyne/v2
go mod tidy

# Installa Fyne CLI
go install fyne.io/fyne/v2/cmd/fyne@latest

# Build con icona
fyne package -os windows -icon icon.ico -name "FFmpeg-GUI" -release
```

### Metodo 2: Build Standard

```bash
# Installa rsrc per embed icona
go install github.com/akavel/rsrc@latest

# Crea file risorsa con icona
rsrc -ico icon.ico -o rsrc.syso

# Build
go build -ldflags "-s -w -H=windowsgui" -o FFmpeg-GUI.exe .
```

### Metodo 3: Build Semplice (senza icona)

```bash
go build -ldflags "-s -w -H=windowsgui" -o FFmpeg-GUI.exe .
```

## 📦 Build con GitHub Actions

1. Vai su **Actions** nel tuo repository
2. Seleziona **Build FFmpeg GUI**
3. Clicca **Run workflow**
4. Scarica l'artifact `FFmpeg-GUI-Windows`

Il workflow crea **due build**:
- **Build principale**: Usa Fyne CLI (con icona embedded)
- **Build alternativo**: Usa rsrc (fallback)

## 🎯 Utilizzo

1. **Posiziona `ffmpeg.exe`** nella stessa cartella dell'eseguibile o nel PATH di sistema
2. **Avvia** `FFmpeg-GUI.exe`
3. **Trascina** i file video nella lista
4. **Configura** codec, preset, qualità e scaling
5. **Avvia** la compressione

### Modalità Disponibili

#### 1. Compressione Standard
- **AV1**: Codec moderno, massima compressione
- **H265**: Ottimo compromesso qualità/dimensione
- **H264**: Massima compatibilità

#### 2. Crea Proxy
File ottimizzati per editing (576p, bassa latenza):
```
-hwaccel cuda -hwaccel_output_format cuda
-c:v av1_nvenc -vf scale_cuda=-2:576
-preset p1 -cq 0 -tune ll -g 30
```

#### 3. Modalità Manuale
Attiva **"Modifica Manuale"** per:
- Personalizzare completamente i comandi FFmpeg
- Usare `%%INPUT%%` e `%%OUTPUT%%` come placeholder
- Processare batch con comandi custom

## 🔧 Parametri FFmpeg (identici all'originale Python)

### Accelerazione Hardware
```
-hwaccel cuda -hwaccel_output_format cuda
```

### Encoding Standard (AV1/H265/H264)
```
-c:v [codec]_nvenc
-preset [p1-p7]
-rc vbr / vbr_hq
-cq [0-51]
-tune hq
-rc-lookahead 64
-spatial-aq 1
-temporal-aq 1
-g 120
-bf 2
-movflags +faststart
```

### Scaling (solo AV1)
```
-vf scale_cuda=-2:[height]
```

### Audio
```
-c:a copy  # Copia stream audio senza ricodifica
```

## 🐛 Troubleshooting

### L'eseguibile non si avvia
- Verifica che `ffmpeg.exe` sia disponibile
- Controlla che la GPU NVIDIA supporti NVENC
- Assicurati che i driver NVIDIA siano aggiornati

### Errore "FFmpeg non trovato"
Posiziona `ffmpeg.exe`:
1. Nella stessa cartella di `FFmpeg-GUI.exe`, OPPURE
2. In una cartella nel PATH di sistema

### L'icona non appare
- Usa il **Metodo 1** (Fyne CLI) per build locali
- Scarica l'artifact **principale** da GitHub Actions
- L'icona appare solo nel file `.exe`, non durante l'esecuzione

### Build fallisce su GitHub
- Verifica che `icon.ico` sia nel repository
- Controlla i log dell'action per errori specifici
- Il workflow ha due job: se uno fallisce, usa l'altro

## 📝 Note Tecniche

### Differenze da Python
- **GUI Framework**: CustomTkinter → Fyne
- **Performance**: Avvio più veloce, minore uso RAM
- **Distribuzione**: Eseguibile singolo (no Python runtime)
- **Compatibilità**: Identici comandi FFmpeg

### Ottimizzazioni
- `-ldflags "-s -w"`: Rimuove simboli debug (-40% dimensione)
- `-H=windowsgui`: Nasconde console Windows
- `syscall.SysProcAttr`: Nasconde console FFmpeg child process

## 📄 Licenza

Stesso progetto dell'originale Python.

## 🤝 Contributi

Pull request benvenute per:
- Supporto Linux/macOS
- Profili codec aggiuntivi
- Miglioramenti UI
- Bug fix

---

**Nota**: Questa versione Go mantiene **identica logica FFmpeg** dell'originale Python per garantire stessi risultati di encoding.
