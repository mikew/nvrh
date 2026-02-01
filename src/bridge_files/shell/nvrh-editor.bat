@echo off

set "FILE_PATH=%~1"
set "LINE_NUMBER=%~2"
set "COLUMN_NUMBER=%~3"

:: Seems to be needed in cmd, otherwise they get passed as `\=\\`
if "%LINE_NUMBER%"=="" set "LINE_NUMBER=-1"
if "%COLUMN_NUMBER%"=="" set "COLUMN_NUMBER=-1"

:: Escape backslashes
set "FILE_PATH=%FILE_PATH:\=\\%"
set "LINE_NUMBER=%LINE_NUMBER:\=\\%"
set "COLUMN_NUMBER=%COLUMN_NUMBER:\=\\%"

:: Escape double quotes
set "FILE_PATH=%FILE_PATH:"=\"%"
set "LINE_NUMBER=%LINE_NUMBER:"=\"%"
set "COLUMN_NUMBER=%COLUMN_NUMBER:"=\"%"

set "SOCKET_PATH={{.SocketPath}}"
set "LOCK_FILE=%TEMP%\nvrh-editor-%RANDOM%.lock"
set "LOCK_FILE=%LOCK_FILE:\=\\%"

nvim --server "%SOCKET_PATH%" --remote-expr "v:lua._G._nvrh.edit_with_lock(\"%FILE_PATH%\", \"%LOCK_FILE%\", \"%LINE_NUMBER%\", \"%COLUMN_NUMBER%\")" > nul 2>&1

pathping 127.0.0.1 -n -q 1 -p 100 >nul

:WAIT
if exist "%LOCK_FILE%" (
  pathping 127.0.0.1 -n -q 1 -p 100 >nul
  goto WAIT
)

exit /b 0
