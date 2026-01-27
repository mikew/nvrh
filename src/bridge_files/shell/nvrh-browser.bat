@echo off
set SOCKET_PATH=%s

start "" nvim --server "%%SOCKET_PATH%%" --remote-send "<cmd>lua _G._nvrh.open_url([[%%1]])<cr>"
