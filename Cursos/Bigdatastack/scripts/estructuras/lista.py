# Una lista de ventas diarias (en euros)
ventas_semana = [1250.00, 980.50, 1100.75, 1430.20, 890.00, 1560.80, 2100.00]

# Acceder por índice (empiezan en 0)
print(ventas_semana[0])   # 1250.00 — lunes (primer día)
print(ventas_semana[-1])  # 2100.00 — domingo (último día)

# Slicing: obtener un rango
entre_semana = ventas_semana[0:5]  # lunes a viernes
fin_de_semana = ventas_semana[5:]  # sábado y domingo
print(f"Entre semana: {entre_semana}")
print(f"Fin de semana: {fin_de_semana}")