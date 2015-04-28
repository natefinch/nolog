nolog
=====

a bit more than a dumb little tool to help filter logging.


nolog runs go test and passes through all command line args to it but you might want to use our own shortcuts for convenience.  

  **-c** false by default, when set to true will color the output logs.

  **-f** false by default, setting this flag will output the lines beginning with [LOG] to a file.

  **-filter** a string to be used to filter tests with -gocheck.f (requires gocheck).

  **-name** "tests.log" by default, is an alternative file name for the ouput file.

  **-v** false by default, will use -test.v=true on the test run.

When no arguments are passed it will behave like the original version, filtering out lines starting with [LOG]
