import subprocess
result = subprocess.run(
    ["mysql", "-u", "salfanet_user", "-psalfanetradius123", "salfanet_radius",
     "--batch", "--skip-column-names",
     "-e", "SHOW TABLES;"],
    capture_output=True, text=True
)
print(result.stdout)
print(result.stderr[:200] if result.stderr else "")
