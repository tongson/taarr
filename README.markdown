# rr

### WHAT

shell script runner inspired by rerun[1], bashing[2] and drist[3]

[1] https://github.com/rerun/rerun\
[2] https://github.com/xsc/bashing\
[3] git://bitreich.org/drist\

Tested on Linux only.
Requires OpenSSH 4.7+ for sftp recursive file transfers. If you can't use 4.7+, a workaround is to use heredocs.

### BUILDING

Requires OmniaJIT to build. Rename rr.mk to Makefile then `make`.


### TUTORIAL

In the source tree, the *TUTORIAL* directory contains a hierarchy that persistently enables IP forwarding through sysctl upon the remote SSH host named *avocado*

    cd TUTORIAL
    rr avocado sysctl:apply

