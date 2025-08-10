if nvrh_mode == "primary" then
  ---@param url string
  function _G._nvrh.open_url(url)
    for _, channel_id in ipairs(_G._nvrh.client_channels) do
      pcall(vim.rpcnotify, tonumber(channel_id), 'open-url', { url })
    end
  end

  vim.api.nvim_create_user_command(
    'NvrhOpenUrl',
    function(args)
      _G._nvrh.open_url(args.args)
    end,
    {
      nargs = 1,
      force = true,
    }
  )

  local original_open = vim.ui.open
  vim.ui.open = function(uri, opts)
    if type(uri) == 'string' and uri:match('^https?://') then
      _G._nvrh.open_url(uri)
      return nil, nil
    else
      return original_open(uri, opts)
    end
  end

  vim.env.BROWSER = browser_script_path
end
