" VaultSync — Vim plugin
" Source this file from your .vimrc:
"   source ~/.config/vault/vault.vim
"
" Statusline integration (add to your statusline):
"   set statusline+=%{VaultSyncStatusline()}

if exists('g:loaded_vaultsync')
  finish
endif
let g:loaded_vaultsync = 1

if !exists('g:vaultsync_notes_dir')
  let g:vaultsync_notes_dir = expand('~/.vault/notes')
endif

if !exists('g:vaultsync_bin')
  let g:vaultsync_bin = 'vault'
endif

augroup VaultSync
  autocmd!
  execute 'autocmd BufWritePost ' . g:vaultsync_notes_dir . '/*.md'
        \ 'silent! call s:push(expand("<afile>"))'
augroup END

command! VaultSyncPush :call s:push(expand('%:p'))
command! VaultSyncStatus :call s:status()

function! s:push(filepath) abort
  let l:result = system(g:vaultsync_bin . ' push ' . shellescape(a:filepath) . ' 2>&1')
  if v:shell_error
    echohl WarningMsg | echomsg '✗ VaultSync: ' . trim(l:result) | echohl None
    return 0
  endif
  echohl MoreMsg | echomsg '✓ VaultSync: synced' | echohl None
  return 1
endfunction

function! s:status() abort
  echo system(g:vaultsync_bin . ' sync status 2>&1')
endfunction

" Returns a short status string for the statusline.
" Shows sync status of the current file if it's under the vault notes dir.
function! VaultSyncStatusline() abort
  let l:file = expand('%:p')
  if l:file !~# g:vaultsync_notes_dir
    return ''
  endif
  let l:result = system(g:vaultsync_bin . ' sync status --json 2>/dev/null')
  if v:shell_error
    return ''
  endif
  " Simple heuristic: check if output contains "synced"
  if l:result =~# '"synced"'
    return ' ✓'
  elseif l:result =~# '"conflict"'
    return ' ⚠'
  elseif l:result =~# '"failed"'
    return ' ✗'
  elseif l:result =~# '"pending"'
    return ' ⟳'
  endif
  return ''
endfunction
