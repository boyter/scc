# This is a comment
// This is another comment

/* This is a
   multiline
   comment
*/

provider "null" {}

locals {
  list = [for i, v in ["foo", "bar"] : v if v == "foo"]
  map  = { for k, v in { "a" = "foo", "b" = "bar" } : v => k }
}

resource "null_resource" "count" {
  count = 2

  triggers = {
    conditional  = 1 < 2 || 2 <= 3 && "a" == "c" ? "foo" : "bar"
    conditional2 = 1 > 2 || 2 >= 3 && "a" != "c" ? "foo" : "bar"
  }

  provisioner "local-exec" {
    command = "echo ${join(" ", local.list)}"
  }
}

resource "null_resource" "foreach" {
  for_each = toset(local.map)
}
