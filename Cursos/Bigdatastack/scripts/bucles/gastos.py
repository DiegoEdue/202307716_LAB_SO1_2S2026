def agregar_gasto(gastos):
    print("Ingrese los datos que se solicitan: ")
    concepto = input("¿Qué compraste? ")
    cantidad = float(input("¿Cuánto costó? "))
    print("Categorías: comida, transporte, ocio, hogar, otros")
    categoria = input("Elige una categoría: ").lower()

    gasto = {
        "concepto": concepto, 
        "cantidad": cantidad, 
        "categoria": categoria
        }

    gastos.append(gasto)
    print(f"[OK] Gasto registrado: {concepto} (Q{cantidad})")

def ver_gastos(gastos):
    if len(gastos) == 0:
        print("No hay ningún gasto")
        return

    total = 0
    for i in range(len(gastos)):
        g = gastos[i]
        total = total + g["cantidad"]
    print(f"  TOTAL: Q{total} ({len(gastos)} gastos)")

def resumen(gastos):
    if len(gastos) == 0:
        print("No hay ningún gasto")
        return

    categorias = {}
    for gasto in gastos:
        cat = gasto['categoria']
        if cat in categorias:
            categorias[cat] = categorias[cat] + gasto['cantidad']
        else:
            categorias[cat] = gasto['cantidad']

    print("="*10 + "RESUMEN POR CATEGORÍA" + "="*10)
    total_general = 0
    for cat in categorias:
        total_cat = categorias[cat]
        total_general = total_general + total_cat
        print(f"  {cat} Q{total_cat}")
    print(f"TOTAL: {total_general}")

def buscar_gastos(gastos):
    print("Buscar por: 1. categoría 2. importe mínimo")
    opcion = input ("Opción: ")

    encontrados = []
    if opcion == "1":
        cat = input("¿Qué categoría desea buscar?: ").lower()
        for gasto in gastos:
            if gasto['categoria'] == cat:
                encontrados.append(gasto)

    elif opcion == "2":
        minimo = float(input("¿Cuál es el mínimo que desea buscar?: "))
        for gasto in gastos:
            if gasto['cantidad'] >= minimo:
                encontrados.append(gasto)

    if len(encontrados) == 0:
        print("No hay ningún gasto con ese criterio")
    else:
        print(f"Encontrados: {len(encontrados)} gastos")
        for g in encontrados:
            print(f"{g['concepto']}: Q{g['cantidad']} {g['categoria']}")

def llenar_lista(gastos):
    gasto = [
        {"concepto": "Papa", "cantidad": 6, "categoria": "comida"},
        {"concepto": "Café", "cantidad": 2.50, "categoria": "comida"},
        {"concepto": "Metro", "cantidad": 1.50, "categoria": "transporte"},
        {"concepto": "Netflix", "cantidad": 12.99, "categoria": "ocio"},
        {"concepto": "Mouse", "cantidad": 75, "categoria": "tecnologia"},
        {"concepto": "Lámpara", "cantidad": 100, "categoria": "hogar"},
    ]
    gastos.extend(gasto)
    print("[OK]: Datos cargados a la lista")

def eliminar_gastos(gastos):
    if len(gastos) == 0:
        print("No hay ningún gasto para eliminar")
        return
    
    print("="*10 + "LISTA DE GASTOS" + "="*10)
    for g in gastos:
        print(f"{g['concepto']}: Q{g['cantidad']} {g['categoria']}")
    print("="*25)
    eliminar = input("Escriba el nombre del producto que desea eliminar: ").lower()

    encontrado = False
    for gasto in gastos[:]:
        if gasto['concepto'].lower() == eliminar:
            gastos.remove(gasto)
            print(f"{gasto['concepto']} eliminado de la lista")
            encontrado = True

    if not encontrado:
        print(f"{eliminar} no fue encontrado en la lista")

def mayor_menor(gastos):
    if len(gastos) == 0:
        print("No hay ningún gasto para analizar")
        return

    cantidades = []
    for g in gastos:
        cantidades.append(g['cantidad'])
    mayor = max(cantidades)
    menor = min(cantidades)

    print("="*10 + "Análisis de mayor y menor" + "="*10)
    print(f"Mayor gasto: Q{mayor}")
    print(f"Menor gasto: Q{menor}")

def mostrar_menu():
    print("=" * 25)
    print("GESTOR DE GASTOS PERSONALES")
    print("=" * 25)
    print("1. Agregar gasto")
    print("2. Ver gastos")
    print("3. Resumen")
    print("4. Buscar gastos")
    print("5. Salir")
    print("6. Llenar lista")
    print("7. Eliminar gasto")
    print("8. Mayor y menor")
    print("=" * 25)

def main():
    gastos = []
    while True:
        mostrar_menu()
        opcion = input("Escoja una opción (1-8): ")
        if opcion == "1":
            agregar_gasto(gastos)
        elif opcion == "2":
            ver_gastos(gastos)
        elif opcion == "3":
            print("3")
            resumen(gastos)
        elif opcion == "4":
            print("4")
            buscar_gastos(gastos)
        elif opcion == "5":
            print("Hasta luego, gracias por usar el sistema.")
            break
        elif opcion == "6":
            llenar_lista(gastos)
        elif opcion == "7":
            eliminar_gastos(gastos)
        elif opcion == "8":
            mayor_menor(gastos)
        else:
            print("Escoja un número dentro del rango.")

if __name__ == "__main__":
    main()