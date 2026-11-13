import csv

# Primero creamos un CSV de ejemplo
with open("ventas.csv", "w", encoding="utf-8", newline="") as f:
    writer = csv.writer(f)
    writer.writerow(["fecha", "producto", "cantidad", "precio"])
    writer.writerow(["2024-01-15", "Laptop Pro", 2, 1299.99])
    writer.writerow(["2024-01-15", "Monitor 4K", 1, 449.00])
    writer.writerow(["2024-01-16", "Teclado mecánico", 5, 89.99])
    writer.writerow(["2024-01-16", "Ratón ergonómico", 3, 59.99])

# Leer el CSV como lista de diccionarios (DictReader)
with open("ventas.csv", "r", encoding="utf-8") as f:
    reader = csv.DictReader(f)
    ventas = list(reader)

# Ahora ventas es una lista de diccionarios!
for venta in ventas:
    total = int(venta["cantidad"]) * float(venta["precio"])
    print(f"{venta['fecha']} | {venta['producto']:20s} | {total:.2f}€")

filas = [
    {"producto": "Laptop", "total": "3899.97"},
    {"producto": "Monitor", "total": "1995.00"},
]

with open("salida.csv", "w", encoding="utf-8", newline="") as f:
    writer = csv.DictWriter(f, fieldnames=["producto", "total"])
    writer.writeheader()        # escribe la fila de nombres
    writer.writerows(filas)     # escribe todas las filas de golpe

with open("ventas_excel.csv", "w", encoding="utf-8-sig", newline="") as f:
    f.write("producto;cantidad;precio\n")
    f.write("Laptop;3;1299,99\n")
    f.write("Monitor;2;449,00\n")

with open("ventas_excel.csv", "r", encoding="utf-8-sig", newline="") as f:
    lector = csv.DictReader(f, delimiter=";")
    for fila in lector:
        # el decimal, de coma a punto, antes de convertirlo a número
        precio = float(fila["precio"].replace(",", "."))
        print(f"{fila['producto']}: {precio:.2f}€")