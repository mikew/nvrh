--- @param args string[]
local function nvrh_connect(args)
  if _G._nvrh then
    local hint_cmd = { 'nvrh', 'client', 'open' }
    for _, arg in ipairs(args) do
      table.insert(hint_cmd, arg)
    end

    vim.notify(
      'Already connected to nvrh, please manually run `'
        .. table.concat(hint_cmd, ' ')
        .. '` in your terminal to start a new session.',
      vim.log.levels.WARN
    )
    return
  end

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
