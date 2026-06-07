" VaultSync — vim-plug compatible loader
" Plug 'ishyverma/vault-sync', { 'dir': 'vim' }

if exists('g:loaded_vault_sync')
  finish
endif
let g:loaded_vault_sync = 1

command! VaultSyncPush call vault#push(expand('%:p'))
command! VaultSyncStatus call vault#status()
