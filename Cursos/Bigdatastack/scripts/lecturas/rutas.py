from pathlib import Path

# pathlib funciona igual en Windows y Mac
ruta_datos = Path("data") / "raw" / "ventas.csv"
print(ruta_datos)  # data/raw/ventas.csv (Mac) o data\raw\ventas.csv (Win)

# Crear directorios si no existen
ruta_datos.parent.mkdir(parents=True, exist_ok=True)

# Verificar si existe
if ruta_datos.exists():
    print(f"Tamaño: {ruta_datos.stat().st_size} bytes")
else:
    print("El archivo no existe aún")

# Listar archivos de un directorio
for archivo in Path("data").glob("*.csv"):
    print(f"Encontrado: {archivo.name}")