local lib = require"lib"
local util, msg, file, fmt = lib.util, lib.msg, lib.file, lib.fmt
local table, io = table, io
local exec = require"exec"
local lfs = require"lfs"
local sf = string.format
local tc = table.concat
local argparse = require "argparse"
local parser = argparse("rr", "run shell scripts locally or remotely over SSH.")
parser:argument"host"
parser:argument"task_command"
parser:handle_options(false)
parser:argument"pargs":args"*"
local args = parser:parse()
local task, command = args.task_command:match("([^:]+):([^:]+)")
local popen = function(str)
    local R = {}
    local pipe = io.popen(str, "r")
    io.flush(pipe)
    R.output = {}
    for ln in pipe:lines() do
        R.output[#R.output + 1] = ln
    end
    local _, status, code = io.close(pipe)
    R.exe = "io.popen"
    R.code = code
    R.status = status
    if code == 0 then
        return code, R
    else
        return nil, R
    end
end
local test = function(m, i)
    local attrib = lfs.attributes(i)
    if attrib and attrib.mode == m then
        return true
    end
end

local script = {}
for l in lfs.dir("lib") do
    local libsh = "lib/"..l
    if test("file", libsh) then
        script[#script+1] = file.read_all(libsh)
    end
end
local tlib = task.."/lib"
if test("directory", tlib) then
    for l in lfs.dir(tlib) do
        local libsh = tlib.."/"..l
        if test("file", libsh) then
            script[#script+1] = file.read_all(libsh)
        end
    end
end
local pargs = args.pargs or ""
script[#script+1] = "set -- "..(tc(pargs, " "))
local main = file.read_all(task.."/"..command)
if main then
    script[#script+1] = main
else
    msg.fatal"Unable to read main script!"
end
if args.host == "local" or args.host == "localhost" then
    local r, o = popen(tc(script, "\n"))
    if not r then
        msg.debug("%s", tc(o.output, "\n"))
        msg.fatal("%s %s %s", o.exe, o.code, o.status)
    end
else
    msg.info(sf("Checking if %s exist", args.host))
    local ssh = exec.ctx"/usr/bin/ssh"
    ssh.env = { LC_ALL="C" }
    local ok, rs = ssh("-a", "-P", "-x", args.host, "uname -n")
    if not ok and (args.host ~= rs.stdout[1]) then
        msg.fatal "Host does not exist."
        fmt.panic "Exiting.\n"
    end
    if test("directory", task.."/files") then
        local sftp = exec.ctx"/usr/bin/sftp"
        sftp.stdin = sf("lcd %s/files\ncd /\nput -rP .\n bye\n", task)
        sftp.env = { LC_ALL="C" }
        sftp.errexit = true
        msg.info(sf("Copying files to '%s'", args.host))
        sftp("-v", "-C", "-b", "/dev/fd/0", args.host)
    end
    ssh.errexit = true
    ssh.stdin = tc(script, "\n")
    msg.info(sf("Running script over '%s'", args.host))
    local t = "./"..(util.random_string(16))
    ssh("-a", "-P", "-x", "-C", args.host)
    msg.ok "Success."
end
