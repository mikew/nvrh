@echo off
setlocal enabledelayedexpansion

set "SOCKET_PATH=%s"
set "LOCK_FILE=%%TEMP%%\nvrh-editor-%%RANDOM%%.lock"

set "FILE_PATH=%%~1"
set "LINE=%%~2"
set "COL=%%~3"

nvim --server "%%SOCKET_PATH%%" --remote-send "<cmd>lua _G._nvrh.edit_with_lock([[%%FILE_PATH%%]], [[%%LOCK_FILE%%]], [[%%LINE%%]], [[%%COL%%]])<cr>"

pathping 127.0.0.1 -n -q 1 -p 100 >nul

:WAIT
if exist "%%LOCK_FILE%%" (
  pathping 127.0.0.1 -n -q 1 -p 100 >nul
  goto WAIT
)

exit /b 0
