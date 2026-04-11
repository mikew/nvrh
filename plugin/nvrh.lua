vim.api.nvim_create_user_command('NvrhOpen', function(args)
  local cmd = { 'nvrh', 'client', 'from-neovim', vim.v.servername }

  for _, arg in ipairs(args.fargs) do
    table.insert(cmd, arg)
  end

  vim.system(cmd, { detach = true })
end, {
  nargs = '*',
  force = true,
})
