$installDir = "$env:USERPROFILE\.uad-tui"
$exePath = "$installDir\uad-tui.exe"

New-Item -ItemType Directory -Force -Path $installDir | Out-Null

Invoke-WebRequest `
  -Uri "https://github.com/Adarshakarki/UAD-TUI/releases/download/0.5/uad-tui.exe" `
  -OutFile $exePath

Start-Process $exePath