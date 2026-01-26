@echo off
setlocal enabledelayedexpansion

set "SOCKET_PATH=%s"
set "LOCK_FILE=%%TEMP%%\nvim_lock_%%RANDOM%%.txt"
set "FILE_PATH=%%~1"
set "LINE=%%~2"
set "COL=%%~3"

set "SAFE_PATH=%%FILE_PATH:\=/%%"
set "SAFE_LOCK=%%LOCK_FILE:\=/%%"

nvim --server "%%SOCKET_PATH%%" --remote-send "<cmd>lua _nvrh.edit_with_lock('%%SAFE_PATH%%', '%%SAFE_LOCK%%', '%%LINE%%', '%%COL%%')<cr>"

:WAIT
if exist "%%LOCK_FILE%%" (
    timeout /t 1 /nobreak >nul
    goto WAIT
)

exit /b 0
