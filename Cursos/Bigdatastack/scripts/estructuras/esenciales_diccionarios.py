cliente = {
    "nombre": "Carlos López",
    "email": "carlos@empresa.com",
    "plan": "premium",
    "gasto_mensual": 450.00,
}

# Obtener todas las claves y valores
print(list(cliente.keys()))    # ["nombre", "email", "plan", "gasto_mensual"]
print(list(cliente.values()))  # ["Carlos López", "carlos@empresa.com", ...]

# Iterar sobre clave-valor (muy común en datos)
for campo, valor in cliente.items():
    print(f"  {campo}: {valor}")

# Actualizar múltiples campos de golpe
cliente.update({"plan": "enterprise", "gasto_mensual": 890.00})

# Eliminar una clave
del cliente["email"]

# Copiar (cuidado: = NO copia, crea una referencia)
cliente_copia = cliente.copy()