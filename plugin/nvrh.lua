--- @param args string[]
local function nvrh_connect(args)
  local cmd = { 'nvrh', 'client', 'from-neovim', vim.v.servername }

  for _, arg in ipairs(args) do
    table.insert(cmd, arg)
  end

  vim.system(cmd, { detach = true })
end

vim.api.nvim_create_user_command('NvrhConnect', function(args)
  nvrh_connect(args.fargs)
end, {
  nargs = '*',
  force = true,
})
