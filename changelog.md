#### 0.17.0

* Local invocation can now use interpreter that take in STDIN
  * Another interpreter besides a posix shell can't take in arguments or use `.files`
  * STDIN works with lua(LadyLua), lua(PUC), python, dash, bash, zsh
* Overhaul Makefile
* Fix local tests

#### 0.16.0

* Better sudo with password handling
* Fix various spinner issues
* Changed spinner: line for copying and dots for runs
* Add spinner when checking SSH hostname
* Disable ssh compression
* Interpreter falls back to "sh" instead of $SHELL environment variable

#### 0.15.0

* Change from sftp to tar for ssh copies
* Support ssh copying with sudo passwords
* Also replace rsync with tar for container runs

#### 0.14.0

* Custom ssh_config (rr.hosts)
* `app=rr` added to JSON log

#### 0.13.0

* security issue: omit environment variables from logged code

#### 0.12.0

* Support `sudo` invocations
* Interpreter can be set in `shell` file
* STDOUT and STDERR encoded to base64 in JSON log
* STDOUT and STDDER also logged as DEBUG entries even without errors
* Only take the first line from the `task` file
* A breaking change; exported environment variables must prefix `rr__`
* Code of script logged along with DEBUG entries

#### 0.11.0

* Script start marked as Info
* Start and Finish now properly marked in json log
* Contents of `task` file used for audit trail
* Added Bash completion script
* Replaced ULID with random hexadecimal string
