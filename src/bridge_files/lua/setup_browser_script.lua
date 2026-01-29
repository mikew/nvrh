if should_initialize then
  local script_contents = [====[
%s
]====]

  vim.fn.writefile(vim.fn.split(script_contents, '\n'), browser_script_path)

  if _G._nvrh.server_info.os ~= 'windows' then
    vim.fn.setfperm(browser_script_path, 'rwxr-xr-x')
  end
end
