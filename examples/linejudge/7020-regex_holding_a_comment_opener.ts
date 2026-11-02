// Reconstruction of LineJudge case 7020 for TypeScript, written from the case
// description in spec 07 01-conformance.md rather than copied from the suite,
// which is not checked out here. TypeScript inherits the regex literal fix
// from JavaScript, so the same input diverges the same way.
const re: RegExp = /[/*]/;
const y: number = 2;
const z: number = 3;
