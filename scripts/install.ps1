$repo = "divyo-argha/octonote"
$repoUrl = "https://github.com/$repo.git"

if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Error "✗ Go is not installed. Install it from https://go.dev/dl/ and retry."
    exit 1
}

$tmpDir = [System.IO.Path]::GetTempPath() + "octonote-" + [Guid]::NewGuid().ToString()
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null
Set-Location $tmpDir

Write-Host "→ Cloning octonote..."
git clone --depth=1 $repoUrl octonote
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to clone repository"
    exit 1
}

Set-Location octonote

Write-Host "→ Building CLI..."
go build -trimpath -ldflags "-s -w" -o octonote.exe ./tui

Write-Host "→ Building GUI (this may take a moment)..."
if (-not (Get-Command "wails" -ErrorAction SilentlyContinue)) {
    Write-Host "→ Wails not found. Installing Wails v2..."
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
}
Set-Location gui
wails build -clean -ldflags "-s -w"
Set-Location ..

$binDir = "$env:USERPROFILE\.octonote\bin"
if (!(Test-Path $binDir)) {
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
}

Move-Item -Path "octonote.exe" -Destination "$binDir\octonote.exe" -Force
Move-Item -Path "gui\build\bin\octonote.exe" -Destination "$binDir\octonote-gui.exe" -Force -ErrorAction SilentlyContinue
Move-Item -Path "gui\build\bin\octonote-gui.exe" -Destination "$binDir\octonote-gui.exe" -Force -ErrorAction SilentlyContinue

Set-Location $env:USERPROFILE
Remove-Item -Recurse -Force $tmpDir

# Add path to User Env Path if not present
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$binDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$binDir", "User")
    $env:Path += ";$binDir"
    Write-Host "Added $binDir to your user PATH variable."
}

Write-Host "✓ Installation complete!"
Write-Host "Please restart your terminal to use the new commands."
Write-Host "Run 'octonote' for the terminal interface."
Write-Host "Run 'octonote-gui' for the desktop application."
