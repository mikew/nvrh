if should_initialize then
  local script_contents = [====[
%s
]====]

  vim.fn.writefile(vim.fn.split(script_contents, '\n'), editor_script_path)

  if _G._nvrh.server_info.os ~= 'windows' then
    os.execute('chmod +x ' .. editor_script_path)
  end
end
