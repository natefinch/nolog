nolog
=====

just a dumb little tool to run juju tests and filter out all the logging


nolog runs go test and passes through all command line args to it.  

nolog filters out lines to stdout that start with [LOG]

If the first arg to nolog is -f, it will write all output to tests.out, as well as printing the filtered output to stdout.
