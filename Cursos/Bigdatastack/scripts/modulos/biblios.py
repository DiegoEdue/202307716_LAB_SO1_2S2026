from datetime import datetime, timedelta
from collections import Counter, defaultdict

# datetime — fundamental para datos temporales
ahora = datetime.now()
ayer = ahora - timedelta(days=1)
print(f"Hoy: {ahora.strftime('%Y-%m-%d %H:%M')}")
print(f"Ayer: {ayer.strftime('%Y-%m-%d')}")

# Counter — contar ocurrencias (perfecto para análisis)
categorias = ["electrónica", "ropa", "electrónica", "comida", "ropa", "ropa"]
conteo = Counter(categorias)
print(f"Conteo: {conteo}")  # Counter({'ropa': 3, 'electrónica': 2, ...})
print(f"Más común: {conteo.most_common(1)}")  # [('ropa', 3)]

# defaultdict — diccionario con valor por defecto
ventas_por_dia = defaultdict(list)
ventas_por_dia["lunes"].append(100)
ventas_por_dia["lunes"].append(200)
ventas_por_dia["martes"].append(150)
print(dict(ventas_por_dia))

from datetime import datetime

# strptime: de texto a fecha (la "p" es de "parse")
fecha = datetime.strptime("2024-01-15", "%Y-%m-%d")
print(fecha)                          # 2024-01-15 00:00:00

# Restar dos fechas da una duración; .days son los días enteros
hoy = datetime(2024, 6, 1)
diferencia = hoy - fecha
print(f"Han pasado {diferencia.days} días")   # Han pasado 138 días