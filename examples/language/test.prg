/*
   This program demonstrates the calculation
   of the factorial of a number using a function.

   https://github.com/boyter/scc/issues/484
*/

FUNCTION Factorial( n )
   LOCAL result := 1
   LOCAL i

   // Loop from 1 to n
   FOR i := 1 TO n
      result := result * i
   NEXT

   RETURN result

// Main program execution starts here
FUNCTION Main()
   LOCAL num := 5
   LOCAL fact

   // Calculate the factorial of the number
   fact := Factorial( num )

   // Check if the factorial is greater than 100
   IF fact > 100
      ? "Factorial is greater than 100"
   ELSE
      ? "Factorial is less than or equal to 100"
   ENDIF

   RETURN