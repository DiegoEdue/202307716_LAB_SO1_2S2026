import os
import shutil

# No toques: limpia carpetas de ejecuciones anteriores
for c in ["data", "src", "tests"]:
    shutil.rmtree(c, ignore_errors=True)

# Carpetas que necesitamos crear
carpetas = [
    "data/raw",
    "data/processed",
    "src",
    "tests",
]

existentes = 0
for carpeta in carpetas:
    # Sustituye la línea de abajo por:  os.makedirs(carpeta, exist_ok=True)
    os.makedirs(carpeta, exist_ok=True)
    if os.path.isdir(carpeta):
        existentes = existentes + 1
        print(f"[OK] Creada: {carpeta}")
    else:
        print(f"[ERROR] Falta:  {carpeta}")

print("")
print(f"{existentes} de {len(carpetas)} carpetas listas.")