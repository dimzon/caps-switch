$ErrorActionPreference = "Stop"

# Встраиваем иконку в .exe (ресурс RT_GROUP_ICON) и собираем тихий бинарь.
go run github.com/akavel/rsrc@latest -ico assets/icon.ico -o rsrc.syso -arch amd64
go build -trimpath -ldflags "-s -w -H windowsgui" -o caps-switch.exe .

Write-Host "Built caps-switch.exe"
