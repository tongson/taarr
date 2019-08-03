local lib = require"lib"
local msg, file = lib.msg, lib.file
local table, io = table, io
local lfs = require"lfs"
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
script[#script+1] = "set -- "..(table.concat(pargs, " "))
local main = file.read_all(task.."/"..command)
if main then
    script[#script+1] = main
else
    msg.fatal"Unable to read main script!"
end
if args.host == "local" then
    local r, o = popen(table.concat(script, "\n"))
    if not r then
        msg.debug("%s", table.concat(o.output, "\n"))
        msg.fatal("%s %s %s", o.exe, o.code, o.status)
    end
else
    -- ssh
end
