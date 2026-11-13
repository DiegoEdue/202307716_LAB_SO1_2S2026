# Una "tabla" de ventas como lista de diccionarios
ventas = [
    {"fecha": "2024-01-15", "producto": "Laptop", "importe": 1299.00},
    {"fecha": "2024-01-15", "producto": "Monitor", "importe": 349.99},
    {"fecha": "2024-01-16", "producto": "Teclado", "importe": 79.99},
    {"fecha": "2024-01-16", "producto": "Laptop", "importe": 1299.00},
    {"fecha": "2024-01-17", "producto": "Ratón", "importe": 29.99},
]

# Filtrar: ventas de más de 100€
grandes = [v for v in ventas if v["importe"] > 100]
print(f"Ventas grandes: {len(grandes)}")

# Transformar: extraer solo importes
importes = [v["importe"] for v in ventas]
print(f"Total facturado: {sum(importes):.2f}€")

# Agrupar: ventas por producto
from collections import Counter
productos_vendidos = [v["producto"] for v in ventas]
conteo = Counter(productos_vendidos)
print(f"Ventas por producto: {dict(conteo)}")