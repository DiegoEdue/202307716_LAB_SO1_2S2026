import json

# Datos que queremos guardar
resumen_diario = {
    "fecha": "2024-01-15",
    "total_ventas": 3057.96,
    "num_transacciones": 4,
    "productos_top": ["Laptop Pro", "Monitor 4K"],
    "meta_alcanzada": True,
}

# Escribir JSON (serializar)
with open("resumen.json", "w", encoding="utf-8") as f:
    json.dump(resumen_diario, f, indent=2, ensure_ascii=False)

print("JSON guardado.")

# Leer JSON (deserializar)
with open("resumen.json", "r", encoding="utf-8") as f:
    datos = json.load(f)

print(f"Fecha: {datos['fecha']}")
print(f"Total: {datos['total_ventas']}€")
print(f"Top: {datos['productos_top']}")