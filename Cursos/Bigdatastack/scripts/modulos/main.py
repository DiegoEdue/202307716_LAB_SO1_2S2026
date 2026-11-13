# --- archivo: main.py ---
"""Script principal que usa el módulo de cálculos."""

import calculos

# Usar funciones del módulo
total = calculos.calcular_total(3, 299.99)
con_iva = calculos.aplicar_iva(total)
final = calculos.calcular_descuento(con_iva, 10)

print(f"Subtotal: {total:.2f}€")
print(f"Con IVA: {con_iva:.2f}€")
print(f"Con descuento 10%: {final:.2f}€")