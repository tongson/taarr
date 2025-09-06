#### 3.1.0

Affluent Handball

* `rr:libs` support (bundled shell script functions)
* `rr:plan` support (for a semblance of the plan-execute flow)
* Upgrade dependencies
* Code upkeep
* Improve tests
* Improve console output

#### 3.0.0

Online Harmonica

* Change imported environment variable prefix from `rr__` to `RR_VAR_`
* Fix handling `.lib` symlinks
* Fix result log
* Code improvements
* Upgrade dependencies

#### 2.7.0

Unguarded Clover

* Set positional parameters early
* Support Ansible `changed=true` detection

#### 2.6.0

Crimson Ointment

* Remove version information from `rrd` output
* Clean up non-terminal STDERR output
* Argument mode priority changed; now longest to shortest

#### 2.5.0

Scrawny Municipal

* Fix include ordering
* Better argument mode detection


#### 2.4.0

Sulk Harmonics

* Support no-namespace scripts `rr no_namespace`
* No longer supports funky `arg:one:two` arguments
* Update dependencies
* Fix panic related to reading from nonexistent script directory

#### 2.3.0

Compacted Monastery

* Prelude (script.pre) and epilogue (script.post) scripts
* Update dependencies
* Major refactor

#### 2.2.1

Unsafe Cupcake

* Fix output during parallel runs
* Print README* file when no `namespace:script` specified
* Include `.lib` in `rrd` (dump) mode

#### 2.2.0

Craving Detonator

* Log SIGINT
* Initial support for Windows
* Update dependencies

#### 2.1.1

Buggy Oven

* Use arguments for the LOG field
* Update dependencies

#### 2.1.0

Degraded Mastiff

* Colorized `rrl` output
* Changed `rrl` headers
* Changed `OP` to `LOG` environment variable
* Removed reading from `OP` file feature
* Added `.ssh/config` and `ssh_config` as valid hosts files
* Changed terminal log line color to cyan
* Various code fixes and improvements

#### 2.0.1

Hypnotic Antennae

* Fix output in terminal mode
* Fix signal handling over SSH
* Update dependencies

#### 2.0.0

Groggy Pauper

* Removed some external dependencies to slash around 500KiB from the executable size
* New extensive testing suite
* New VARS file for includes and variables
* Changed SSH config from `rr.hosts` to `HOSTS`
* Changed log filename from `rr.json` to `LOG`
* Changed `rrl` mode output headers and log format
* Changed string for "repaired" result/status detection to `__REPAIRED__`
* Changed `__REPAIRED__` output detection from STDOUT to STDERR
* Quicker copying over SSH because of one less SSH connection
* Quicker remote runs, removed SSH hostname matching
* More compact `rrl` mode output
* STDOUT now streamed in terminal mode
* Several code improvements and dependency upgrades

#### 1.0.4

* Update dependencies
* Free performance improvement
* Code style
* Improve "make" scripts

#### 1.0.3

* Update dependencies
* Remove spinner
* Use isatty to determine console

#### 1.0.2

* Update dependencies
* Indicate connections in the output
* Fix handling of huge log files
* Better ansi escape sequence for the spinner
* Fix tests

#### 1.0.1

* Update dependencies

#### 1.0.0

* Add plain mode for reusing output

#### 0.19.0

* Remove script level `task` file
  * Top-level `TASK` file is now the task field in the log
  * Can be overridden by env variable `TASK`
* Better looking table separator in report mode
* Shorter start timestamps in report mode

#### 0.18.0

* Added Teleport mode `rrt` & `rro`
* Added report mode `rrl`
* "Idempotence" by checking '+++++repaired+++++' string from STDOUT
* Log to `rr.json` even if not in console mode
* "elapsed" to "duration"
* ID's are just 8 characters now
* Duration truncated at seconds

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
