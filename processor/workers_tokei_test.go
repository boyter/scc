package processor

import (
	"testing"
)

func TestCountStatsJava(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Java",
	}

	fileJob.SetContent(`/* 23 lines 16 code 4 comments 3 blanks */

/*
* Simple test class
*/
public class Test
{
 int j = 0; // Not counted
 public static void main(String[] args)
 {
     Foo f = new Foo();
     f.bar();

 }
}

class Foo
{
 public void bar()
 {
   System.out.println("FooBar"); //Not counted
 }
}`)

	CountStats(&fileJob)

	if fileJob.Lines != 23 {
		t.Errorf("Expected 23 lines")
	}

	if fileJob.Code != 16 {
		t.Errorf("Expected 16 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsAccuracyCPlusPlus(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "C++",
	}

	fileJob.SetContent(`/* 15 lines 7 code 4 comments 4 blanks */

#include <iostream>


using namespace std;

/*
 * Simple test
 */
int main()
{
    cout<<"Hello world"<<endl;
    return 0;
}
`)

	CountStats(&fileJob)

	if fileJob.Lines != 15 {
		t.Errorf("Expected 15 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 7 {
		t.Errorf("Expected 7 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsAccuracyRakefile(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rakefile",
	}

	fileJob.SetContent(`# 10 lines 4 code 2 comments 4 blanks

# this is a rakefile

task default: %w[test]

task :test do # not counted
  ruby "test/unittest.rb"
end

`)

	CountStats(&fileJob)

	if fileJob.Lines != 10 {
		t.Errorf("Expected 10 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsAccuracyRuby(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Ruby",
	}

	fileJob.SetContent(`# 20 lines 9 code 8 comments 3 blanks
x = 3
if x < 2
  p = "Smaller"
else
  p = "Bigger"
end

=begin
  Comments
  Comments
  Comments
  Comments
=end

# testing.
while x > 2 and x < 10
  x += 1
end

`)

	CountStats(&fileJob)

	if fileJob.Lines != 20 {
		t.Errorf("Expected 20 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 9 {
		t.Errorf("Expected 9 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 8 {
		t.Errorf("Expected 8 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsAccuracyTokeiTest(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.SetContent(`// 39 lines 32 code 2 comments 5 blanks

/* /**/ */
fn main() {
  let start = "/*";
  loop {
      if x.len() >= 2 && x[0] == '*' && x[1] == '/' { // found the */
          break;
      }
  }
}

fn foo() {
  let this_ends = "a \"test/*.";
  call1();
  call2();
  let this_does_not = /* a /* nested */ comment " */
      "*/another /*test
          call3();
          */";
}

fn foobar() {
  let does_not_start = // "
      "until here,
      test/*
      test"; // a quote: "
  let also_doesnt_start = /* " */
      "until here,
      test,*/
      test"; // another quote: "
}

fn foo() {
  let a = 4; // /*
  let b = 5;
  let c = 6; // */
}

`)

	CountStats(&fileJob)

	// 39 lines 32 code 2 comments 5 blanks
	if fileJob.Lines != 39 {
		t.Errorf("Expected 39 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 32 {
		t.Errorf("Expected 32 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 5 {
		t.Errorf("Expected 5 lines got %d", fileJob.Blank)
	}
}
