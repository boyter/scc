#!/usr/bin/perl

for($i=1; $i<10; $i++){
    for($j=1; $j<$i+1; $j++){
       printf "%-4s%-2d  ", "$j*$i=", $i*$j;
    }
    print "\n";
}
