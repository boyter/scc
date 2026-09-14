<?php
// SPDX-License-Identifier: MIT

# A hash comment, which PHP has and most of the C family does not.

/* A block comment
   over more than one line. */

declare(strict_types=1);

#[Attribute]
final class Counter
{
    private array $counts = [];

    public function __construct(private string $name)
    {
    }

    public function tally(array $words): int
    {
        $total = 0;
        foreach ($words as $word) {
            if ($word === '') {
                continue;
            }
            $this->counts[$word] = ($this->counts[$word] ?? 0) + 1;
            $total++;
        }

        return $total;
    }

    public function describe(): string
    {
        $single = 'a single quoted string with a " in it';
        $double = "a double quoted string with a ' in it and {$this->name}";

        return $single . ' ' . $double;
    }
}

$counter = new Counter('sample');
$total = $counter->tally(['one', 'two', '', 'two']);
while ($total > 0) {
    $total--;
}
echo $counter->describe(), PHP_EOL;
