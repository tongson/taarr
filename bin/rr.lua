local lib = require"lib"
local msg, file, fmt = lib.msg, lib.file, lib.fmt
local table, io, string = table, io, string
local exec = require"exec"
local lfs = require"lfs"
local sf = string.format
local tc = table.concat
local argparse = require "argparse"
local parser = argparse("rr", "run shell scripts locally or remotely over SSH.")
parser:argument"host"
parser:argument"group_command"
parser:handle_options(false)
parser:argument"pargs":args"*"
local args = parser:parse()
local host = args.host
local group, command = args.group_command:match("([^:]+):([^:]+)")
local template = function(s, v) return (string.gsub(s, "%${[%s]-([^}%G]+)[%s]-}", v)) end
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
local copy = function(shost, dir)
    local sftp = exec.ctx"/usr/bin/sftp"
    sftp.stdin = sf("lcd %s\ncd /\nput -rP .\n bye\n", dir)
    sftp.env = { LC_ALL="C" }
    sftp.errexit = true
    msg.info(sf("Copying files to '%s'", shost))
    sftp("-v", "-C", "-b", "/dev/fd/0", shost)
end

local ENV = {}
if test("file", "rr.lua") then
  local source = file.read_all("rr.lua")
  local chunk, err = loadstring(source)
  if chunk then
    setfenv(chunk, ENV)
    chunk()
  else
    local tbl = {}
    local src = io.open("rr.lua")
    for ln in src:lines() do
      tbl[#tbl + 1] = ln
    end
    local ln = string.match(err, "^.+:([%d]+):%s.*")
    local sp = string.rep(" ", string.len(ln))
    local lerr = string.match(err, "^.+:[%d]+:%s(.*)")
    msg.fatal"Problem parsing rr.lua."
    return fmt.panic("error: %s\n%s |\n%s | %s\n%s |\n", lerr, sp, ln, tbl[tonumber(ln)], sp)
  end
end

local script = {}
for l in lfs.dir("lib") do
    local libsh = "lib/"..l
    if test("file", libsh) then
        if next(ENV) then
            script[#script+1] = template(file.read_all(libsh, ENV))
        else
            script[#script+1] = file.read_all(libsh)
        end
    end
end
local tlib = group.."/lib"
if test("directory", tlib) then
    for l in lfs.dir(tlib) do
        local libsh = tlib.."/"..l
        if test("file", libsh) then
            if next(ENV) then
                script[#script+1] = template(file.read_all(libsh, ENV))
            else
                script[#script+1] = file.read_all(libsh)
            end
        end
    end
end
local pargs = args.pargs or ""
script[#script+1] = "set -- "..(tc(pargs, " "))
local main = file.read_all(group.."/"..command)
if main then
    if next(ENV) then
        script[#script+1] = template(file.read_all(group.."/"..command), ENV)
    else
        script[#script+1] = file.read_all(group.."/"..command)
    end
else
    msg.fatal"Unable to read main script!"
end
if host == "local" or host == "localhost" then
    local r, o = popen(tc(script, "\n"))
    if not r then
        msg.debug("%s", tc(o.output, "\n"))
        msg.fatal("%s %s %s", o.exe, o.code, o.status)
    end
else
    msg.info(sf("Checking if %s exist", host))
    local ssh = exec.ctx"/usr/bin/ssh"
    ssh.env = { LC_ALL="C" }
    local ok, rs = ssh("-T", "-a", "-P", "-x", host, "uname -n")
    if not ok and (host ~= rs.stdout[1]) then
        msg.fatal "Host does not exist."
        fmt.panic "Exiting.\n"
    end
    local dirs = { "files", "files-"..host, group.."/files", group.."/files-"..host }
    for _, d in ipairs(dirs) do
        if test("directory", d) then
            copy(host, d)
        end
    end
    ssh.errexit = true
    ssh.stdin = tc(script, "\n")
    msg.info(sf("Running script over '%s'", args.host))
    ssh("-T", "-a", "-P", "-x", "-C", args.host)
    msg.ok "Success."
end
