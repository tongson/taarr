package main

const cVERSION = "2.3.0"
const cCODE = "Compacted Monastery"

const cOP = "LOG"
const cINC = "VARS"
const cHOSTS0 = ".ssh/config"
const cHOSTS1 = "ssh_config"
const cHOSTS2 = "HOSTS"
const cLOG = "LOG"
const cREPAIRED = "__REPAIRED__"
const cRUN = "script"
const cPRE = "script.pre"
const cPOST = "script.post"
const cINTERP = "shell"
const cDOC = "readme"
const cTIME = "02 Jan 06 15:04"

const cSTDOUT = " ┌─ stdout"
const cSTDERR = " ┌─ stderr"
const cSTDDBG = " ┌─ debug"
const cFOOTER = " └─"

const cANSI = "\x1b[1G\x1b[0036m%s\x1b[0000m %s"

const cTARC = "--no-same-owner --no-same-permissions"
const cTARX = "--no-same-owner --no-same-permissions --no-overwrite-dir --no-acls --no-selinux --no-xattrs --touch"

const eUNSPECIFIED = "You must specify the `namespace:script`"

// Output
const (
	cJson  = iota
	cTerm  = iota
	cPlain = iota
)

// Call
const (
	cDefault  = iota
	cDump     = iota
	cLog      = iota
	cTeleport = iota
)

// Sudo?
const (
	cNoSudo = iota
	cSudo   = iota
)

// Sudo password?
const (
	cNoSudoPasswd = iota
	cSudoPasswd   = iota
)

const cPmodes = `rr  = local or ssh
rrs = ssh + sudo
rru = ssh + sudo + nopasswd
rrt = teleport
rro = teleport + sudo
rrd = dump
rrv = forced verbose
rrl = report`

