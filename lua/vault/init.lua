-- VaultSync — lazy.nvim plugin entry point
-- Install:
--   {
--     'ishyverma/vault-sync',
--     config = function()
--       require('vault').setup()
--     end,
--   }

local ok, vault = pcall(require, 'vault')
if ok then
  return vault
end

-- Fallback: load from vim/ subdirectory
local path = vim.fn.expand('~/.local/share/nvim/lazy/vault-sync/vim/lua/vault/init.lua')
if vim.fn.filereadable(path) == 1 then
  return dofile(path)
end

return {}
