#!/usr/bin/env python3
"""Debug script: check rxPower SNMP data for ONU 1/1/1:1"""
import subprocess

OLT = "136.1.1.100"
COMMUNITY = "public"
BASE = "1.3.6.1.4.1.3902.1012"
PON_IDX = "268501248"  # port 1/1/1

def snmpwalk(oid):
    r = subprocess.run(["snmpwalk", "-v2c", "-c", COMMUNITY, "-Oq", OLT, oid],
                       capture_output=True, text=True, timeout=15)
    return r.stdout.strip().split("\n")

def snmpget(oid):
    r = subprocess.run(["snmpget", "-v2c", "-c", COMMUNITY, "-Oq", OLT, oid],
                       capture_output=True, text=True, timeout=10)
    return r.stdout.strip()

# rxPower OID column 10
rx_oid = f"{BASE}.3.50.12.1.1.10.{PON_IDX}"
# txPower OID column 11
tx_oid = f"{BASE}.3.50.12.1.1.11.{PON_IDX}"

print("=== BulkWalk rxPower for port 268501248 ===")
rx_lines = snmpwalk(rx_oid)
print(f"Total entries: {len(rx_lines)}")
for l in rx_lines[:5]:
    print(" ", l)

print()
print("=== Direct GET: ONU 1 rxPower (.1.1) ===")
print(snmpget(f"{rx_oid}.1.1"))

print()
print("=== Direct GET: ONU 1 txPower (.1.1) ===")
print(snmpget(f"{tx_oid}.1.1"))

print()
# Check if ONU 1 is FIRST in the walk results
print("=== First 2 entries of rxPower walk ===")
for l in rx_lines[:2]:
    print(" ", l)

# Check the regStatus OID for ONU 1
reg_oid = f"{BASE}.3.50.12.1.1.1.{PON_IDX}"
print()
print("=== regStatus walk (col 1) ===")
reg_lines = snmpwalk(reg_oid)
for l in reg_lines[:3]:
    print(" ", l)

print()
print("=== Direct GET regStatus ONU 1 ===")
print(snmpget(f"{reg_oid}.1.1"))
