" VaultSync autoload functions
" These are loaded on demand by Vim's autoload mechanism

let s:notes_dir = expand('~/.vault/notes')
let s:vault_bin = 'vault'

function vault#push(filepath) abort
  let l:result = system(s:vault_bin . ' push ' . shellescape(a:filepath) . ' 2>&1')
  if v:shell_error != 0
    echohl WarningMsg | echo '✗ VaultSync: ' . trim(l:result) | echohl None
    return 0
  endif
  echohl MoreMsg | echo '✓ VaultSync: synced' | echohl None
  return 1
endfunction

function vault#status() abort
  let l:result = system(s:vault_bin . ' sync status 2>&1')
  echo l:result
endfunction

function vault#statusline() abort
  let l:file = expand('%:p')
  if l:file !~# s:notes_dir
    return ''
  endif
  let l:basename = fnamemodify(l:file, ':t')
  let l:result = system(s:vault_bin . ' sync status --json 2>/dev/null')
  if v:shell_error != 0 | return '' | endif
  try
    let l:data = json_decode(l:result)
  catch
    return ''
  endtry
  let l:entries = get(l:data, 'entries', l:data)
  for l:entry in l:entries
    if l:entry.note ==# l:basename || l:entry.note ==# substitute(l:basename, '\.md$', '', '')
      if l:entry.status ==# 'synced'   | return ' ✓'
      elseif l:entry.status ==# 'conflict' | return ' ⚠'
      elseif l:entry.status ==# 'failed'   | return ' ✗'
      elseif l:entry.status ==# 'pending'  | return ' ⟳'
      endif
    endif
  endfor
  return ' ○'
endfunction

" Auto-sync on write
augroup vault_sync
  autocmd!
  execute 'autocmd BufWritePost ' . s:notes_dir . '/*.md call vault#push(expand("<afile>:p"))'
augroup END
