# rr

# WHAT

shell script manager and runner inspired by rerun[1], bashing[2] and drist[3]

[1] https://github.com/rerun/rerun  
[2] https://github.com/xsc/bashing  
[3] git://bitreich.org/drist  

# WHY

I tried building by own Lua-based configuration management software. A little more than thousand commits in, I realized the oneshot nature of shell scripts more convenient.

CFEngine, Puppet, Chef and others did not offer advantages over a combination of rerun/bashing + drist. Cons mostly outweight the pros. Slow and complicated are the common complaints.

According to the Lindy effect; shell scripts, openssh and tar will outlive these mentioned CM software.

I mostly manage a handful of servers so `rr` serves my needs just fine.

# INVOCATION

## LOCAL

Run locally, the default. Requires tar(1) for `.files`.
```
rr namespace:script
rr localhost namespace:script
```

## CONTAINER

Run on a local container's PID via nsenter(1). Requires rsync(1) for `.files`.
```
rr 1333 namespace:script
```

## REMOTE

Run remote via OpenSSH. Requires OpenSSH 4.7+ for `.files`.
```
rr remotehost namespace:script
```

# MULTICALL MODES

You can use a symlink to activate the modes.

## Verbose

When called as `rrv` _or_ a console is detected it runs in verbose mode.
```
Tue, 20 Jul 2021 20:07:16 +0800 rr 0.10.0 "Kilowatt Triceps"
Tue, 20 Jul 2021 20:07:16 +0800 Running tmp:test via local
Tue, 20 Jul 2021 20:07:26 +0800 Done. Output:
 local ┌── stdout
 local │
 local │ 3
 local │
Tue, 20 Jul 2021 20:07:26 +0800 Total run time: 10.057316469s. All OK.
```

## Dump

When called as `rrd`, dumps the generated script.

## Silent

When called as `rr` _and_ a console is not detected it only shows errors as structured JSON.
```
{"level":"error","stdout":"ss\n","stderr":"ee\n","time":"2021-07-20T20:16:04+08:00","message":"Output"}
{"level":"error","elapsed":"1.798478ms","time":"2021-07-20T20:16:04+08:00","message":"Something went wrong."}
```

# TUTORIAL

In the source tree, the *TUTORIAL* directory contains a hierarchy that persistently enables IP forwarding through sysctl upon the remote SSH host named *avocado*

First you have to setup SSH public-key authentication for a remote host with hostname `avocado`. It's important that the hostnames match.

Once that is done, you can proceed with the example:

    cd TUTORIAL
    rr avocado sysctl:apply

Steps that `rr` performs:

1. Copy via sftp `.files-avocado/` to `avocado:/`
2. Generates the script:

```
#!/bin/sh
unset IFS
set -efu
PATH=/bin:/sbin:/usr/bin:/usr/sbin
LC_ALL=C
sysctl --system
```

3. Runs the script on host avocado via SSH.

# REFERENCE

### Invocation
    
    rr avocado sysctl:apply --quiet --names
       ^       ^      ^       ^
       host namespace script  arguments

Set the host to `local` or `localhost` for localhost invocations. Mainly used
for "pull" operations.

### Hierarchy

    TOPLEVEL
    ├── .files                          <--- synced to any host
    ├── .files-avocado                  <--- synced to the avocado host
    ├── .lib                            <--- sourced by all scripts
    └── namespace
        ├── .files                      <--- synced to any host when namespace:* is called
        ├── .files-avocado              <--- synced to the avocado host when namespace:* is called
        ├── .lib                        <--- sourced along with namespace:* scripts
        └── script directory
            ├── .files                  <--- synced to any host when namespace:script is called
            ├── .files-avocado          <--- synced to the avocado host when namespace:script is called
            ├── .lib                    <--- sourced along with namespace:script scripts
            └── script                  <--- the actual shell script

### Notes

Tested on Linux and macOS.

Remote host only requires OpenSSH server.

Requires OpenSSH 4.7+ for sftp recursive file transfers. If you can't use 4.7+, a workaround is to use heredocs.  

Scripts should reference `$@` if it wants to use arguments passed.


