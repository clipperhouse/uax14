This repository will be a Go language implementation of UAX #14, the Unicode Line Breaking Algorithm.

The algorithm is specified at https://www.unicode.org/reports/tr14/

# Research

Begin by understanding the specification and description at https://www.unicode.org/reports/tr14/.

The detailed algorithm is at https://www.unicode.org/reports/tr14/#Algorithm. Understand its syntax and intent. This is what we will implement.

Look at a few well-regarded existing implementations (in any language), to understand desirable APIs.

# Implementation style

The implementation style of github.com/clipperhouse/uax29 is desirable for this project. Among its key implementation characteristics:

## Trie lookup

Code-generation of trie data structure, a KV mapping of UTF-8 bytes to Unicode categories. The category is expressed as an int enum (`iota`), and is powers of two (`<< iota`) to allow bitwise operations.

A code generation example can be found in the internal/gen folder of the uax29 package. A similar and cleaner codegen trie can be found in github.com/clipperhouse/displaywidth/internal/gen.

The code generation downloads the relevant Unicode data files, parses them, and generates the trie data structure using the triegen package (from x/text).

## Implementation of a SplitFunc

The bufio.SplitFunc interface is used to implement the algorithm.

The SplitFunc iterates over the bytes, looks up the relevant category, and applies the rules of the algorithm to the current position. The decision is whether to continue or break.

In the case of uax29/graphemes, for example, the rules are numbered GB1, GB2, etc. The rules for UAX #14 are numbered LB1, LB2, etc. I like that the uax29 split func's read like the algorithm specification.

It's important to have a clear notiion of what "current position" means within this function. Lookaheads and lookbacks will play a role.

## Iterator

In turn, an iterator will use the SplitFunc to iterate over the bytes and track state. Look at the Iterator type in the uax29 graphemes package for an example.

The iterator may also incoude optinizations or other options for the user to configure, but we won't focus on those yet.

# API

I am not yet clear on the API for this project., but I believe I want the user to do something like:

```go
myString := "Hello, world!"
iterator := uax14.NewIterator(myString)
for iterator.Next() {
    // Current() is the slice of the string since the last break opportunity
    iterator.Current()
    // MustBreak() represents UAX 14's "mandatory break"
    iterator.MustBreak()
    // CanBreak() represents UAX 14's "optional break" or "opportunity"
    iterator.CanBreak()
}
```

UAX 14 also defines "must not break", but I believe we don't need to expose that in the API -- we simply iterate until the first break opportunity.

# Planning

Develop an implementation plan for this proposed Go language implementation of UAX #14. Start with the "Research" section above.

Then the plan (in order):

- Understand the UAX #14 specification
- Understand the required Unicode character categories for UAX #14
- Locate the data files for those needed categories. This is probably a good starting point: https://www.unicode.org/reports/tr44/
- Implement the trie code generation, similar to `uax29` and `displaywidth` packages.
  - `uax29` is more similar to our current goals, but its implementation is a bit messy. `displaywidth` code generation is better organized. Use both packages as inspiration.
- Exposure of a lookup function which uses the trie
- Implement the SplitFunc for the algorithm specification: https://www.unicode.org/reports/tr14/#Algorithm
  - This will be the core of the implementation.
  - Look at the `uax29/graphemes` package for an examples
  - It should read like the specification as closely as possible.
- Implement the Iterator
- Implement the API
- Create tests.
  - Does the specification include test cases or a test suite, as UAX #29 does?
