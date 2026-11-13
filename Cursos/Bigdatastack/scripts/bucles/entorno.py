import sys, os, platform

# Tu versión de Python
print(sys.version)            # ej: 3.13.1 (main, Dec 3 2025...)
print(sys.version_info)       # (3, 13, 1, 'final', 0) — como tupla

# Tu sistema operativo
print(platform.system())      # 'Windows', 'Darwin' (Mac) o 'Linux'
print(platform.release())     # ej: '10.0.19045' o '23.5.0'

# Desde dónde se ejecuta tu script
print(os.getcwd())            # la carpeta actual de trabajo