package processor

import (
	"bytes"
	"reflect"
	"testing"
)

func TestIsWhitespace(t *testing.T) {
	if !isWhitespace(' ') {
		t.Errorf("Expected to be true")
	}
}

func TestCountStatsLines(t *testing.T) {
	fileJob := FileJob{
		Content: []byte(""),
		Lines:   0,
	}

	// Both tokei and sloccount count this as 0 so lets follow suit
	// cloc ignores the file itself because it is empty
	CountStats(&fileJob)
	if fileJob.Lines != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Lines)
	}

	// Interestingly this file would be 0 lines in "wc -l" because it only counts newlines
	// all others count this as 1
	fileJob.Lines = 0
	fileJob.Content = []byte("a")
	CountStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	fileJob.Lines = 0
	fileJob.Content = []byte("a\n")
	CountStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	// tokei counts this as 1 because its still on a single line unless something follows
	// the newline its still 1 line
	fileJob.Lines = 0
	fileJob.Content = []byte("1\n")
	CountStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	fileJob.Lines = 0
	fileJob.Content = []byte("1\n2\n")
	CountStats(&fileJob)
	if fileJob.Lines != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Lines)
	}

	fileJob.Lines = 0
	fileJob.Content = []byte("1\n2\n3")
	CountStats(&fileJob)
	if fileJob.Lines != 3 {
		t.Errorf("Three lines expected got %d", fileJob.Lines)
	}

	content := ""
	for i := 0; i < 5000; i++ {
		content += "a\n"
		fileJob.Lines = 0
		fileJob.Content = []byte(content)
		CountStats(&fileJob)
		if fileJob.Lines != int64(i+1) {
			t.Errorf("Expected %d got %d", i+1, fileJob.Lines)
		}
	}
}

func TestCountStatsCode(t *testing.T) {
	fileJob := FileJob{
		Content: []byte(""),
		Code:    0,
	}

	// Both tokei and sloccount count this as 0 so lets follow suit
	// cloc ignores the file itself because it is empty
	CountStats(&fileJob)
	if fileJob.Code != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Code)
	}

	// Interestingly this file would be 0 lines in "wc -l" because it only counts newlines
	// all others count this as 1
	fileJob.Code = 0
	fileJob.Content = []byte("a")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("i++ # comment")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("i++ // comment")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("a\n")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	// tokei counts this as 1 because its still on a single line unless something follows
	// the newline its still 1 line
	fileJob.Code = 0
	fileJob.Content = []byte("1\n")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("1\n2\n")
	CountStats(&fileJob)
	if fileJob.Code != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("1\n2\n3")
	CountStats(&fileJob)
	if fileJob.Code != 3 {
		t.Errorf("Three lines expected got %d", fileJob.Code)
	}

	content := ""
	for i := 0; i < 100; i++ {
		content += "a\n"
		fileJob.Code = 0
		fileJob.Content = []byte(content)
		CountStats(&fileJob)
		if fileJob.Code != int64(i+1) {
			t.Errorf("Expected %d got %d", i+1, fileJob.Code)
		}
	}
}

func TestCountStatsWithQuotes(t *testing.T) {
	fileJob := FileJob{}

	fileJob.Code = 0
	fileJob.Comment = 0
	fileJob.Complexity = 0
	fileJob.Content = []byte(`var test = "/*";`)
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("No line expected got %d", fileJob.Comment)
	}

	fileJob.Code = 0
	fileJob.Comment = 0
	fileJob.Complexity = 0
	fileJob.Content = []byte(`t = " if ";`)
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("No line expected got %d", fileJob.Comment)
	}
	if fileJob.Complexity != 0 {
		t.Errorf("No line expected got %d", fileJob.Complexity)
	}

	fileJob.Code = 0
	fileJob.Comment = 0
	fileJob.Complexity = 0
	fileJob.Content = []byte(`t = " if switch for while do loop != == && || ";`)
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("No line expected got %d", fileJob.Comment)
	}
	if fileJob.Complexity != 0 {
		t.Errorf("No line expected got %d", fileJob.Complexity)
	}
}

func TestCountStatsBlankLines(t *testing.T) {
	fileJob := FileJob{
		Content: []byte(""),
		Blank:   0,
	}

	CountStats(&fileJob)
	if fileJob.Blank != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte(" ")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("\n")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("\n ")
	CountStats(&fileJob)
	if fileJob.Blank != 2 {
		t.Errorf("Two line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("            ")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("            \n             ")
	CountStats(&fileJob)
	if fileJob.Blank != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("\r\n\r\n")
	CountStats(&fileJob)
	if fileJob.Blank != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("\r\n")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}
}

func TestCountStatsComplexityCount(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	checks := []string{
		"if ",
		"	if ",
		"if a.equals(b) {",
		"if(",
		" if(i.equals(0))",
		"    if(",
		"    if( ",
	}

	for _, check := range checks {
		fileJob.Complexity = 0
		fileJob.Content = []byte(check)
		fileJob.Language = "Java"
		CountStats(&fileJob)
		if fileJob.Complexity != 1 {
			t.Errorf("Expected complexity of 1 got %d for %s", fileJob.Complexity, check)
		}
	}
}

func TestCountStatsComplexityCountFalse(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	checks := []string{
		"if",
		"aif ",
		"aif(",
	}

	for _, check := range checks {
		fileJob.Complexity = 0
		fileJob.Content = []byte(check)
		fileJob.Language = "Java"
		CountStats(&fileJob)
		if fileJob.Complexity != 0 {
			t.Errorf("Expected complexity of 0 got %d for %s", fileJob.Complexity, check)
		}
	}

}

type linecounter struct {
	blanks   int
	comments int
	code     int
	loc      int
	stop     bool
}

func (l *linecounter) ProcessLine(job *FileJob, currentLine int64, lineType LineType) bool {
	l.loc++
	switch lineType {
	case LINE_BLANK:
		l.blanks++
	case LINE_COMMENT:
		l.comments++
	case LINE_CODE:
		l.code++
	}
	return !l.stop
}

func TestCountStatsCallback(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	fileJob.Content = []byte(`package foo

import com.foo.bar;

// this is a comment
class A {
}`)
	var lc linecounter
	fileJob.Language = "Java"
	fileJob.Callback = &lc
	CountStats(&fileJob)
	if lc.loc != 7 {
		t.Errorf("Expected loc of 7 got %d", lc.loc)
	}
	if lc.blanks != 2 {
		t.Errorf("Expected loc of 2 got %d", lc.blanks)
	}
	if lc.comments != 1 {
		t.Errorf("Expected loc of 1 got %d", lc.comments)
	}
	if lc.code != 4 {
		t.Errorf("Expected loc of 4 got %d", lc.code)
	}
}

func TestCountStatsCallbackInterrupt(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	fileJob.Content = []byte(`package foo

import com.foo.bar;

// this is a comment
class A {
}`)
	var lc linecounter
	lc.stop = true
	fileJob.Language = "Java"
	fileJob.Callback = &lc
	CountStats(&fileJob)
	if lc.loc != 1 {
		t.Errorf("Expected loc of 1 got %d", lc.loc)
	}
}

func TestCountStatsAccuracy(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Java",
	}

	fileJob.Content = []byte(`/* 23 lines 16 code 4 comments 3 blanks */

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

func TestCountStats(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Java",
	}

	fileJob.Content = []byte(`/**/
`)

	CountStats(&fileJob)

	if fileJob.Lines != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Lines)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Comment)
	}
}

func TestCountStatsAccuracyTwo(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "C++",
	}

	fileJob.Content = []byte(`/* 15 lines 7 code 4 comments 4 blanks */

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
}

// TODO improve logic so the below works
//func TestCountStatsAccuracyTokeiTest(t *testing.T) {
//	ProcessConstants()
//	fileJob := FileJob{
//		Language: "Rust",
//	}
//
//	fileJob.Content = []byte(`// 39 lines 32 code 2 comments 5 blanks
//
///* /**/ */
//fn main() {
//    let start = "/*";
//    loop {
//        if x.len() >= 2 && x[0] == '*' && x[1] == '/' { // found the */
//            break;
//        }
//    }
//}
//
//fn foo() {
//    let this_ends = "a \"test/*.";
//    call1();
//    call2();
//    let this_does_not = /* a /* nested */ comment " */
//        "*/another /*test
//            call3();
//            */";
//}
//
//fn foobar() {
//    let does_not_start = // "
//        "until here,
//        test/*
//        test"; // a quote: "
//    let also_doesnt_start = /* " */
//        "until here,
//        test,*/
//        test"; // another quote: "
//}
//
//fn foo() {
//    let a = 4; // /*
//    let b = 5;
//    let c = 6; // */
//}
//
//`)
//
//	CountStats(&fileJob)
//
//	// 39 lines 32 code 2 comments 5 blanks
//	if fileJob.Lines != 39 {
//		t.Errorf("Expected 39 lines got %d", fileJob.Lines)
//	}
//
//	if fileJob.Code != 32 {
//		t.Errorf("Expected 32 lines got %d", fileJob.Code)
//	}
//
//	if fileJob.Comment != 2 {
//		t.Errorf("Expected 2 lines got %d", fileJob.Comment)
//	}
//}

//////////////////////////////////////////////////
// Benchmarks Below
//////////////////////////////////////////////////

func BenchmarkCountStatsLinesEmpty(b *testing.B) {
	fileJob := FileJob{
		Content: []byte(""),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesSingleChar(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesTwoLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a\na"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesThreeLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a\na\na"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesShortLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesShortEmptyLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("          "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesThreeShortLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890\n1234567890\n1234567890"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesThreeShortEmptyLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("          \n          \n          "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesLongLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesLongMixedLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890          1234567890          1234567890          1234567890          1234567890          "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesLongAlternateLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFiveHundredLongLines(b *testing.B) {
	b.StopTimer()
	content := ""
	for i := 0; i < 500; i++ {
		content += "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890\n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFiveHundredLongLinesTriggerComplexityIf(b *testing.B) {
	b.StopTimer()
	content := ""
	for i := 0; i < 500; i++ {
		content += "iiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiii\n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFiveHundredLongLinesTriggerComplexityFor(b *testing.B) {
	b.StopTimer()
	content := ""
	for i := 0; i < 500; i++ {
		content += "fofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofo\n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFourHundredLongLinesMixed(b *testing.B) {
	b.StopTimer()
	content := ""
	for i := 0; i < 100; i++ {
		content += "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890\n"
		content += "1234567890          1234567890          1234567890          1234567890          1234567890          \n"
		content += "                                                                                                    \n"
		content += "a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a \n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCheckByteEqualityReflect(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := reflect.DeepEqual(one[1:], two[1:])

		if equal {
			count++
		}
	}

	b.Log(count)
}

func BenchmarkCheckByteEqualityBytes(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := bytes.Equal(one[1:], two[1:])

		if equal {
			count++
		}
	}

	b.Log(count)
}

// This appears to be faster than bytes.Equal because it does not need
// to do length comparison checks at the start
func BenchmarkCheckByteEqualityLoop(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := true

		for j := 1; j < len(one); j++ {
			if one[j] != two[j] {
				equal = false
				break
			}
		}

		if equal {
			count++
		}
	}

	b.Log(count)
}

// Check if the 1 offset makes a difference, which it does by ~1 ns
func BenchmarkCheckByteEqualityLoopWithAddtional(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := true

		// Don't start at 1 like the above but 0 to do a full scan
		for j := 0; j < len(one); j++ {
			if one[j] != two[j] {
				equal = false
				break
			}
		}

		if equal {
			count++
		}
	}

	b.Log(count)
}

func BenchmarkCheckArrayCheck(b *testing.B) {
	array := []byte{
		'a',
		'b',
		'c',
		'd',
		'e',
		'f',
		'g',
		'h',
		'i',
		'j',
	}

	var searchFor byte = 'j'
	found := 0

	for i := 0; i < b.N; i++ {
		for index := 0; index < len(array); index++ {
			if array[index] == searchFor {
				found++
				break
			}
		}
	}

	b.Log(found)
}

func BenchmarkCheckMapCheck(b *testing.B) {
	array := map[byte]bool{
		'a': true,
		'b': true,
		'c': true,
		'd': true,
		'e': true,
		'f': true,
		'g': true,
		'h': true,
		'i': true,
		'j': true,
	}

	var searchFor byte = 'j'
	found := 0

	for i := 0; i < b.N; i++ {

		_, ok := array[searchFor]

		if ok {
			found++
		}
	}

	b.Log(found)
}
