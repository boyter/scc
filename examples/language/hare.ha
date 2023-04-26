use fmt;
use strings;

export type error = !str;

type example = struct {
	number: i64,
	error: error,
};

fn function(a: i32, b: i32) i32 = a + b;

export fn main() void = {
	for (true) {
		break;
	};

	const string = "A string";	// a comment
	const another_string = `aaa`;
	let iter = strings::iter(string);

	let r = match (strings::next(&iter)) {
	case void => return;
	case let r: rune =>
		// comment
		yield r;
	};

	switch (r) {
	case 'a' =>
		abort();
	case =>
		if (r != 'd') fmt::println("not d!")! else fmt::println("d!")!;
	};

	for (let i = 0z; i <= 10; i += 1) {
		fmt::print(i, ' ')!;
	};
};
