# Compila o servidor e o cliente com o ícone embutido (o olho de Tiamath).
# Uso: .\build.ps1          (na pasta do projeto)
# Precisa do go-winres:  go install github.com/tc-hib/go-winres@latest
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
$winres = Join-Path (go env GOPATH) 'bin\go-winres.exe'

# O ícone é desenhado em código (cliente/story.go, windowIcon); grava em PNG.
go run ./cliente -gerar-icone cliente/winres/icone.png

if (Test-Path $winres) {
    Push-Location cliente
    & $winres simply --icon winres/icone.png --manifest gui --product-name AsciiMMO `
        --file-description 'AsciiMMO - cliente' --original-filename asciimmo-cliente.exe
    Pop-Location
    Push-Location servidor
    & $winres simply --icon ../cliente/winres/icone.png --manifest cli --product-name AsciiMMO `
        --file-description 'AsciiMMO - servidor' --original-filename asciimmo.exe
    Pop-Location
} else {
    Write-Warning "go-winres não encontrado: os .exe sairão sem ícone embutido."
}

go build -o asciimmo.exe ./servidor
go build -o asciimmo-cliente.exe ./cliente
Write-Host 'Pronto: asciimmo.exe e asciimmo-cliente.exe'
