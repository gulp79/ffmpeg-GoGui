# FFmpeg GUI - Versione Go

GUI moderna per FFmpeg con accelerazione hardware NVIDIA CUDA.

[![Build Status](https://github.com/gulp79/ffmpeg-gui-go/workflows/Build%20FFmpeg%20GUI/badge.svg)](https://github.com/gulp79/ffmpeg-gui-go/actions)

## 📋 Requisiti

### Runtime
- **Windows 10/11** (64-bit)
- **GPU NVIDIA** con supporto NVENC
- **FFmpeg** con CUDA support ([download qui](https://github.com/BtbN/FFmpeg-Builds/releases))

### Build
- **Go 1.23+** ([download](https://go.dev/dl/))
- **GCC** (per Fyne CGO): [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) o [MinGW-w64](https://www.mingw-w64.org/)

## 🚀 Build Rapida (Windows)

### Metodo 1: Script PowerShell (Raccomandato)

```powershell
# Build con Fyne Tools (include icona)
.\build.ps1 -Method fyne

# Build manuale (goversioninfo)
.\build.ps1 -Method manual

# Build semplice (veloce, senza icona)
.\build.ps1 -Method simple
```

### Metodo 2: Comandi Manuali

```bash
# Inizializza (prima volta)
go mod download
go mod tidy

# Metodo A: Con Fyne CLI (RACCOMANDATO - include icona)
go install fyne.io/tools/cmd/fyne@latest
fyne package -os windows -icon icon.ico -name FFmpeg-GUI -appID com.ffmpeg.gui -release

# Metodo B: Con goversioninfo (alternativa)
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
goversioninfo -icon=icon.ico
go build -ldflags "-s -w -H=windowsgui" -o FFmpeg-GUI.exe .

# Metodo C: Build diretta (senza icona)
go build -ldflags "-s -w -H=windowsgui" -o FFmpeg-GUI.exe .
```

## 📦 Build con GitHub Actions

1. **Committa i file** nel repository:
   ```bash
   git add main.go go.mod go.sum icon.ico .github/workflows/build.yml
   git commit -m "Setup FFmpeg GUI Go"
   git push
   ```

2. **Esegui la build**:
   - Vai su **Actions** → **Build FFmpeg GUI**
   - Clicca **Run workflow**
   - Seleziona il branch e clicca **Run workflow**

3. **Scarica l'eseguibile**:
   - Attendi il completamento (3-5 minuti)
   - Scarica uno degli artifact:
     - `FFmpeg-GUI-Windows` (Fyne, con icona) ⭐ **Raccomandato**
     - `FFmpeg-GUI-Windows-Manual` (goversioninfo, con icona)
     - `FFmpeg-GUI-Windows-Simple` (fallback, senza icona)

### ⚠️ Nota: go.sum e Cache

Il workflow ora:
- Genera automaticamente `go.sum` se mancante
- Usa Go 1.23.5 (ultima versione stabile)
- Crea **3 build diverse** per massima compatibilità
- Disabilita cache fino a quando `go.sum` non è committato

**Per abilitare la cache** (build più veloci):
1. Esegui il workflow una volta
2. Committa il `go.sum` generato
3. Rimuovi `cache: false` dal workflow

## 🎯 Utilizzo

### Setup Iniziale

1. **Scarica FFmpeg CUDA** da [BtbN/FFmpeg-Builds](https://github.com/BtbN/FFmpeg-Builds/releases)
   - Scegli: `ffmpeg-master-latest-win64-gpl-shared.zip`
   - Estrai e copia `ffmpeg.exe` nella cartella del programma

2. **Avvia** `FFmpeg-GUI.exe`

3. Se FFmpeg non viene trovato, apparirà un messaggio informativo

### Workflow Base

1. **Trascina** i file video nella lista (o usa "Aggiungi File")
2. **Seleziona** codec, preset e qualità
3. **Configura** scaling (solo AV1)
4. **Clicca** "Avvia Compressione"
5. **Monitora** il progresso nella console

### Modalità Disponibili

#### 🎬 Compressione Standard

| Codec | Uso Consigliato | Note |
|-------|-----------------|------|
| **AV1** | Massima compressione | Supporta scaling CUDA |
| **H265** | Bilanciato | Ottima compatibilità |
| **H264** | Universale | Massima compatibilità |

**Preset**: `p1` (veloce) → `p7` (lento, migliore qualità)  
**CQ**: `0` (auto) / `1` (massima qualità) → `51` (minima qualità)

#### 📹 Crea Proxy

Genera file ottimizzati per editing:
- Risoluzione: **576p**
- Codec: **AV1 NVENC**
- Preset: **p1** (latenza bassa)
- Output: `./proxy/[filename]`

Configurazione fissa:
```bash
-hwaccel cuda -hwaccel_output_format cuda
-c:v av1_nvenc -vf scale_cuda=-2:576
-preset p1 -cq 0 -tune ll -g 30
-c:a copy
```

#### ⚙️ Modalità Manuale

Per comandi FFmpeg personalizzati:

1. **Attiva** "Modifica Manuale"
2. **Modifica** il comando nell'anteprima
3. **Usa** placeholder:
   - `%%INPUT%%` → percorso file input
   - `%%OUTPUT%%` → percorso file output
4. **Avvia** la coda

Esempio:
```bash
ffmpeg -i %%INPUT%% -c:v libx264 -crf 23 %%OUTPUT%%
```

## 🔧 Parametri FFmpeg

### Accelerazione Hardware (NVIDIA)
```bash
-hwaccel cuda
-hwaccel_output_format cuda
```

### Encoding Video
```bash
-c:v [av1_nvenc|hevc_nvenc|h264_nvenc]
-preset [p1-p7]              # p1=veloce, p7=lento/migliore
-rc [vbr|vbr_hq]             # Rate control
-cq [0-51]                   # Qualità (0=auto, 1=max, 51=min)
-tune hq                     # Ottimizzazione qualità
-rc-lookahead 64             # Lookahead frames
-spatial-aq 1                # Adaptive quantization spaziale
-temporal-aq 1               # Adaptive quantization temporale
-g 120                       # GOP size (keyframe ogni 120 frame)
-bf 2                        # B-frames
-movflags +faststart         # Ottimizzazione streaming
```

### Scaling (Solo AV1)
```bash
-vf scale_cuda=-2:[height]   # Mantiene aspect ratio
```

Risoluzioni disponibili: **4k (2160p)**, **2k (1440p)**, **1080p**, **720p**, **576p**, **480p**

### Audio
```bash
-c:a copy                    # Copia senza ricodifica
```

## 🐛 Troubleshooting

### ❌ L'eseguibile non si avvia

**Sintomo**: Doppio click, nulla accade

**Cause possibili**:
1. **GCC mancante durante la build**
   ```bash
   # Installa TDM-GCC o MinGW-w64
   # Riesegui la build
   ```

2. **DLL mancanti**
   ```bash
   # Usa build "simple" o "manual" invece di Fyne
   ```

3. **Antivirus blocca l'eseguibile**
   ```bash
   # Aggiungi eccezione nell'antivirus
   # Rigenera con build firmata
   ```

**Debug**: Avvia da PowerShell per vedere errori
```powershell
.\FFmpeg-GUI.exe
```

### ❌ "FFmpeg non trovato"

**Soluzione**:
1. Scarica ffmpeg.exe da [BtbN/FFmpeg-Builds](https://github.com/BtbN/FFmpeg-Builds/releases)
2. Posiziona nella stessa cartella di FFmpeg-GUI.exe
3. **Oppure** aggiungi al PATH:
   ```powershell
   $env:PATH += ";C:\path\to\ffmpeg"
   ```

### ❌ Errore "CUDA not available"

**Causa**: Driver NVIDIA obsoleti o GPU non supporta NVENC

**Verifica**:
```bash
ffmpeg -hwaccels
# Deve mostrare "cuda" nella lista
```

**Soluzione**:
1. Aggiorna driver NVIDIA da [nvidia.com](https://www.nvidia.com/download/index.aspx)
2. Verifica che la GPU supporti NVENC ([lista compatibilità](https://developer.nvidia.com/video-encode-and-decode-gpu-support-matrix-new))

### ❌ Build GitHub fallisce

**Errore comune**: `missing appID parameter`

**Causa**: Versione vecchia di Fyne CLI

**Soluzione**: Il workflow aggiornato usa:
```yaml
go-version: '1.23.5'           # Ultima versione Go
fyne.io/tools/cmd/fyne@latest  # Nuovo Fyne CLI
-appID com.ffmpeg.gui          # Parametro richiesto
```

**Se persiste**:
- Usa artifact `FFmpeg-GUI-Windows-Manual`
- Controlla log dettagliati in Actions

### ⚠️ L'icona non appare

**Normale durante esecuzione**: L'icona appare solo nel file `.exe`, non nella finestra

**Per embedding icona**:
1. Usa build Fyne o Manual (non Simple)
2. Verifica che `icon.ico` esista nella root
3. L'icona appare in Explorer/Taskbar

### 🔍 Debug Avanzato

**Abilita console per vedere errori**:
```go
// In main.go, rimuovi temporaneamente:
// -H=windowsgui
go build -ldflags "-s -w" -o FFmpeg-GUI-Debug.exe .
```

**Avvia con log**:
```powershell
.\FFmpeg-GUI-Debug.exe 2>&1 | Tee-Object log.txt
```

## 📊 Confronto Build

| Metodo | Icona | Dimensione | Velocità | Compatibilità |
|--------|-------|------------|----------|---------------|
| **Fyne Package** | ✅ | ~25 MB | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **goversioninfo** | ✅ | ~23 MB | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Build Simple** | ❌ | ~22 MB | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |

**Raccomandazione**: Usa **Fyne Package** per distribuzione finale

## 🔄 Differenze da Versione Python

| Aspetto | Python (CustomTkinter) | Go (Fyne) |
|---------|------------------------|-----------|
| **Dimensione** | ~80 MB (con runtime) | ~25 MB |
| **Avvio** | 2-3 secondi | <1 secondo |
| **RAM** | ~150 MB | ~50 MB |
| **Distribuzione** | Richiede PyInstaller | Eseguibile singolo |
| **Comandi FFmpeg** | ✅ Identici | ✅ Identici |
| **Funzionalità** | ✅ Complete | ✅ Complete |

**Garanzia**: I comandi FFmpeg sono **identici** per risultati consistenti

## 📝 Struttura File

```
ffmpeg-gui-go/
├── main.go              # Codice principale
├── go.mod               # Dipendenze Go
├── go.sum               # Checksum dipendenze (auto-generato)
├── icon.ico             # Icona applicazione
├── build.ps1            # Script build locale
├── .gitignore           # File da ignorare
├── .github/
│   └── workflows/
│       └── build.yml    # GitHub Actions workflow
└── README.md            # Questo file
```

## 🤝 Contributi

Pull request benvenute per:
- ✨ Supporto Linux/macOS
- 🎨 Temi personalizzati
- 🔧 Profili codec aggiuntivi
- 🐛 Bug fix
- 📚 Miglioramenti documentazione

## 📄 Licenza

Stesso progetto dell'originale Python - [GPL-3.0](LICENSE)

## 🙏 Credits

- **Fyne**: Framework GUI cross-platform
- **FFmpeg**: Swiss army knife del multimedia
- **Versione Python originale**: [gulp79/ffmpeg-gui](https://github.com/gulp79/ffmpeg-gui)

---

**Build testato su**: Windows 10/11, Go 1.23.5, Fyne v2.5.3

**Hardware testato**: NVIDIA RTX 2060/3060/4070 con driver 560+

**Per supporto**: Apri un [Issue](https://github.com/gulp79/ffmpeg-gui-go/issues)
