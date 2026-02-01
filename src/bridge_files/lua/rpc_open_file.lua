if should_initialize then
  ---@param filename string
  ---@param line? number
  ---@param col? number
  local function default_open_file(filename, line, col)
    vim.cmd.tabedit(filename)
    local window = vim.api.nvim_get_current_win()

    if line ~= nil then
      pcall(vim.api.nvim_win_set_cursor, window, { line, col or 0 })
    end
  end

  ---@param filename string
  ---@param lock_path string
  ---@param line_arg string
  ---@param col_arg string
  function _G._nvrh.edit_with_lock(filename, lock_path, line_arg, col_arg)
    ---@type number?
    local line = nil
    ---@type number?
    local col = nil

    if line_arg ~= '' and line_arg ~= '-1' then
      line = tonumber(line_arg)
    end

    if col_arg ~= '' and col_arg ~= '-1' then
      -- col_arg is 1-based, but neovim expects 0-based.
      col = tonumber(col_arg) - 1
    end

    local handler = _G.nvrh_open_file_handler or default_open_file
    handler(filename, line, col)

    local window = vim.api.nvim_get_current_win()

    local lock_file, err = io.open(lock_path, 'w')
    if lock_file then
      vim.api.nvim_create_autocmd('WinClosed', {
      local function cleanup_lock()
        pcall(os.remove, lock_path)
      end
        callback = function(args)
          if args.match == tostring(window) then
            cleanup_lock()
          end
        end,
      })

      vim.api.nvim_create_autocmd('VimLeavePre', {
        callback = function()
          cleanup_lock()
        end,
      })

      lock_file:write('')
      lock_file:close()
    else
      local message = 'Failed to create lock file at "'
        .. lock_path
        .. '": '
        .. tostring(err)

      if vim and vim.notify then
        vim.notify(message, vim.log.levels.ERROR)
      else
        error(message)
      end
    end
  end

  vim.env.NVRH_EDITOR = editor_script_path
  vim.env.EDITOR = editor_script_path
  vim.env.VISUAL = editor_script_path
  vim.env.GIT_EDITOR = editor_script_path
  vim.env.LAUNCH_EDITOR = editor_script_path
end
