# rr

# WHAT

shell script runner via SSH inspired by rerun[1], bashing[2] and drist[3]

[1] https://github.com/rerun/rerun  
[2] https://github.com/xsc/bashing  
[3] git://bitreich.org/drist  

# NOTES

Tested on Linux only.  

Remote host only requires OpenSSH installed. It's recommended that the host running `rr` should configure remote hosts through `~/.ssh/config`.  

Requires OpenSSH 4.7+ for sftp recursive file transfers. If you can't use 4.7+, a workaround is to use heredocs.  

# BUILDING

Requires [OmniaJIT](https://github.com/tongson/OmniaJIT/) to build.  
Rename rr.mk to Makefile then `make`.


# TUTORIAL

In the source tree, the *TUTORIAL* directory contains a hierarchy that persistently enables IP forwarding through sysctl upon the remote SSH host named *avocado*

    cd TUTORIAL
    rr avocado sysctl:apply

Steps that `rr` performs:

1. Copy files-avocado/etc/sysctl.d/forwarding.conf to avocado:/etc/sysctl.d/forwarding.conf
2. Generates the script:

    #!/bin/sh
    unset IFS
    set -efu
    PATH=/bin:/sbin:/usr/bin:/usr/sbin
    LC_ALL=C
    sysctl --system

3. Runs the script on host avocado via SSH.

# REFERENCE

### Hierarchy

    TOPLEVEL
    ├── files                          <--- synced to any host
    ├── files-avocado                  <--- synced to the avocado host
    ├── lib                            <--- sourced by all scripts
    └── task
        ├── files                      <--- synced to any host when task:* is called
        ├── files-avocado              <--- synced to the avocado host then task:* is called
        ├── lib                        <--- sourced along with task:* scripts
        └── script

