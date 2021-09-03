0.14.0

+ Custom ssh_config (rr.hosts)
+ `app=rr` added to JSON log

0.13.0

+ security issue; omit environment variables from logged code

0.12.0

+ Support `sudo` invocations
+ Interpreter can be set in `shell` file
+ STDOUT and STDERR encoded to base64 in JSON log
+ STDOUT and STDDER also logged as DEBUG entries even without errors
+ Only take the first line from the `task` file
+ A breaking change; exported environment variables must prefix `rr__`
+ Code of script logged along with DEBUG entries


0.11.0

+ Script start marked as Info
+ Start and Finish now properly marked in json log
+ Contents of `task` file used for audit trail
+ Added Bash completion script
+ Replaced ULID with random hexadecimal string