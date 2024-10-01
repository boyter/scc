# Use gojq for infinite precision integer arithmetic
def tobase($b):
    def digit: "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"[.:.+1];
    def mod: . % $b;
    def div: ((. - mod) / $b);
    def digits: recurse( select(. >= $b) | div) | mod ;

    select(2 <= $b and $b <= 36)
    | [digits | digit] | reverse | add;

def send_more_money:
    def choose(m;n;used): ([range(m;n+1)] - used)[];
    def num(a;b;c;d): 1000*a + 100*b + 10*c + d;
    def num(a;b;c;d;e): 10*num(a;b;c;d) + e;
    first(
      1 as $m
      | 0 as $o
      | choose(8;9;[]) as $s
      | choose(2;9;[$s]) as $e
      | choose(2;9;[$s,$e]) as $n
      | choose(2;9;[$s,$e,$n]) as $d
      | choose(2;9;[$s,$e,$n,$d]) as $r
      | choose(2;9;[$s,$e,$n,$d,$r]) as $y
      | select(num($s;$e;$n;$d) + num($m;$o;$r;$e) ==
               num($m;$o;$n;$e;$y))
      | [$s,$e,$n,$d,$m,$o,$r,$e,$m,$o,$n,$e,$y] );