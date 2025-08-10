local script_contents = [[
#!/bin/sh

SOCKET_PATH="%s"

exec nvim --server "$SOCKET_PATH" --remote-expr "v:lua._nvrh.open_url('$1')" > /dev/null
]]
script_contents = string.format(script_contents, socket_path)

vim.fn.writefile(vim.fn.split(script_contents, '\n'), browser_script_path)
os.execute('chmod +x ' .. browser_script_path)
