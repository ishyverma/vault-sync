-- VaultSync — Neovim Lua plugin
-- Install with lazy.nvim:
--   { 'ishyverma/vault-sync', dir = 'vim' }
--
-- Statusline integration (add to your statusline config):
--   require('vault').statusline()
--
-- Or with lualine/nvim-statusline:
--   local vault = require('vault')
--   table.insert(sections.lualine_x, { vault.statusline })

local M = {}

local notes_dir = vim.fn.expand('~/.vault/notes')
local vault_bin = 'vault'

function M.push(filepath)
  local result = vim.fn.system(vault_bin .. ' push ' .. vim.fn.shellescape(filepath) .. ' 2>&1')
  if vim.v.shell_error ~= 0 then
    vim.notify('✗ VaultSync: ' .. vim.trim(result), vim.log.levels.WARN)
    return false
  end
  vim.notify('✓ VaultSync: synced', vim.log.levels.INFO)
  return true
end

function M.status()
  local result = vim.fn.system(vault_bin .. ' sync status 2>&1')
  vim.notify(result, vim.log.levels.INFO)
end

-- Returns a short status string for statusline integration.
function M.statusline()
  local file = vim.fn.expand('%:p')
  if not file:match(notes_dir) then
    return ''
  end
  local ok, result = pcall(vim.fn.system, vault_bin .. ' sync status --json 2>/dev/null')
  if not ok or vim.v.shell_error ~= 0 then
    return ''
  end
  if result:find('"synced"') then
    return ' ✓'
  elseif result:find('"conflict"') then
    return ' ⚠'
  elseif result:find('"failed"') then
    return ' ✗'
  elseif result:find('"pending"') then
    return ' ⟳'
  end
  return ''
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
