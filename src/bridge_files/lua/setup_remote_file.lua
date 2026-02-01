local file_path, file_contents, permission = ...

local file, err = io.open(file_path, 'w')
if file then
  file:write(file_contents)
  file:close()
else
  vim.print('Error opening file: ' .. err)
end

if permission and permission ~= '' then
  vim.fn.setfperm(file_path, permission)
end

permission = nil
file_contents = nil
file_path = nil
