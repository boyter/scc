const url = `/foo/bar/*`;

function getUrl(path?: string): string {
  return path ?? url;
}

function getUrl2() {
  return url;
}
