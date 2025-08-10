if nvrh_mode == "primary" and should_map_ports then
  local nvrh_port_scanner = {
    active_watchers = {},

    patterns = {
      -- "port 3000"
      "port%s+(%d+)",
      -- "localhost:3000"
      "localhost:(%d+)",
      -- "0.0.0.0:3000"
      "%d+%.%d+%.%d+%.%d+:(%d+)",
      -- ":3000" at start of line
      "^:(%d+)",
      -- ":3000" but avoid eslint errors (error in foo.tsx:3)
      "%s+:(%d+)",
      -- http://some.domain.com:3000 / https://some.domain.com:3000
      "https?://[^/]+:(%d+)",
    },
  }

  function nvrh_port_scanner.attach_port_watcher(bufnr)
    if vim.bo[bufnr].buftype ~= "terminal" then
      return
    end

    -- Already watching?
    if nvrh_port_scanner.active_watchers[bufnr] then
      return
    end

    local function on_lines(_, _, _, lastline, new_lastline, _)
      local lines = vim.api.nvim_buf_get_lines(bufnr, lastline, new_lastline, false)

      for _, line in ipairs(lines) do
        for _, pattern in ipairs(nvrh_port_scanner.patterns) do
          local port = string.match(line, pattern)
          if port then
            _G._nvrh.tunnel_port(port)
            break
          end
        end
      end
    end

    vim.api.nvim_buf_attach(bufnr, false, {
      on_lines = on_lines,
    })

    nvrh_port_scanner.active_watchers[bufnr] = true
  end

  -- Attach watcher on TermOpen
  vim.api.nvim_create_autocmd("TermOpen", {
    callback = function(args)
      nvrh_port_scanner.attach_port_watcher(args.buf)
    end,
  })
end
