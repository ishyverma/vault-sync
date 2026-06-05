" VaultSync — Vim plugin
" Source this file from your .vimrc:
"   source ~/.config/vault/vault.vim

if exists('g:loaded_vaultsync')
  finish
endif
let g:loaded_vaultsync = 1

if !exists('g:vaultsync_notes_dir')
  let g:vaultsync_notes_dir = expand('~/.vault/notes')
endif

augroup VaultSync
  autocmd!
  execute 'autocmd BufWritePost ' . g:vaultsync_notes_dir . '/*.md'
        \ 'silent! call s:push(expand("<afile>"))'
augroup END

command! VaultSyncPush :call s:push(expand('%:p'))
command! VaultSyncStatus :call s:status()

function! s:push(filepath) abort
  let l:result = system('vault push ' . shellescape(a:filepath) . ' 2>&1')
  if v:shell_error
    echohl WarningMsg | echomsg '✗ VaultSync: ' . trim(l:result) | echohl None
    return 0
  endif
  echohl MoreMsg | echomsg '✓ VaultSync: synced' | echohl None
  return 1
endfunction

function! s:status() abort
  echo system('vault sync status 2>&1')
endfunction
