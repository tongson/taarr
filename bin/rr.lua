-- Includes
local lib = require"lib"
local msg, file, fmt, str = lib.msg, lib.file, lib.fmt, lib.str
local table, io, string = table, io, string
local exec = require"exec"
local lfs = require"lfs"
local sf = string.format
local tc = table.concat
local argparse = require "argparse"

-- Arguments
local parser = argparse("rr", "run shell scripts locally or remotely over SSH.")
parser:argument"host"
parser:argument"group_command"
parser:handle_options(false)
parser:argument"pargs":args"*"
local args = parser:parse()
local host = args.host
local group, command = args.group_command:match("([^:]+):([^:]+)")

-- Funcs: "A little copying is better than a little dependency"
local lfs_test = function(m)
    return function(i)
        local attrib = lfs.attributes(i)
        if attrib and attrib.mode == m then
            return true
        end
    end
end
local template = function(s, v) return (string.gsub(s, "%${[%s]-([^}%G]+)[%s]-}", v)) end
local popen = function(s)
    local pipe = io.popen(s, "r")
    io.flush(pipe)
    local output = {}
    for ln in pipe:lines() do
        output[#output + 1] = ln
    end
    local _, status, code = io.close(pipe)
    if code ~= 0 then
        msg.debug(sf("%s", tc(output, "\n")))
        msg.fatal(sf("%s %s %s", "io.open", code, status))
        fmt.panic"Exiting.\n"
    end
end

local lua_parse = function(f)
    local source = file.read_all(f)
    return function(e)
        local chunk, err = loadstring(source)
        if chunk then
            setfenv(chunk, e)
            chunk()
            return true
        else
            local tbl = str.to_table(source)
            local ln = string.match(err, "^.+:([%d]+):%s.*")
            local sp = string.rep(" ", string.len(ln))
            local lerr = string.match(err, "^.+:[%d]+:%s(.*)")
            return fmt.panic("error: %s\n%s |\n%s | %s\n%s |\n", lerr, sp, ln, tbl[tonumber(ln)], sp)
        end
    end
end

-- Main
local isFile = lfs_test"file"
local isDir = lfs_test"directory"

local env = {}
for _, f in ipairs{ "vars", "vars-"..host, group.."/vars", group.."/vars-"..host } do
    if isFile(f) then
        msg.debug(sf("%s found. Parsing.", f))
        local runLua = lua_parse(f)
        runLua(env)
    else
        msg.debug(sf("%s not found. Skipping.", SRC))
    end
end

if not isDir(group) then
    msg.fatal(sf("Unable to find script group '%s'.", group))
    fmt.panic"Exiting.\n"
end
local script = {}
for l in lfs.dir("lib") do
    local libsh = "lib/"..l
    if isFile(libsh) then
        if next(env) then
            script[#script+1] = template(file.read_all(libsh, env))
        else
            script[#script+1] = file.read_all(libsh)
        end
    end
end
local tlib = group.."/lib"
if isDir(tlib) then
    for l in lfs.dir(tlib) do
        local libsh = tlib.."/"..l
        if isFile(libsh) then
            if next(env) then
                script[#script+1] = template(file.read_all(libsh, env))
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
    if next(env) then
        script[#script+1] = template(file.read_all(group.."/"..command), env)
    else
        script[#script+1] = file.read_all(group.."/"..command)
    end
else
    msg.fatal"Unable to read main script!"
    fmt.panic"Exiting.\n"
end
if host == "local" or host == "localhost" then
    local tar = [[#!/bin/sh
        LC_ALL=C
        set -efu
        unset IFS
        PATH=/bin:/usr/bin
        tar -C %s -cpf - . | tar -C / -xpf -
    ]]
    for _, d in ipairs{ "files", "files-local", "files-localhost", group.."/files", group.."/files-local", group.."/files-localhost" } do
        if isDir(d) then popen(sf(tar, d)) end
    end
    popen(tc(script, "\n"))
else
    msg.info(sf("Checking if %s exist", host))
    local ssh = exec.ctx"/usr/bin/ssh"
    ssh.env = { LC_ALL="C" }
    local ok, rs = ssh("-T", "-a", "-P", "-x", host, "uname -n")
    if not ok and (host ~= rs.stdout[1]) then
        msg.fatal "Host does not exist."
        fmt.panic "Exiting.\n"
    end
    local copy = function(shost)
        return function(dir)
            local sftp = exec.ctx"/usr/bin/sftp"
            sftp.stdin = sf("lcd %s\ncd /\nput -rP .\n bye\n", dir)
            sftp.env = { LC_ALL="C" }
            sftp.errexit = true
            msg.debug(sf("Copying %s to '%s'", dir, shost))
            sftp("-C", "-b", "/dev/fd/0", shost)
        end
    end
    local toHost = copy(host)
    for _, d in ipairs{ "files", "files-"..host, group.."/files", group.."/files-"..host } do
        if isDir(d) then toHost(d) end
    end
    ssh.errexit = true
    ssh.stdin = tc(script, "\n")
    msg.debug(sf("Running script over '%s'", args.host))
    ssh("-T", "-a", "-P", "-x", "-C", args.host)
    msg.ok "Success."
end
