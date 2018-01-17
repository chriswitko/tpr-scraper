# Simple folder structure

```
github.com/soundcloud/simple/
    README.md
    Makefile
    main.go
    main_test.go
    support.go
    support_test.go
```

Maybe at some point you want to create a new support package. Use a subdirectory in your main repo, and import it using the fully-qualified name. If the package only has one file, or one struct, it almost certainly doesn’t need to be separated.

# More complex version

```
github.com/soundcloud/complex/
    README.md
    Makefile
    complex-server/
        main.go
        main_test.go
        handlers.go
        handlers_test.go
    complex-worker/
        main.go
        main_test.go
        process.go
        process_test.go
    shared/
        foo.go
        foo_test.go
        bar.go
        bar_test.go
```

Note that there’s never a src directory involved. With the exception of a vendor subdirectory (more on that below) your repos shouldn’t contain a directory named src, or represent their own

# Binaries

| Shipping a | Vendor path |	Procedure |
| ---------- | ----------- | ---------- |
| Binary | _vendor | Blessed build with prefixed GOPATH |
| Library | vendor | Rewrite your imports |

If you ship a binary, create a _vendor subdirectory in the root of your repository. (With a leading underscore, so the go tool ignores it when doing e.g. go test ./....) Treat it like its own GOPATH; for example, copy the dependency github.com/user/dep to _vendor/src/github.com/user/dep. 

Then, write a so-called blessed build procedure, which prepends _vendor to any existing GOPATH. (Remember: GOPATH is actually a list of paths, searched in order by the go tool when resolving imports.) For example, you might have a top-level Makefile that looks like this:

# Makefile

```go
GO ?= go
GOPATH := $(CURDIR)/_vendor:$(GOPATH)

all: build

build:
    $(GO) build
```

# Notes

* Define flags in func main() to prevent from reading them arbitrarily in your code as globals, which forces you to abide strict dependency injection, which makes testing easier.