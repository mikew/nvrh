@echo off
setlocal enabledelayedexpansion

set "FILE_PATH=%%~1"
set "LINE=%%~2"
set "COL=%%~3"

:: Escape backslashes
set "FILE_PATH=!FILE_PATH:\=\\!"
set "LINE_NUMBER=!LINE_NUMBER:\=\\!"
set "COLUMN_NUMBER=!COLUMN_NUMBER:\=\\!"

:: Escape double quotes
set "FILE_PATH=!FILE_PATH:"=\"!"
set "LINE_NUMBER=!LINE_NUMBER:"=\"!"
set "COLUMN_NUMBER=!COLUMN_NUMBER:"=\"!"

set "SOCKET_PATH=%s"
set "LOCK_FILE=%%TEMP%%\nvrh-editor-%%RANDOM%%.lock"
set "LOCK_FILE=!FILE_PATH:\=\\!"

nvim --server "%%SOCKET_PATH%%" --remote-expr "v:lua._G._nvrh.edit_with_lock(\"!FILE_PATH!\", \"!LOCK_FILE!\", \"!LINE!\", \"!COL!\")" > nul 2>&1

pathping 127.0.0.1 -n -q 1 -p 100 >nul

:WAIT
if exist "%%LOCK_FILE%%" (
  pathping 127.0.0.1 -n -q 1 -p 100 >nul
  goto WAIT
)

exit /b 0
