import subprocess
result = subprocess.run(
    ["mysql", "-u", "salfanet_user", "-psalfanetradius123", "salfanet_radius",
     "--batch",
     "-e", "DESCRIBE olt_onu_status; SELECT * FROM olt_onu_status LIMIT 2;"],
    capture_output=True, text=True
)
print(result.stdout[:5000])
print(result.stderr[:200] if result.stderr else "")
