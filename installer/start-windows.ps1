$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
& "$Root\installer\bin\NusaMedia-Installer-Windows-amd64.exe" --root "$Root"
