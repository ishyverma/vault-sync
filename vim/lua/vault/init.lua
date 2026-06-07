-- VaultSync Neovim Plugin
-- Install with lazy.nvim:
--   {
--     'ishyverma/vault-sync',
--     dir = 'vim',
--     config = function()
--       vim.cmd('VaultSyncSetup')
--     end,
--   }

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

function M.statusline()
  local file = vim.fn.expand('%:p')
  if not file:match(notes_dir) then
    return ''
  end
  local basename = vim.fn.fnamemodify(file, ':t')
  local ok, result = pcall(vim.fn.system, vault_bin .. ' sync status --json 2>/dev/null')
  if not ok or vim.v.shell_error ~= 0 then
    return ''
  end
  local ok2, data = pcall(vim.fn.json_decode, result)
  if not ok2 or type(data) ~= 'table' then
    return ''
  end
  local entries = data.entries or data
  for _, entry in ipairs(entries) do
    if entry.note == basename or entry.note == basename:gsub('%.md$', '') then
      if entry.status == 'synced' then return ' ✓'
      elseif entry.status == 'conflict' then return ' ⚠'
      elseif entry.status == 'failed' then return ' ✗'
      elseif entry.status == 'pending' then return ' ⟳'
      end
    end
  end
  return ' ○'
end

function M.setup(opts)
  opts = opts or {}
  notes_dir = vim.fn.expand(opts.notes_dir or '~/.vault/notes')
  vault_bin = opts.bin or 'vault'

  vim.api.nvim_create_autocmd('BufWritePost', {
    pattern = notes_dir .. '/*.md',
    callback = function(ev)
      M.push(ev.file)
    end,
  })

  vim.api.nvim_create_user_command('VaultSyncPush', function()
    M.push(vim.fn.expand('%:p'))
  end, {})

  vim.api.nvim_create_user_command('VaultSyncStatus', function()
    M.status()
  end, {})
end

vim.api.nvim_create_user_command('VaultSyncSetup', function()
  M.setup()
end, {})

return M
