" func
function! ToggleLineNumbers()
  if &number
    set nonumber
    echo "turn off line numbers"
  else
    set number
    echo "turn on line numbers"
  endif
endfunction

nnoremap <F2> :call ToggleLineNumbers()<CR>

" auto format go source code
autocmd BufWritePre *.go :silent! execute '%!gofmt'

set tabstop=4
set shiftwidth=4
set expandtab
set number
