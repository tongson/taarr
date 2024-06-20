package main

const cVERSION = "2.2.0"
const cCODE = "Craving Detonator"

const cOP = "LOG"
const cINC = "VARS"
const cHOSTS0 = ".ssh/config"
const cHOSTS1 = "ssh_config"
const cHOSTS2 = "HOSTS"
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

const cANSI = "\x1b[0036m%s\x1b[0000m"

const cTARC = "--no-same-owner --no-same-permissions"
const cTARX = "--no-same-owner --no-same-permissions --no-overwrite-dir --no-acls --no-selinux --no-xattrs --touch"

const oJson int = 0
const oTerm int = 1
const oPlain int = 2

const eUNSPECIFIED = "You must specify the `namespace:script`"
