temperaturas = [62.3, 65.1, 71.8, 68.4, 76.2, 73.9, 69.5, 64.8, 61.2, 67.7, 72.1, 70.3]
UMBRAL_CRITICO = 75.0

maxima = max(temperaturas)
minima = min(temperaturas)
promedio = sum(temperaturas) / len(temperaturas)
supero_umbral = maxima > UMBRAL_CRITICO

print("MONITOREO TÉRMICO DEL SERVIDOR")
print(f"Máxima: {maxima:.1f}°C")
print(f"Mínima: {minima:.1f}°C")
print(f"Promedio: {promedio:.1f}°C")
print(f"¿Alerta térmica? {supero_umbral}")