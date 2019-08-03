Project Cuba
------------

Experiment in allowing workers to own the means of production.

Go's built in `chan`s are based on the Communicating Sequential Processes (CSP)
model. They work well when the processing is sequential, i.e.

```
Producer -> Processor [-> Processor ...] -> Consumer
```

But not everything fits well with this model, like when work is discovered as
part of the processing. Examples are walking a directory structure, and finding
more subdirectories as you go -- or a web crawler following hyperlinks from the
pages it crawls.

Cuba attempts to build a parallel system where processes can both consume and
produce work simultaneously.
