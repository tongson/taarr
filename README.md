# rr

# WHAT

shell script manager and runner inspired by rerun[1], bashing[2] and drist[3]

[1] https://github.com/rerun/rerun  
[2] https://github.com/xsc/bashing  
[3] git://bitreich.org/drist  

Your shell script multi-tool. Automate all the things. Can be used for:

1. Runbooks
2. Launchers
3. Deployment
4. Release Engineering
5. Builds
6. Compiles
7. Testing

Checkout the companion [ll](https://github.com/tongson/LadyLua) (Lua interpreter) for anything complex with shell scripts.

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

## Console(verbose)

When called as `rrv` _or_ a console is detected it runs in verbose mode.
```
Thu, 22 Jul 2021 14:01:43 +0800 rr 0.10.0 "Kilowatt Triceps"
Thu, 22 Jul 2021 14:01:43 +0800 Running tmp:test via local...
Thu, 22 Jul 2021 14:01:43 +0800 Running test...
Thu, 22 Jul 2021 14:01:53 +0800 Done. Output:
 local ┌─ stdout
 local │
 local │ 3
 local │ 
 local └─
Thu, 22 Jul 2021 14:01:53 +0800 Total run time: 10.059915355s. All OK.
```

In this mode it also logs to the file `rr.json` in the current working directory.

```
$ cat rr.json
{"level":"debug","id":"01FB6V9FH664R3ACSG60SK2E1M","namespace":"tmp","script":"env","target":"local","time":"2021-07-22T18:13:32+08:00","message":"running"}
{"level":"debug","id":"01FB6V9FH664R3ACSG60SK2E1M","script":"env","time":"2021-07-22T18:13:32+08:00","message":"running"}
{"level":"info","id":"01FB6V9FH664R3ACSG60SK2E1M","result":"success","time":"2021-07-22T18:13:32+08:00","message":"run"}
{"level":"debug","id":"01FB6V9FH664R3ACSG60SK2E1M","elapsed":"101.57144ms","time":"2021-07-22T18:13:32+08:00","message":"success"}
```

## Silent

When called as `rr` _and_ a console is not detected it only shows errors as structured JSON.
```
{"level":"error","stdout":"ss\n","stderr":"ee\n","time":"2021-07-20T20:16:04+08:00","message":"Output"}
{"level":"error","elapsed":"1.798478ms","time":"2021-07-20T20:16:04+08:00","message":"Something went wrong."}
```

## Dump

When called as `rrd`, dumps the generated script. This is mainly for debugging and allows running scripts
generated from a managed namespace without the `rr` executable.

# Audit Trail

The contents of the `task` file inside `namespace/script` will be logged as the `message` field. Without the `task` file
the default is `UNDEFINED`.

```
{"level":"info","id":"01FB8AKE5H64RK4C9G6WS30CSQ","namespace":"tmp","script":"fail","target":"local","time":"2021-07-23T08:00:21+08:00","message":"testing this\n"}
{"level":"debug","id":"01FB8AKE5H64RK4C9G6WS30CSQ","script":"fail","time":"2021-07-23T08:00:21+08:00","message":"running"}
{"level":"error","id":"01FB8AKE5H64RK4C9G6WS30CSQ","error":"exit status 255","time":"2021-07-23T08:00:21+08:00","message":"testing this\n"}
{"level":"debug","id":"01FB8AKE5H64RK4C9G6WS30CSQ","elapsed":"101.687784ms","time":"2021-07-23T08:00:21+08:00","message":"failed"}
{"level":"info","id":"01FB8AMA1F74SK4CSP64RK8DSM","namespace":"tmp","script":"ns","target":"local","time":"2021-07-23T08:00:50+08:00","message":"UNDEFINED"}
{"level":"debug","id":"01FB8AMA1F74SK4CSP64RK8DSM","directory":"tmp/ns/.files","time":"2021-07-23T08:00:50+08:00","message":"copying"}
{"level":"info","id":"01FB8AMA1F74SK4CSP64RK8DSM","result":"success","time":"2021-07-23T08:00:50+08:00","message":"copy"}
{"level":"debug","id":"01FB8AMA1F74SK4CSP64RK8DSM","script":"ns","time":"2021-07-23T08:00:50+08:00","message":"running"}
{"level":"info","id":"01FB8AMA1F74SK4CSP64RK8DSM","result":"success","time":"2021-07-23T08:01:00+08:00","message":"UNDEFINED"}
{"level":"debug","id":"01FB8AMA1F74SK4CSP64RK8DSM","elapsed":"10.180211517s","time":"2021-07-23T08:01:00+08:00","message":"success"}
```

# DOCS & READMES

Any case insensitive file named `readme*` in the namespace and script directories can be shown by invoking `rr` by the
following ways: 

Prints `namespace/readme*`

```
rr namespace
rr namespace/
```

Prints `namespace/script/readme*`

```
rr namespace/script
rr namespace/script/
```

# Environment Variables

Any environment variable prefixed with `rr` are passed to the script.

For example, the `rrUSERNAME=test` environment variable is passed to the script as `USERNAME=test`

```
$ cat tmp/env/script
echo $USERNAME

$ env rrUSERNAME=test rr tmp:env
Thu, 22 Jul 2021 18:13:32 +0800 rr 0.10.0 "Kilowatt Triceps"
Thu, 22 Jul 2021 18:13:32 +0800 Running tmp:env via local...
Thu, 22 Jul 2021 18:13:32 +0800 Running env...
Thu, 22 Jul 2021 18:13:32 +0800 Done. Output:
 local ┌─ stdout
 local │
 local │ test
 local │ 
 local └─
Thu, 22 Jul 2021 18:13:32 +0800 Total run time: 101.648007ms. All OK.
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

    TOPLEVEL                            <--- Directory of namespaces or a project
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


