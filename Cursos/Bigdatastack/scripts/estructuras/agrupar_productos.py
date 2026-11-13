productos = [
    {"nombre": "Laptop", "categoria": "electrónica", "precio": 1299},
    {"nombre": "Camiseta", "categoria": "ropa", "precio": 25},
    {"nombre": "Monitor", "categoria": "electrónica", "precio": 399},
    {"nombre": "Pantalón", "categoria": "ropa", "precio": 55},
    {"nombre": "Teclado", "categoria": "electrónica", "precio": 89},
    {"nombre": "Zapatos", "categoria": "ropa", "precio": 120},
]

# Agrupa productos por categoría
por_categoria = {}
for producto in productos:
    cat = producto["categoria"]
    if cat not in por_categoria:
        por_categoria[cat] = []     # <- ya hecha: crea la lista la primera vez
    por_categoria[cat].append(producto)                          # <- añade el producto ENTERO a la lista (no la sustituyas). Pista: .append()

# No toques estas líneas: son las que muestran el resultado
for cat, prods in por_categoria.items():
    nombres = [p["nombre"] for p in prods]
    print(f"{cat}: {nombres}")