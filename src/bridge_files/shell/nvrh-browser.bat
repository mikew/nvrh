@echo off

set "URL=%~1"

:: Escape backslashes
set "URL=%URL:\=\\%"

:: Escape double quotes
set "URL=%URL:"=\"%"

set "SOCKET_PATH={{.SocketPath}}"

nvim --server "%SOCKET_PATH%" --remote-expr "v:lua._G._nvrh.open_url(\"%URL%\")" > nul 2>&1
