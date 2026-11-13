ventas = [
    {"producto": "USB", "precio_unitario": 12.99, "cantidad": 3},
    {"producto": "Teclado", "precio_unitario": 50.00, "cantidad": 2},
    {"producto": "Monitor", "precio_unitario": 349.99, "cantidad": 1},
    {"producto": "Laptop", "precio_unitario": 899.00, "cantidad": 2},
    {"producto": "Cable", "precio_unitario": 8.50, "cantidad": 10},
    {"producto": "Silla", "precio_unitario": 250.00, "cantidad": 2},
]

# Enriquece cada registro
for venta in ventas:
    venta["total"] = venta['precio_unitario'] * venta['cantidad']       # <- precio_unitario por cantidad

    # Clasifica: pequeña (<100) · mediana (100-500) · grande (>500)
    if venta["total"] < 100:
        venta["categoria_importe"] = "pequeña"
    elif venta["total"] <= 500:
        venta["categoria_importe"] = "mediana"
    else:
        venta["categoria_importe"] = "grande"

# No toques estas líneas: son las que muestran el resultado
for v in ventas:
    print(f"{v['producto']}: {v['total']:.2f}€ ({v['categoria_importe']})")