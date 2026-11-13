# Datos de ventas del lunes
ventas_lunes = [150.0, 89.50, 220.0, 45.99, 310.75, 67.00]

# Calculamos métricas básicas
total = sum(ventas_lunes)
num_ventas = len(ventas_lunes)
promedio = total / num_ventas

# Mostramos un mini-reporte
print("=" * 40)
print("  REPORTE DE VENTAS - LUNES")
print("=" * 40)
print(f"  Transacciones: {num_ventas}")
print(f"  Total:         {total:.2f}€")
print(f"  Promedio:      {promedio:.2f}€")
print(f"  Máxima:        {max(ventas_lunes):.2f}€")
print(f"  Mínima:        {min(ventas_lunes):.2f}€")
print("=" * 40)
print("\n¡Python funciona! Tu entorno está listo.")