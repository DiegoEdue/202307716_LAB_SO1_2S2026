# Crear y modificar listas
productos_stock = ["laptop", "monitor", "teclado", "ratón"]

# Añadir elementos
productos_stock.append("webcam")           # Añade al final
productos_stock.insert(0, "servidor")      # Inserta en posición 0

# Eliminar elementos
productos_stock.remove("ratón")            # Elimina por valor
eliminado = productos_stock.pop()          # Elimina y devuelve el último

# Longitud
print(f"Productos en stock: {len(productos_stock)}")

# Ordenar
precios = [299.99, 49.99, 899.00, 129.50, 1499.00]
precios.sort()                  # Ordena la lista original (in-place)
precios_desc = sorted(precios, reverse=True)  # Crea nueva lista ordenada

# Buscar
print("laptop" in productos_stock)  # True o False
print(precios.index(899.00))        # Posición del elemento