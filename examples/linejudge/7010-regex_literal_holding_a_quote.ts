// Reconstruction of LineJudge case 7010 for TypeScript, written from the case
// description in spec 07 01-conformance.md rather than copied from the suite,
// which is not checked out here. TypeScript inherits the regex literal fix
// from JavaScript, so the same input diverges the same way.
const re: RegExp = /["']/;
// a comment
const x: number = 1;
