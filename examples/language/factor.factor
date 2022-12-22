! Written for SCC by CapitalEx
USING: combinators io kernel math.order math.parser random
ranges ;
IN: simple-guessing-game

: pick-number ( -- n )
    100 [1..b] random ;

: read-number ( -- n )
    "Enter a guess: " write readln dec> ;

: guessing-game ( n -- )
    dup read-number <=> dup {
        { +lt+ [ "Too high!" print t ] }
        { +gt+ [ "Too low!"  print t ] }
        [ drop "You won!" print f ]
    } case [ guessing-game ] [ drop ] if ;

MAIN: [
    "I'm thinking of a number between 1 and 100" print
    pick-number
    guessing-game
]