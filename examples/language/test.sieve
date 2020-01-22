require ["fileinto", "reject", "vacation", "notify", "envelope", "body", "relational", "regex", "subaddress", "copy", "mailbox", "mboxmetadata", "servermetadata", "date", "index", "comparator-i;ascii-numeric", "variables", "imap4flags", "editheader", "duplicate", "vacation-seconds", "fcc", "x-cyrus-jmapquery"];
# whitelist first since we can't use the address-book whitelisting feature
# when the from = to address
if header :contains "From" "root@armakuni.example.com" {
  fileinto "INBOX.ln";
  stop;
}

### 1. Sieve generated for save-on-SMTP identities
if header :contains "X-Resolved-To" "+personalitysentitem-15989017@" {
  setflag "\\Seen";
  fileinto "\\Sent";
  stop;
}
if header :contains "X-Resolved-To" "+personalitysentitem-15989021@" {
  setflag "\\Seen";
  fileinto "\\Sent";
  stop;
}

### 2. Sieve generated for discard rules
if not header :matches "X-Spam-Known-Sender" "yes*" {
  if address :is ["To","Cc","Resent-To","X-Delivered-To"] "alldaychemist@t.example.com" {
    discard;
    stop;
  }
}

### 3. Sieve generated for spam protection
if not header :matches "X-Spam-Known-Sender" "yes*" {
  if 
    allof(
    header :contains "X-Backscatter" "yes",
    not header :matches "X-LinkName" "*"
    )
  {
    discard;
    stop;
  }
  if header :value "ge" :comparator "i;ascii-numeric" "X-Spam-score" "8" {
    fileinto "\\Junk";
    stop;
  }
}

### 4. Sieve generated for forwarding rules
if 
  anyof(
  address :contains "From" "majswps.example.com",
  address :contains "From" "majorsweeps.example.comcom",
  address :contains "From" "mjsupdates.example.com"
  )
{
  redirect :copy "somespammer@mediacompany.example.com";
  discard;
}
if address :contains "From" "spammer.example.com" {
  redirect :copy "abuse@someisp.example.com";
  discard;
}

### 5. Sieve generated for vacation responses
# You do not have vacation responses enabled.



### 6. Sieve generated for calendar preferences
if
  allof(
  header :is "X-ME-Cal-Method" "request",
  not exists "X-ME-Cal-Exists",
  header :contains "X-Spam-Known-Sender" "in-addressbook"
  )
{
  notify :method "addcal";
}
elsif exists "X-ME-Cal-Exists" {
  notify :method "updatecal";
}

### 7. Sieve generated for organise rules
if header :contains ["List-Id","List-Post"] "<freebsd-announce.freebsd.org>" {
  fileinto "INBOX.ml.freebsd-announce";
}
elsif header :contains ["List-Id","List-Post"] "<iccrg.irtf.org>" {
  fileinto "INBOX.ml.iccrg";
}
elsif 
  anyof(
  header :contains ["List-Id","List-Post"] "<~sircmpwn/aerc@lists.sr.ht>",
  header :contains ["List-Id","List-Post"] "<~sircmpwn/aerc.lists.sr.ht>"
  )
{
  fileinto "INBOX.ml.aerc";
}
elsif header :contains ["List-Id","List-Post"] "<sidrops.ietf.org>" {
  fileinto "INBOX.ml.sidrops";
}
elsif header :contains ["List-Id","List-Post"] "<tuhs.minnie.tuhs.org>" {
  fileinto "INBOX.ml.tuhs";
}
elsif header :contains ["List-Id","List-Post"] "<lynx-dev.nongnu.org>" {
  fileinto "INBOX.ml.lynx-dev";
}
elsif header :contains ["List-Id","List-Post"] "<info-cyrus.lists.andrew.cmu.edu>" {
  fileinto "INBOX.ml.cyrus-info";
}
elsif header :contains ["List-Id","List-Post"] "<cyrus-announce.lists.andrew.cmu.edu>" {
  fileinto "INBOX.ml.cyrus-announce";
}
elsif header :contains ["List-Id","List-Post"] "<cyrus-devel.lists.andrew.cmu.edu>" {
  fileinto "INBOX.ml.cyrus-devel";
}
elsif header :contains ["List-Id","List-Post"] "<mutt-announce.mutt.org>" {
  fileinto "INBOX.ml.mutt-announce";
}
elsif header :contains ["List-Id","List-Post"] "<mutt-users.mutt.org>" {
  fileinto "INBOX.ml.mutt-users";
}
elsif header :contains ["List-Id","List-Post"] "<mutt-dev.mutt.org>" {
  fileinto "INBOX.ml.mutt-dev";
}
elsif header :contains ["List-Id","List-Post"] "<neomutt-users-neomutt.org>" {
  fileinto "INBOX.ml.neomutt-users";
}
elsif header :contains ["List-Id","List-Post"] "<neomutt-devel-neomutt.org>" {
  fileinto "INBOX.ml.neomutt-devel";
}
elsif header :contains ["List-Id","List-Post"] "<ccan.lists.ozlabs.org>" {
  fileinto "INBOX.ml.ccan";
}
elsif header :contains ["List-Id","List-Post"] "<messaging.moderncrypto.org>" {
  fileinto "INBOX.ml.messaging";
}
elsif header :contains ["List-Id","List-Post"] "<noise.moderncrypto.org>" {
  fileinto "INBOX.ml.noise";
}
elsif header :contains ["List-Id","List-Post"] "<cfrg.irtf.org>" {
  fileinto "INBOX.ml.cfrg";
}
elsif header :contains ["List-Id","List-Post"] "<pearg.irtf.org>" {
  fileinto "INBOX.ml.pearg";
}
elsif header :contains ["List-Id","List-Post"] "<ausnog.lists.ausnog.net>" {
  fileinto "INBOX.ml.ausnog";
}
elsif header :contains ["List-Id","List-Post"] "<wireguard.lists.zx2c4.com>" {
  fileinto "INBOX.ml.wireguard";
}
elsif header :contains ["List-Id","List-Post"] "<juniper-nsp.puck.nether.net>" {
  fileinto "INBOX.ml.jnsp";
}
elsif header :contains ["List-Id","List-Post"] "<cisco-nsp.puck.nether.net>" {
  fileinto "INBOX.ml.cnsp";
}
elsif header :contains ["List-Id","List-Post"] "<jmap.ietf.org>" {
  fileinto "INBOX.ml.jmap";
}
elsif header :contains ["List-Id","List-Post"] "<oss-security.lists.openwall.com>" {
  fileinto "INBOX.ml.oss-security";
}
elsif header :contains "Mail-Followup-To" "cryptanalytic-algorithms@list.cr.yp.to" {
  fileinto "INBOX.ml.cryptanalytic-algorithms";
}
elsif header :contains ["List-Id","List-Post"] "<cryptography.metzdowd.com>" {
  fileinto "INBOX.ml.cryptography";
}
elsif header :contains ["List-Id","List-Post"] "<nanog.nanog.org>" {
  fileinto "INBOX.ml.nanog";
}
elsif header :contains ["List-Id","List-Post"] "<icnrg.irtf.org>" {
  fileinto "INBOX.ml.icnrg";
}
elsif
  anyof(
  address :is ["To","Cc","Resent-To","X-Delivered-To"] "root@armakuni.example.net",
  address :is ["To","Cc","Resent-To","X-Delivered-To"] "myname@armakuni.example.net"
  )
{
  fileinto "INBOX.ln";
}

### 8. Sieve generated for fetch mail filing
# You have no pop-links filing into special folders.


# Start user advanced sieve block 'PostFolder' {{{
else {
}
# }}} End user advanced sieve block 'PostFolder'


