@echo off

rem Set environment variables.
rem set FOO=bar
%s

rem Change directory if needed.
rem cd /d Documents
%s

rem Finally launch nvim.
start "" /WAIT %s
