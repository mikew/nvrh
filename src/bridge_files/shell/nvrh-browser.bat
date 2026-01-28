@echo off
setlocal enabledelayedexpansion

set "URL=%%~1"

:: Escape backslashes
set "URL=!URL:\=\\!"

:: Escape double quotes
set "URL=!URL:"=\"!"

set "SOCKET_PATH=%s"

REM start "" nvim --server "%%SOCKET_PATH%%" --remote-expr "v:lua._G._nvrh.open_url(\"!URL!\")" > nul 2>&1
nvim --server "%%SOCKET_PATH%%" --remote-expr "v:lua._G._nvrh.open_url(\"!URL!\")" > nul 2>&1
