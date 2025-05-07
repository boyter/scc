const url = `/foo/bar/*`;

function getUrl(path?: string): string {
  return path ?? url;
}

function getUrl2() {
  return url;
}
// test for issue 610 */
/* 11 lins, 2 comments, 2 blanks, 7 codes, 1 complexity */
