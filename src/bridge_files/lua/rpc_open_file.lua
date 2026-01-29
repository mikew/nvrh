if should_initialize then
  ---@param filename string
  ---@param lock_path string
  ---@param line string
  ---@param col string
  function _G._nvrh.edit_with_lock(filename, lock_path, line, col)
    local ok = pcall(vim.cmd.tabedit, filename)
    if not ok then
      return
    end

    local window = vim.api.nvim_get_current_win()

    if line ~= '' and line ~= '-1' then
      if col == '' or col == '-1' then
        col = '1'
      end

      pcall(
        vim.api.nvim_win_set_cursor,
        window,
        { tonumber(line), tonumber(col) - 1 }
      )
    end

    local function cleanup_lock()
      pcall(os.remove, lock_path)
    end

    local lock_file, err = io.open(lock_path, 'w')
    if lock_file then
      vim.api.nvim_create_autocmd('WinClosed', {
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
