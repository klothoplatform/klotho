# Errors and Logging
This document is for developers of klotho to help standardize how we surface problems.


## panic
Programmer error - these should be easily detectable via unit tests and most commonly found in initializers. Users should never see these.

### Examples:
```go
var commentRegex = regexp.MustCompile(`(?m)^(\s*)`) // MustCompile panics if the regex compile fails
```

```go
query, err := sitter.NewQuery([]byte(q), lang)
if err != nil {
  // Panic because this is a programmer error with the query string.
  panic(errors.Wrapf(err, "Error constructing query for %s", q))
}
```

## error
Return an error if there is a problem that should cause a compilation failure. Callers will decide whether to end compilation immediately or continue the rest before failing.

### Examples:
```go
err := f.Reparse([]byte(fileContent))
if err != nil {
  // any time we need to (re)parse a file, we should return the error to surface any syntax errors caused by the transformation.
  return f, err
}
```

```go
proxy, err := MakeProxyFile(original, names)
if err != nil {
  // the proxy file template failed to render. This could be a programmer error (in the template) or a configuration error (something required was missing).
  return err
}
```

## log.warn
A misconfiguration or other user mistake that has a high likelihood of causing problems down the line. These don't cause a compilation failure by default.

### Eamples:
```go
if object != nil && !query.NodeContentEquals(object, file.program, "exports") {
  // the @klotho annotation was specified in a way that is not supported
  lang.CompilerLog(
    zap.WarnLevel,
    annotation,
    file,
    "expected object of assignment to be 'exports'",
  )
  return nil
}
```

```go
if len(routes) == 0 {
  // we expected to find additional statements to go along with the annotation, but did not find any
  lang.CompilerLog(
    zap.WarnLevel,
    capNode,
    f,
    "No routes found",
  )
}
```

## log.info
Information that would be useful to a user that wouldn't be surfaced through a resource on the `CompilationResult`.

### Examples:
```go
// The compilation result for a gateway would have all the routes, but this gives more details on how many routes came from which middlewares.
lang.CompilerLog(
  zap.InfoLevel,
  capNode,
  f,
  fmt.Sprintf("Found %d routes for middleware %s", len(routes), mwImportName),
)
```

## log.debug
These show in verbose mode and can give insight into the compiler process. Mostly useful for compiler development and marginally useful to a regular user.

### Examples:
```go
// This is similar to the middleware log.info example except this gives even finer detail more suitable for a verbose mode.
log.Debugf("Got %d verb functions", len(verbFuncs))
```
