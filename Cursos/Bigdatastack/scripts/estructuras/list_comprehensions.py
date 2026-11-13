# Tenemos precios sin IVA
precios_sin_iva = [100, 250, 49.99, 899, 1299.50]

# Calcular precios con IVA (21%) — SIN list comprehension
precios_con_iva = []
for precio in precios_sin_iva:
    precios_con_iva.append(precio * 1.21)

# Lo MISMO con list comprehension (1 línea)
precios_con_iva = [precio * 1.21 for precio in precios_sin_iva]
print(precios_con_iva)

# Filtrar: solo precios mayores a 200€
caros = [p for p in precios_con_iva if p > 200]
print(f"Productos caros (>200€): {caros}")

# Transformar: redondear a 2 decimales
redondeados = [round(p, 2) for p in precios_con_iva]
print(f"Redondeados: {redondeados}")