# docker wrapper that returns a nushell table
def docker [
  ...args:string # command to be passed to the real docker command
  ] {
  let data = (^docker $args --format={{json .}}|lines|each {$it|from json})
  if Labels in ($data|get) {
    $data|docker labels
  } {
    $data
  }
  
}

# subcommand used to reformat docker labels into their own table
def 'docker labels' [] {
  update Labels {
    get Labels|
    split row ','|
    where ($it|str starts-with ' ') == $false|
    split column '=' name value
  }
}