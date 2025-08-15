if should_initialize then
  local script_contents = [[
%s
]]

  vim.fn.writefile(vim.fn.split(script_contents, '\n'), browser_script_path)
  os.execute('chmod +x ' .. browser_script_path)
end
