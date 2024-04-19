local foo = "bar";

// This is a comment
# This is another comment
/*
 * This is a bigger comment
 */

{
  local bar = "foo",

  array: [
    "foo",
    "bar",
    123,
    { foo: "bar" },
  ],

  number: 3e10,
  anotherNumber: 3.14,
  yetAnotherNumber: 4,
  bool: true,

  f:: function(x) x > 0,

  object: {
    foo: $.f(1) || $.f(-1),
    bar: if std.objectHas(self, "foo") then "foo" else "bar",
    another: {
      foo: self["bar.bar"],
    },
  } + {
    another+: {
      "bar.bar": |||
        foo
        %(bar)s
      ||| % $.object,
    },
  },
}
