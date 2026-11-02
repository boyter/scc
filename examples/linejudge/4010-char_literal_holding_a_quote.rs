// Reconstruction of LineJudge case 4010, written from the case description in
// spec 07 01-conformance.md rather than copied from the suite, which is not
// checked out here. It reproduces the counts the spec records: the suite wants
// 1 code and 2 comment, and the generic loop answers 3 code and 0 comment
// because the quote inside the character literal opens a string that never
// closes.
let c = '"';
// a comment
// another comment
