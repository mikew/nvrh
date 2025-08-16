@echo off
set SOCKET_PATH=%s

start "" nvim --server "%%SOCKET_PATH%%" --remote-expr "v:lua._nvrh.open_url('%%1')" >nul 2>&1
