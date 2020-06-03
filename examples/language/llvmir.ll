; f returns the sum of the given operands.
define i32 @f(i32 %x, i32 %y) {
	%result = add i32 %x, %y
	ret i32 %result
}