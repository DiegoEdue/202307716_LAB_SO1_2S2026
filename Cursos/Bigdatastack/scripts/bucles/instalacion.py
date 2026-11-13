import sys

mayor = sys.version_info.major
menor = sys.version_info.minor
version_corta = str(mayor) + "." + str(menor)
version_ok = float(version_corta) >= 3.10

print(f"Version de Python: {version_corta}")
print(f"Version 3.10 o superior: {version_ok}")