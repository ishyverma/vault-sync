-- VaultSync — Neovim Lua plugin
-- Install with lazy.nvim:
--   { 'ishyverma/vault-sync', dir = 'vim' }

local M = {}

local notes_dir = vim.fn.expand('~/.vault/notes')

function M.push(filepath)
  local result = vim.fn.system('vault push ' .. vim.fn.shellescape(filepath) .. ' 2>&1')
  if vim.v.shell_error ~= 0 then
    vim.notify('✗ VaultSync: ' .. vim.trim(result), vim.log.levels.WARN)
    return false
  end
  vim.notify('✓ VaultSync: synced', vim.log.levels.INFO)
  return true
end

function M.status()
  local result = vim.fn.system('vault sync status 2>&1')
  vim.notify(result, vim.log.levels.INFO)
end

vim.api.nvim_create_autocmd('BufWritePost', {
  pattern = notes_dir .. '/*.md',
  callback = function(ev)
    M.push(ev.file)
  end,
})

vim.api.nvim_create_user_command('VaultSyncPush', function(opts)
  M.push(vim.fn.expand('%:p'))
end, {})

vim.api.nvim_create_user_command('VaultSyncStatus', function()
  M.status()
end, {})

return M
