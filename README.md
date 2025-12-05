FFmpeg GUI (Go Version)

Versione riscritta in Go della GUI per FFmpeg. Molto più leggera, veloce e nativa.

Requisiti di Sviluppo

Installa Go.

Installa un compilatore C (necessario per Fyne):

Windows: Installa TDM-GCC o MSYS2.

Linux: sudo apt install gcc libgl1-mesa-dev xorg-dev

Compilazione Locale

Per ottenere l'eseguibile ottimizzato (piccolo e senza console):

# Inizializza il modulo (solo prima volta)
go mod init ffmpeg-gui-go
go get fyne.io/fyne/v2
go mod tidy

# Compila
go build -ldflags "-s -w -H=windowsgui" -o FFmpeg-GUI.exe


Note

L'eseguibile richiede ffmpeg.exe nella stessa cartella o nel PATH di sistema.

Supporta il Drag & Drop dei file.

Tema scuro con accenti verdi come l'originale.
