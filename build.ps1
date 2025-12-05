# Script di Build per FFmpeg GUI (Go)
# Uso: .\build.ps1 [-Method fyne|manual|simple]

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("fyne", "manual", "simple")]
    [string]$Method = "fyne"
)

$ErrorActionPreference = "Stop"

Write-Host "`n=== FFmpeg GUI - Build Script ===" -ForegroundColor Cyan
Write-Host "Metodo selezionato: $Method`n" -ForegroundColor Yellow

# Verifica Go installato
try {
    $goVersion = go version
    Write-Host "✓ Go trovato: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Go non trovato! Installa Go da https://go.dev/dl/" -ForegroundColor Red
    exit 1
}

# Inizializza modulo se necessario
if (-not (Test-Path "go.mod")) {
    Write-Host "Inizializzazione modulo Go..." -ForegroundColor Yellow
    go mod init github.com/gulp79/ffmpeg-gui-go
}

# Installa dipendenze
Write-Host "Installazione dipendenze..." -ForegroundColor Yellow
go get fyne.io/fyne/v2@latest
go mod tidy
Write-Host "✓ Dipendenze installate" -ForegroundColor Green

# Verifica icona
if (Test-Path "icon.ico") {
    Write-Host "✓ Icon.ico trovato" -ForegroundColor Green
    $iconExists = $true
} else {
    Write-Host "⚠ Icon.ico non trovato - build senza icona" -ForegroundColor Yellow
    $iconExists = $false
}

switch ($Method) {
    "fyne" {
        Write-Host "`nMetodo 1: Build con Fyne Tools (raccomandato)" -ForegroundColor Cyan
        
        # Installa Fyne Tools
        Write-Host "Installazione Fyne Tools..." -ForegroundColor Yellow
        go install fyne.io/tools/cmd/fyne@latest
        
        # Aggiungi Go bin al PATH temporaneamente
        $env:PATH += ";$env:USERPROFILE\go\bin"
        
        # Build
        Write-Host "Compilazione con Fyne Package..." -ForegroundColor Yellow
        
        if ($iconExists) {
            fyne package -os windows -icon icon.ico -name FFmpeg-GUI -appID com.ffmpeg.gui -release
        } else {
            fyne package -os windows -name FFmpeg-GUI -appID com.ffmpeg.gui -release
        }
        
        $outputFile = "FFmpeg-GUI.exe"
    }
    
    "manual" {
        Write-Host "`nMetodo 2: Build Manuale con goversioninfo" -ForegroundColor Cyan
        
        # Installa goversioninfo
        Write-Host "Installazione goversioninfo..." -ForegroundColor Yellow
        go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
        
        # Aggiungi Go bin al PATH temporaneamente
        $env:PATH += ";$env:USERPROFILE\go\bin"
        
        # Crea versioninfo.json
        Write-Host "Creazione versioninfo.json..." -ForegroundColor Yellow
        $versionInfo = @"
{
  "FixedFileInfo": {
    "FileVersion": {
      "Major": 1,
      "Minor": 0,
      "Patch": 0,
      "Build": 0
    },
    "ProductVersion": {
      "Major": 1,
      "Minor": 0,
      "Patch": 0,
      "Build": 0
    },
    "FileFlagsMask": "3f",
    "FileFlags ": "00",
    "FileOS": "040004",
    "FileType": "01",
    "FileSubType": "00"
  },
  "StringFileInfo": {
    "Comments": "FFmpeg GUI con accelerazione CUDA",
    "CompanyName": "gulp79",
    "FileDescription": "FFmpeg GUI",
    "FileVersion": "1.0.0.0",
    "InternalName": "ffmpeg-gui",
    "LegalCopyright": "© 2025",
    "OriginalFilename": "FFmpeg-GUI.exe",
    "ProductName": "FFmpeg GUI",
    "ProductVersion": "1.0.0.0"
  },
  "VarFileInfo": {
    "Translation": {
      "LangID": "0409",
      "CharsetID": "04B0"
    }
  },
  "IconPath": "icon.ico",
  "ManifestPath": ""
}
"@
        $versionInfo | Out-File -FilePath "versioninfo.json" -Encoding UTF8
        
        # Genera resource.syso
        Write-Host "Generazione resource.syso..." -ForegroundColor Yellow
        if ($iconExists) {
            goversioninfo -icon=icon.ico
        } else {
            goversioninfo
        }
        
        # Build
        Write-Host "Compilazione..." -ForegroundColor Yellow
        go build -v -ldflags "-s -w -H=windowsgui" -o FFmpeg-GUI.exe .
        
        # Pulizia
        Remove-Item "versioninfo.json" -ErrorAction SilentlyContinue
        Remove-Item "resource.syso" -ErrorAction SilentlyContinue
        
        $outputFile = "FFmpeg-GUI.exe"
    }
    
    "simple" {
        Write-Host "`nMetodo 3: Build Semplice (veloce, senza icona)" -ForegroundColor Cyan
        
        Write-Host "Compilazione..." -ForegroundColor Yellow
        go build -v -ldflags "-s -w -H=windowsgui" -o FFmpeg-GUI.exe .
        
        $outputFile = "FFmpeg-GUI.exe"
    }
}

# Verifica output
if (Test-Path $outputFile) {
    $fileInfo = Get-Item $outputFile
    $sizeMB = [math]::Round($fileInfo.Length / 1MB, 2)
    
    Write-Host "`n✓ BUILD COMPLETATA!" -ForegroundColor Green
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Green
    Write-Host "File: $($fileInfo.Name)" -ForegroundColor White
    Write-Host "Dimensione: $sizeMB MB" -ForegroundColor White
    Write-Host "Percorso: $($fileInfo.FullName)" -ForegroundColor White
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Green
    
    # Verifica PE header
    $bytes = [System.IO.File]::ReadAllBytes($outputFile)
    if ($bytes[0] -eq 0x4D -and $bytes[1] -eq 0x5A) {
        Write-Host "✓ Eseguibile valido (PE format)" -ForegroundColor Green
    }
    
    Write-Host "`nPer testare: .$outputFile" -ForegroundColor Cyan
    Write-Host "Ricorda: ffmpeg.exe deve essere nella stessa cartella o nel PATH`n" -ForegroundColor Yellow
    
} else {
    Write-Host "`n✗ BUILD FALLITA!" -ForegroundColor Red
    Write-Host "L'eseguibile non è stato creato. Controlla gli errori sopra.`n" -ForegroundColor Red
    exit 1
}
