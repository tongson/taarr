package main

const cVERSION = "2.0.2"
const cCODE = "\"Degraded Mastiff\""

const cOP = "LOG"
const cINC = "VARS"
const cHOSTS0 = "HOSTS"
const cHOSTS1 = "ssh_config"
const cHOSTS2 = ".ssh/config"
const cLOG = "LOG"
const cREPAIRED = "__REPAIRED__"
const cRUN = "script"
const cINTERP = "shell"
const cDOC = "readme"
const cTIME = "02 Jan 06 15:04"

const cSTDOUT = " ┌─ stdout"
const cSTDERR = " ┌─ stderr"
const cSTDDBG = " ┌─ debug"
const cFOOTER = " └─"

const cANSI = "\n\033[1A\033[K\033[38;2;85;85;85m%s\033[0m"

const cTARC = "--no-same-owner --no-same-permissions"
const cTARX = "--no-same-owner --no-same-permissions --no-overwrite-dir --no-acls --no-selinux --no-xattrs --touch"

const oJson int = 0
const oTerm int = 1
const oPlain int = 2