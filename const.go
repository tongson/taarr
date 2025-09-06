package main

const cVERSION = "3.1.0"
const cCODE = "Affluent Handball"

const cOK = "ok"
const cOP = "LOG"
const cINC = "VARS"
const cHOSTS0 = ".ssh/config"
const cHOSTS1 = "ssh_config"
const cHOSTS2 = "HOSTS"
const cLOG = "LOG"
const cREPAIRED = "__REPAIRED__"
const cCHANGED = "changed=true"
const cRUN = "script"
const cPRE = "script.pre"
const cPOST = "script.post"
const cINTERP = "shell"
const cPLAN = "plan"
const cDOC = "readme"
const cTIME = "02 Jan 06 15:04"
const cVAR = "RR_VAR_"

const cSTDOUT = " ┌── stdout"
const cSTDERR = " ┌── stderr"
const cSTDDBG = " ┌── debug"
const cFOOTER = " └──"
const cPPLAN = " ┌── plan"
const cPPARA = " ┌── parameters"

const cANSI = "\x1b[1G\x1b[0036m%s\x1b[0000m %s"

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
	cPlan     = iota
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

// Override hostname
const (
	cPhaseMain     = iota
	cPhasePrelude  = iota
	cPhaseEpilogue = iota
)

// Arg mode
const (
	cArgNone       = iota
	cArgLocalSolo  = iota
	cArgLocalHier  = iota
	cArgRemoteSolo = iota
	cArgRemoteHier = iota
)

const cTARC = "--no-same-owner --no-same-permissions"
const cTARX = "--no-same-owner --no-same-permissions --no-overwrite-dir --no-acls --no-selinux --no-xattrs --touch"

const eUNSPECIFIED = "You must specify the `namespace:script`"

const (
	cExitUsage         = 64
	cExitNoInput       = 66
	cExitOsErr         = 71
	cExitCantCreate    = 73
	cExitCannotExecute = 126
	cExitNotFound      = 127
	cExitInterrupted   = 130
)

const cPmodes = `
rr:sudo, rru = ssh + sudo + nopasswd
rr:dump, rrd = dump
rr:log, rrl  = changelog
rr:plan      = plan
rr           = local or ssh
rrs          = ssh + sudo
rrt          = teleport
rro          = teleport + sudo
rrv          = forced verbose
`
