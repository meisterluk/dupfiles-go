dupfiles-go
===========

:author:    meisterluk
:license:   BSD 3-clause
:version:   0.3.0 "first public release"

Find equivalent nodes in a filesystem tree.
See the project homepage for more information:

  http://lukas-prokop.at/proj/dupfiles/

How to run it

  Recommended approach (you need to trust my binaries):
  1. Download an appropriate binary from the project homepage.
  2. Check the md5sum or shasum of the binary. If fine, continue.
     Otherwise you download connection is compromised.
  3. Run it in a terminal (Linux: bash, Windows: cmd.exe)

  Alternative approach (you don't need to trust me):
  1. Check out the source code at github.
     Check that nothing evil is done.
  2. Compile the software yourself (you need to download
     Go's toolchain available at https://golang.org/dl/ )
  3. The bin/ directory of dupfiles contains a binary file.
  4. Run it in a terminal (Linux: bash, Windows: cmd.exe)


For the following data I used the following reference system:
Reference system: Thinkpad x220 tablet, Linux xubuntu 15.10, x86_64

Performance

  Equivalence:
    directory:    131243 nodes (find ~ | wc -l)
    total size:   56.34 MB (du --summarize ~)
    runtime:      16 minutes (time dupfiles HOME1 ~ HOME2 ~)

  Be aware that the most expensive routine (matching) is not
  yet concurrent. Hence there is room for improvement which
  is expected to be used soon.

Memory

  Equivalence:
    directory:        131243 nodes
    total size:       56.34 MB
    static memory:    2 MB
    [ size bin/dupfiles-go | cut -f4 ]
    dynamic memory:   150 MB
    [ /usr/bin/time -f '%MkB' dupfiles HOME1 ~ HOME2 ~ ]

Dependencies

  I currently do not depend on any dependencies.

Compilation

  1. Set up GOPATH and GOROOT as described in the official documentation:
     https://golang.org/doc/code.html
  2. Run "go install github.com/meisterluk/dupfiles-go"

Backwards compatibility

  Until the 1.0 release is hit, the API might change

Pull requests

  I am very interested in any feedback and will accept
  any pull requests I consider useful.

best regards,
meisterluk
