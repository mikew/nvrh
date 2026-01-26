if should_initialize then
  ---@param filename string
  ---@param lock_path string
  ---@param line string
  ---@param col string
  function _G._nvrh.edit_with_lock(filename, lock_path, line, col)
    vim.cmd.tabedit(filename)
    local window = vim.api.nvim_get_current_win()

    if line ~= '' then
      if col == '' then
        col = '1'
      end

      vim.api.nvim_win_set_cursor(0, { tonumber(line), tonumber(col) - 1 })
    end

    local lock_file = io.open(lock_path, 'w')
    if lock_file then
      lock_file:write('')
      lock_file:close()
    end

    vim.api.nvim_create_autocmd('WinClosed', {
      callback = function(args)
        if args.match == tostring(window) then
          os.remove(lock_path)
        end
      end,
    })
  end

  vim.env.NVRH_EDITOR = editor_script_path
  vim.env.EDITOR = editor_script_path
  vim.env.VISUAL = editor_script_path
  vim.env.GIT_EDITOR = editor_script_path
  vim.env.LAUNCH_EDITOR = editor_script_path
end
