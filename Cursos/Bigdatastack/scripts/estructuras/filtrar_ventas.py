ventas_mes = [
    {"cliente": "Ana", "importe": 320.00},
    {"cliente": "Luis", "importe": 890.50},
    {"cliente": "María", "importe": 150.00},
    {"cliente": "Pedro", "importe": 1200.00},
    {"cliente": "Sara", "importe": 45.99},
    {"cliente": "Jorge", "importe": 670.00},
]

# Filtra las ventas de más de 500€ con una list comprehension
# Pista: [v for v in ventas_mes if ...]   (ojo: "más de" es >, no >=)
ventas_vip = [v for v in ventas_mes if v['importe'] > 500]

# Suma el importe de esas ventas VIP
# Pista: sum() necesita NÚMEROS, y cada elemento de ventas_vip es un
#        diccionario. Saca antes el importe: sum(v["importe"] for v in ...)
total_vip = sum(v['importe'] for v in ventas_vip)

# No toques estas líneas: son las que muestran el resultado
print("Clientes VIP (>500€):")
for venta in ventas_vip:
    print(f"  {venta['cliente']}: {venta['importe']:.2f}€")
print("")
print(f"Total VIP: {total_vip:.2f}€")
