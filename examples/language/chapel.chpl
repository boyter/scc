class Integer {
  var x:int;
}
proc deferInFunction() {
  var c = new unmanaged Integer(1);
  writeln("created ", c);
  defer {
    writeln("defer action: deleting ", c);
    delete c;
  }
  // ... (function body, possibly including return statements)
  // The defer action is executed no matter how this function returns.
}
deferInFunction();