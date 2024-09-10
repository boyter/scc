#!/usr/bin/nawk -f

BEGIN {
    print("Enter string to encode:")
}

{
    print("Encoded string:")
    print(rot13($0))
}

function rot13(str, new, idx) {
    new = ""
    for (idx = 1; idx <= length; ++idx) {
        new = new rot13_impl(substr(str, idx, 1))
    }
    return new
}

function rot13_impl(ch, list, idx) {
    list = "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZ"
    idx = index(list, ch)
    if (idx == 0) {
        return ch
    } else {
        return substr(list, idx + 13, 1)
    }
}
